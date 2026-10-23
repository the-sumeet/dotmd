package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

func newTestService(t *testing.T, names ...string) *FileService {
	t.Helper()
	dir := t.TempDir()
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return &FileService{dir: dir}
}

func TestDuplicateShortName(t *testing.T) {
	f := newTestService(t, "a.md")
	for _, want := range []string{"a copy.md", "a copy 2.md"} {
		got, err := f.Duplicate("a.md")
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("Duplicate = %q, want %q", got, want)
		}
	}
}

func TestDuplicateLongName(t *testing.T) {
	name := strings.Repeat("a", 247) + ".md" // 250 bytes
	f := newTestService(t, name)
	for _, suffix := range []string{" copy.md", " copy 2.md"} {
		got, err := f.Duplicate(name)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != maxNameBytes || !strings.HasSuffix(got, suffix) {
			t.Errorf("Duplicate = %q (%d bytes), want %d bytes ending in %q", got, len(got), maxNameBytes, suffix)
		}
		if _, err := os.Stat(filepath.Join(f.dir, got)); err != nil {
			t.Error(err)
		}
	}
}

func TestDuplicateLongMultibyteName(t *testing.T) {
	name := strings.Repeat("日", 82) + ".md" // 249 bytes
	f := newTestService(t, name)
	got, err := f.Duplicate(name)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) > maxNameBytes || !utf8.ValidString(got) || !strings.HasSuffix(got, " copy.md") {
		t.Errorf("Duplicate = %q (%d bytes), want valid UTF-8 within %d bytes ending in \" copy.md\"", got, len(got), maxNameBytes)
	}
}

func TestDuplicateLongFolderName(t *testing.T) {
	f := newTestService(t)
	name := strings.Repeat("d", 255)
	if err := os.Mkdir(filepath.Join(f.dir, name), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.dir, name, "n.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := f.Duplicate(name)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != maxNameBytes || !strings.HasSuffix(got, " copy") {
		t.Errorf("Duplicate = %q (%d bytes), want %d bytes ending in \" copy\"", got, len(got), maxNameBytes)
	}
	if _, err := os.Stat(filepath.Join(f.dir, got, "n.md")); err != nil {
		t.Error(err)
	}
}

func TestCreateFile(t *testing.T) {
	f := newTestService(t)
	if err := os.Mkdir(filepath.Join(f.dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ dir, name, want string }{
		{"", "a", "a.md"},
		{"", "b.txt", "b.txt"},
		{"sub", "c", "sub/c.md"},
	} {
		got, err := f.CreateFile(tc.dir, tc.name)
		if err != nil {
			t.Fatalf("CreateFile(%q, %q): %v", tc.dir, tc.name, err)
		}
		if got != tc.want {
			t.Errorf("CreateFile(%q, %q) = %q, want %q", tc.dir, tc.name, got, tc.want)
		}
		info, err := os.Stat(filepath.Join(f.dir, filepath.FromSlash(got)))
		if err != nil || info.IsDir() {
			t.Fatalf("%q should be a file (err %v)", got, err)
		}
		if info.Mode().Perm()&0o111 != 0 {
			t.Errorf("%q is executable (%v)", got, info.Mode().Perm())
		}
	}
}

func TestCreateTrimsName(t *testing.T) {
	f := newTestService(t)
	got, err := f.CreateFile("", "  a  ")
	if err != nil || got != "a.md" {
		t.Errorf("CreateFile = %q, %v, want a.md", got, err)
	}
}

func TestCreateFolder(t *testing.T) {
	f := newTestService(t)
	got, err := f.CreateFolder("", "notes")
	if err != nil || got != "notes" {
		t.Fatalf("CreateFolder = %q, %v", got, err)
	}
	got, err = f.CreateFolder("notes", "2024")
	if err != nil || got != "notes/2024" {
		t.Fatalf("CreateFolder = %q, %v", got, err)
	}
	if info, err := os.Stat(filepath.Join(f.dir, "notes", "2024")); err != nil || !info.IsDir() {
		t.Errorf("notes/2024 should be a folder (err %v)", err)
	}
}

func TestCreateKeepsExisting(t *testing.T) {
	f := newTestService(t, "a.md")
	if _, err := f.CreateFile("", "a"); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Errorf("CreateFile over existing file: err = %v, want already exists", err)
	}
	if data, _ := os.ReadFile(filepath.Join(f.dir, "a.md")); string(data) != "x" {
		t.Errorf("existing file was modified: %q", data)
	}
	if _, err := f.CreateFolder("", "a.md"); err == nil {
		t.Error("CreateFolder over existing file: want error")
	}
}

func TestCreateRejectsBadInput(t *testing.T) {
	f := newTestService(t, "file.md")
	long := strings.Repeat("a", maxNameBytes) // ".md" is added, so this ends up too long
	for _, tc := range []struct{ dir, name string }{
		{"", ""},
		{"", "  "},
		{"", "a/b"},
		{"", `a\b`},
		{"", ".hidden"},
		{"", ".."},
		{"", long},
		{"missing", "a"},
		{"file.md", "a"},
		{"..", "a"},
		{"../outside", "a"},
	} {
		if got, err := f.CreateFile(tc.dir, tc.name); err == nil {
			t.Errorf("CreateFile(%q, %q) = %q, want error", tc.dir, tc.name, got)
		}
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(f.dir), "outside")); err == nil {
		t.Error("created something outside the data directory")
	}
}
