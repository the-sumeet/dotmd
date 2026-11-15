package notes

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRename(t *testing.T) {
	s := newTestService(t, "Plan.md", "Old.markdown", "Taken.md", "Dir/a.md", "Dir/assets/pic.png", "x.png")
	cases := []struct{ path, name, want string }{
		{"Plan.md", "Roadmap", "Roadmap.md"},
		{"Old.markdown", " New ", "New.markdown"},
		{"Roadmap.md", "Roadmap.markdown", "Roadmap.markdown"},
		{"x.png", "photo", "photo.png"},
		{"photo.png", "photo.PNG", "photo.PNG"},
		{"Dir", "Projects", "Projects"},
		{"Taken.md", "Taken", "Taken.md"}, // unchanged
		{"Taken.md", "taken", "taken.md"}, // case only
	}
	for _, c := range cases {
		got, err := s.Rename(c.path, c.name)
		if err != nil {
			t.Errorf("Rename(%q, %q): %v", c.path, c.name, err)
			continue
		}
		if got != c.want {
			t.Errorf("Rename(%q, %q) = %q, want %q", c.path, c.name, got, c.want)
		}
	}
	// A folder's contents move with it.
	for _, p := range []string{"Projects/a.md", "Projects/assets/pic.png"} {
		if _, err := os.Stat(filepath.Join(s.Root(), filepath.FromSlash(p))); err != nil {
			t.Errorf("%s missing after folder rename: %v", p, err)
		}
	}
}

func TestRenameRejects(t *testing.T) {
	s := newTestService(t, "a.md", "b.md", "Dir/c.md", "notes.txt", ".hidden/x.md")
	if err := os.Symlink("a.md", filepath.Join(s.Root(), "link.md")); err != nil {
		t.Fatal(err)
	}
	cases := []struct{ path, name string }{
		{"a.md", ""},
		{"a.md", "  "},
		{"a.md", ".hidden"},
		{"a.md", "x/y"},
		{"a.md", `x\y`},
		{"a.md", "b"},   // taken
		{"Dir", "a.md"}, // folder onto a note
		{"missing.md", "z"},
		{"notes.txt", "z"},
		{".hidden", "shown"},
		{".hidden/x.md", "z"},
		{"link.md", "z"},
		{"", "z"},
		{".", "z"},
		{"../a.md", "z"},
	}
	for _, c := range cases {
		if got, err := s.Rename(c.path, c.name); err == nil {
			t.Errorf("Rename(%q, %q) = %q, want error", c.path, c.name, got)
		}
	}
}

// A hard link is the same file as the note, but renaming onto it must fail
// rather than (as rename(2) does) succeed without doing anything.
func TestRenameOntoHardLink(t *testing.T) {
	s := newTestService(t, "a.md")
	if err := os.Link(filepath.Join(s.Root(), "a.md"), filepath.Join(s.Root(), "b.md")); err != nil {
		t.Skip("no hard links here:", err)
	}
	if got, err := s.Rename("a.md", "b"); err != errExists {
		t.Errorf("Rename onto a hard link = %q, %v; want %v", got, err, errExists)
	}
}

func TestDelete(t *testing.T) {
	var trashed []string
	moveToTrash = func(abs string) error {
		trashed = append(trashed, abs)
		return os.RemoveAll(abs)
	}
	t.Cleanup(func() { moveToTrash = trash })

	s := newTestService(t, "n.md", "Dir/a.md", "pic.png", "notes.txt", ".git/x.md",
		"Sub/x.md", "Sub/assets/A.JPG", "Keep/assets/1.png", "Keep/assets/2.png")
	for _, p := range []string{"notes.txt", ".git", ".git/x.md", "", ".", "..", "../n.md", "missing.md"} {
		if err := s.Delete(p); err == nil {
			t.Errorf("Delete(%q) succeeded, want error", p)
		}
	}
	for _, p := range []string{"n.md", "Dir", "pic.png", "Sub/assets/A.JPG", "Keep/assets/1.png"} {
		if err := s.Delete(p); err != nil {
			t.Errorf("Delete(%q): %v", p, err)
		}
	}
	root := s.Root()
	want := []string{
		filepath.Join(root, "n.md"),
		filepath.Join(root, "Dir"),
		filepath.Join(root, "pic.png"),
		filepath.Join(root, "Sub", "assets", "A.JPG"),
		filepath.Join(root, "Keep", "assets", "1.png"),
	}
	if !reflect.DeepEqual(trashed, want) {
		t.Errorf("trashed %v, want %v", trashed, want)
	}
	// An emptied assets folder goes too; one still holding images stays.
	if _, err := os.Stat(filepath.Join(root, "Sub", "assets")); !os.IsNotExist(err) {
		t.Errorf("Sub/assets still exists (err = %v)", err)
	}
	for _, p := range []string{"Sub/x.md", "Keep/assets/2.png"} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(p))); err != nil {
			t.Errorf("%s removed: %v", p, err)
		}
	}
}
