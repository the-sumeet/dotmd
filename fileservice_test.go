package main

import (
	"net/http"
	"net/http/httptest"
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

func TestRenameMovesIntoFolder(t *testing.T) {
	f := newTestService(t, "a.md")
	if err := os.Mkdir(filepath.Join(f.dir, "dir"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := f.Rename("a.md", "dir/a.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(f.dir, "dir", "a.md")); err != nil {
		t.Error(err)
	}
	if _, err := os.Stat(filepath.Join(f.dir, "a.md")); !os.IsNotExist(err) {
		t.Errorf("source still exists: %v", err)
	}
}

func TestRenameKeepsExisting(t *testing.T) {
	f := newTestService(t, "a.md", "b.md")
	if err := f.Rename("a.md", "b.md"); err == nil {
		t.Error("Rename onto an existing file succeeded")
	}
	if _, err := os.Stat(filepath.Join(f.dir, "a.md")); err != nil {
		t.Error(err)
	}
}

func TestRenameCaseOnly(t *testing.T) {
	f := newTestService(t, "a.md")
	if err := f.Rename("a.md", "A.md"); err != nil {
		t.Fatal(err)
	}
}

func TestRenameRejectsBadMoves(t *testing.T) {
	f := newTestService(t, "a.md")
	if err := os.MkdirAll(filepath.Join(f.dir, "dir", "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, from, to string }{
		{"folder into itself", "dir", "dir/dir"},
		{"folder into descendant", "dir", "dir/sub/dir"},
		{"missing destination folder", "a.md", "nope/a.md"},
		{"file as destination folder", "dir", "a.md/dir"},
		{"outside data dir", "a.md", "../a.md"},
	} {
		if err := f.Rename(tc.from, tc.to); err == nil {
			t.Errorf("%s: Rename(%q, %q) succeeded", tc.name, tc.from, tc.to)
		}
	}
}

func TestRevealRejectsBadInput(t *testing.T) {
	f := newTestService(t, "a.md")
	for _, rel := range []string{"", ".", "..", "../outside", "missing.md"} {
		if err := f.Reveal(rel); err == nil {
			t.Errorf("Reveal(%q) succeeded, want error", rel)
		}
	}
}

func TestSaveImage(t *testing.T) {
	f := newTestService(t)
	if err := os.Mkdir(filepath.Join(f.dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	first, err := f.SaveImage("sub", ".PNG", []byte("one"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(first, "sub/image-") || !strings.HasSuffix(first, ".png") {
		t.Errorf("SaveImage = %q, want sub/image-….png", first)
	}
	// Saving again in the same second must not overwrite.
	second, err := f.SaveImage("sub", ".png", []byte("two"))
	if err != nil {
		t.Fatal(err)
	}
	if second == first {
		t.Fatalf("second save reused %q", first)
	}
	for path, want := range map[string]string{first: "one", second: "two"} {
		got, err := os.ReadFile(filepath.Join(f.dir, filepath.FromSlash(path)))
		if err != nil || string(got) != want {
			t.Errorf("%q = %q (err %v), want %q", path, got, err, want)
		}
	}
	root, err := f.SaveImage("", ".jpg", []byte("x"))
	if err != nil || strings.Contains(root, "/") {
		t.Errorf("SaveImage in the root = %q (err %v)", root, err)
	}
}

func TestSaveImageRejectsHugeImage(t *testing.T) {
	f := newTestService(t)
	if got, err := f.SaveImage("", ".png", make([]byte, maxImageBytes+1)); err == nil {
		t.Errorf("SaveImage = %q, want an error", got)
	}
}

func TestSaveImageRejectsBadInput(t *testing.T) {
	f := newTestService(t, "a.md")
	for _, tc := range []struct{ dir, ext string }{
		{"", ".exe"},
		{"", ""},
		{"..", ".png"},
		{"../x", ".png"},
		{"a.md", ".png"},
		{"missing", ".png"},
	} {
		if got, err := f.SaveImage(tc.dir, tc.ext, []byte("x")); err == nil {
			t.Errorf("SaveImage(%q, %q) = %q, want an error", tc.dir, tc.ext, got)
		}
	}
}

func TestHandler(t *testing.T) {
	f := newTestService(t, "pic.png", "notes.md", ".secret")
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "out.png"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(f.dir, "link")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(f.dir, "dir.png"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(f.dir, ".hidden"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.dir, ".hidden", "a.png"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string]int{
		"/files/pic.png":       http.StatusOK,
		"/files/.secret":       http.StatusNotFound,
		"/files/.hidden/a.png": http.StatusNotFound,
		"/files/nope.png":      http.StatusNotFound,
		"/files/notes.md":      http.StatusNotFound, // not an image
		"/files/":              http.StatusNotFound, // no folder listings
		"/files/dir.png":       http.StatusNotFound,
		"/files/link/out.png":  http.StatusNotFound, // symlink leaves the data directory
	} {
		rec := httptest.NewRecorder()
		f.handler().ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		if rec.Code != want {
			t.Errorf("GET %s = %d, want %d", path, rec.Code, want)
		}
		if want == http.StatusOK && rec.Header().Get("Cache-Control") != "no-cache" {
			t.Errorf("GET %s: Cache-Control = %q, want no-cache", path, rec.Header().Get("Cache-Control"))
		}
	}
}
