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
