package notes

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestResolve(t *testing.T) {
	s := &NotesService{root: t.TempDir()}
	cases := []struct {
		path string
		ok   bool
	}{
		{"note.md", true},
		{"a/b/note.md", true},
		{"a/../note.md", true},
		{"", false},
		{".", false},
		{"..", false},
		{"../note.md", false},
		{"a/../../note.md", false},
		{"/etc/passwd", false},
	}
	for _, c := range cases {
		_, err := s.resolve(c.path)
		if (err == nil) != c.ok {
			t.Errorf("resolve(%q): err = %v, want ok = %v", c.path, err, c.ok)
		}
	}
}

func TestTree(t *testing.T) {
	root := t.TempDir()
	for _, f := range []string{
		"b.md", "A.md", "notes.txt", ".hidden.md", "z/inner.markdown", "z/skip.png", ".git/x.md", "empty/",
	} {
		p := filepath.Join(root, f)
		if f[len(f)-1] == '/' {
			if err := os.MkdirAll(p, 0o755); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	got, err := (&NotesService{root: root}).Tree()
	if err != nil {
		t.Fatal(err)
	}
	want := []Node{
		{Name: "empty", Path: "empty", IsDir: true, Children: []Node{}},
		{Name: "z", Path: "z", IsDir: true, Children: []Node{
			{Name: "inner.markdown", Path: "z/inner.markdown", Children: []Node{}},
		}},
		{Name: "A.md", Path: "A.md", Children: []Node{}},
		{Name: "b.md", Path: "b.md", Children: []Node{}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Tree() =\n%+v\nwant\n%+v", got, want)
	}
}

func TestWriteRead(t *testing.T) {
	s := &NotesService{root: t.TempDir()}
	if err := s.Write("sub/n.md", "# hi"); err != nil {
		t.Fatal(err)
	}
	got, err := s.Read("sub/n.md")
	if err != nil || got != "# hi" {
		t.Fatalf("Read = %q, %v", got, err)
	}
	if err := s.Write("x.txt", ""); err == nil {
		t.Error("Write(non-markdown) succeeded, want error")
	}
	entries, _ := os.ReadDir(filepath.Join(s.root, "sub"))
	if len(entries) != 1 {
		t.Errorf("temp file left behind: %v", entries)
	}
}

func TestCreate(t *testing.T) {
	s := &NotesService{root: t.TempDir()}

	got, err := s.CreateNote(" Projects/idea ")
	if err != nil || got != "Projects/idea.md" {
		t.Fatalf("CreateNote = %q, %v", got, err)
	}
	if _, err := s.CreateNote("Projects/idea.md"); err == nil {
		t.Error("CreateNote(existing) succeeded, want error")
	}
	if got, err := s.CreateNote("todo.markdown"); err != nil || got != "todo.markdown" {
		t.Errorf("CreateNote(todo.markdown) = %q, %v", got, err)
	}

	if got, err := s.CreateFolder("Archive/2026"); err != nil || got != "Archive/2026" {
		t.Errorf("CreateFolder = %q, %v", got, err)
	}
	if _, err := s.CreateFolder("Archive"); err == nil {
		t.Error("CreateFolder(existing) succeeded, want error")
	}

	// Spaces around segments are trimmed rather than kept in names.
	if got, err := s.CreateFolder(" Archive /  2027 "); err != nil || got != "Archive/2027" {
		t.Errorf("CreateFolder(spaced) = %q, %v", got, err)
	}
	if got, err := s.CreateNote("Archive / plan "); err != nil || got != "Archive/plan.md" {
		t.Errorf("CreateNote(spaced) = %q, %v", got, err)
	}

	for _, bad := range []string{"", "  ", ".hidden", " .x", "a/ .b", "a/.b", "a//b", "../x", "/abs"} {
		if _, err := s.CreateNote(bad); err == nil {
			t.Errorf("CreateNote(%q) succeeded, want error", bad)
		}
		if _, err := s.CreateFolder(bad); err == nil {
			t.Errorf("CreateFolder(%q) succeeded, want error", bad)
		}
	}
}

// An unreadable subfolder must not hide the rest of the notes.
func TestUnreadableSubfolder(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can read everything")
	}
	root := t.TempDir()
	locked := filepath.Join(root, "locked")
	for _, p := range []string{"ok.md", "locked/x.md"} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, p)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, p), []byte("---\ntags: [a]\n---\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(locked, 0o755) })

	s := &NotesService{root: root}
	tree, err := s.Tree()
	if err != nil {
		t.Fatalf("Tree: %v", err)
	}
	want := []Node{
		{Name: "locked", Path: "locked", IsDir: true, Children: []Node{}},
		{Name: "ok.md", Path: "ok.md", Children: []Node{}},
	}
	if !reflect.DeepEqual(tree, want) {
		t.Errorf("Tree() = %+v, want %+v", tree, want)
	}
	tags, err := s.Tags()
	if err != nil {
		t.Fatalf("Tags: %v", err)
	}
	if len(tags) != 1 || !reflect.DeepEqual(tags[0].Notes, []string{"ok.md"}) {
		t.Errorf("Tags() = %+v, want only ok.md tagged", tags)
	}
}
