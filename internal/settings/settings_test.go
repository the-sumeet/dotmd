package settings

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// fakeNotes stands in for the notes service, checking folders the same way.
type fakeNotes struct {
	root string
}

func (f *fakeNotes) Root() string { return f.root }

func (f *fakeNotes) SetRoot(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("not a folder")
	}
	f.root = path
	return nil
}

func TestLoadDefaults(t *testing.T) {
	dir := t.TempDir()
	cases := map[string]string{
		"missing": "",
		"corrupt": "{not json",
		"empty":   "",
	}
	for name, content := range cases {
		path := filepath.Join(dir, name+".json")
		if name != "missing" {
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		if got := Load(path); got != Defaults() {
			t.Errorf("%s: Load = %+v, want defaults %+v", name, got, Defaults())
		}
	}
}

func TestLoadUnknownTheme(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{"theme":"neon","wideLayout":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	got := Load(path)
	if got.Theme != ThemeSystem || !got.WideLayout {
		t.Errorf("Load = %+v, want theme system and wideLayout kept", got)
	}
}

func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "settings.json") // folder is created
	notesDir := t.TempDir()
	notes := &fakeNotes{root: dir}
	s := New(path, Defaults(), notes)

	if err := s.SetTheme(ThemeDark); err != nil {
		t.Fatal(err)
	}
	if err := s.SetWideLayout(true); err != nil {
		t.Fatal(err)
	}
	st, err := s.setNotesFolder(notesDir)
	if err != nil {
		t.Fatal(err)
	}

	want := Settings{NotesFolder: notesDir, Theme: ThemeDark, WideLayout: true}
	if got := Load(path); got != want {
		t.Errorf("Load after saving = %+v, want %+v", got, want)
	}
	if *st != s.Get() {
		t.Errorf("setNotesFolder state = %+v, Get = %+v", *st, s.Get())
	}
	if st.NotesFolder != notesDir || st.Theme != ThemeDark || !st.WideLayout || st.MissingFolder != "" {
		t.Errorf("state = %+v", *st)
	}
	if notes.root != notesDir {
		t.Errorf("notes root = %q, want %q", notes.root, notesDir)
	}
}

func TestStateFolders(t *testing.T) {
	def, other := t.TempDir(), t.TempDir()
	path := filepath.Join(t.TempDir(), "settings.json")

	// Notes in the default folder.
	s := New(path, Defaults(), &fakeNotes{root: def})
	s.defaultRoot = def
	if st := s.Get(); !st.DefaultFolder || st.MissingFolder != "" {
		t.Errorf("default folder: state = %+v", st)
	}

	// The saved folder was missing at startup, so the default is in use.
	saved := Settings{NotesFolder: filepath.Join(other, "gone"), Theme: ThemeSystem}
	s = New(path, saved, &fakeNotes{root: def})
	s.defaultRoot = def
	if st := s.Get(); !st.DefaultFolder || st.MissingFolder != saved.NotesFolder {
		t.Errorf("missing folder: state = %+v", st)
	}

	// The saved folder is in use.
	saved.NotesFolder = other
	s = New(path, saved, &fakeNotes{root: other})
	s.defaultRoot = def
	if st := s.Get(); st.DefaultFolder || st.MissingFolder != "" {
		t.Errorf("chosen folder: state = %+v", st)
	}
}

func TestSetThemeRejectsUnknown(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	s := New(path, Defaults(), &fakeNotes{})
	if err := s.SetTheme("neon"); err == nil {
		t.Error("SetTheme(neon): want error")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("a rejected theme was saved")
	}
}

func TestSetNotesFolderErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	file := filepath.Join(dir, "file.md")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	notes := &fakeNotes{root: dir}
	s := New(path, Defaults(), notes)

	for _, p := range []string{filepath.Join(dir, "missing"), file} {
		if _, err := s.setNotesFolder(p); err == nil {
			t.Errorf("setNotesFolder(%q): want error", p)
		}
	}
	if notes.root != dir {
		t.Errorf("root = %q after refused switches, want %q", notes.root, dir)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("a refused folder was saved")
	}
}

func TestSetNotesFolderSaveFails(t *testing.T) {
	// A file where the config folder should be makes saving fail.
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	notes := &fakeNotes{root: dir}
	s := New(filepath.Join(blocker, "settings.json"), Defaults(), notes)
	if _, err := s.setNotesFolder(t.TempDir()); err == nil {
		t.Fatal("setNotesFolder: want error when the file can't be written")
	}
	if notes.root != dir {
		t.Errorf("root = %q, want it switched back to %q", notes.root, dir)
	}
}

func TestFailedSaveKeepsSetting(t *testing.T) {
	// A file where the config folder should be makes every save fail.
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	s := New(filepath.Join(blocker, "settings.json"), Defaults(), &fakeNotes{})
	if err := s.SetWideLayout(true); err == nil {
		t.Fatal("SetWideLayout: want error when the file can't be written")
	}
	if s.Get().WideLayout {
		t.Error("WideLayout changed although saving failed")
	}
}
