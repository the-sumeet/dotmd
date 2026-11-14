package notes

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func pngBytes(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func newTestService(t *testing.T, files ...string) *NotesService {
	t.Helper()
	root := t.TempDir()
	for _, f := range files {
		p := filepath.Join(root, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return &NotesService{root: root}
}

var assetHref = regexp.MustCompile(`^assets/my-plan-\d{8}-\d{6}(-\d+)?\.png$`)

func TestSaveImage(t *testing.T) {
	s := newTestService(t, "Projects/My Plan.md")
	data := base64.StdEncoding.EncodeToString(pngBytes(t))

	first, err := s.SaveImage("Projects/My Plan.md", "shot.PNG", data)
	if err != nil {
		t.Fatal(err)
	}
	if !assetHref.MatchString(first) {
		t.Fatalf("href = %q, want assets/my-plan-<time>.png", first)
	}
	if _, err := os.Stat(filepath.Join(s.Root(), "Projects", filepath.FromSlash(first))); err != nil {
		t.Fatalf("saved file missing: %v", err)
	}

	// Same second: the name gets a counter instead of overwriting.
	second, err := s.SaveImage("Projects/My Plan.md", "", data)
	if err != nil {
		t.Fatal(err)
	}
	if second == first || !assetHref.MatchString(second) {
		t.Fatalf("second href = %q (first %q)", second, first)
	}
}

func TestSaveImageRejects(t *testing.T) {
	s := newTestService(t, "a.md")
	text := base64.StdEncoding.EncodeToString([]byte("hello, not an image"))
	svg := base64.StdEncoding.EncodeToString([]byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`))
	png := base64.StdEncoding.EncodeToString(pngBytes(t))
	big := base64.StdEncoding.EncodeToString(make([]byte, maxImageBytes+1))
	cases := []struct {
		name, note, file, data string
		ok                     bool
	}{
		{"text", "a.md", "a.png", text, false},
		{"bad base64", "a.md", "a.png", "%%%", false},
		{"too large", "a.md", "a.png", big, false},
		{"not a note", "a.txt", "a.png", png, false},
		{"escaping note", "../a.md", "a.png", png, false},
		{"svg by name", "a.md", "x.svg", svg, true},
		{"svg without name", "a.md", "", svg, false},
	}
	for _, c := range cases {
		_, err := s.SaveImage(c.note, c.file, c.data)
		if (err == nil) != c.ok {
			t.Errorf("%s: err = %v, want ok = %v", c.name, err, c.ok)
		}
	}
}

func TestDownloadImage(t *testing.T) {
	img := pngBytes(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cat.png":
			w.Header().Set("Content-Type", "image/png")
			w.Write(img)
		case "/page":
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte("<html></html>"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	s := newTestService(t, "My Plan.md")

	href, err := s.DownloadImage("My Plan.md", srv.URL+"/cat.png")
	if err != nil {
		t.Fatal(err)
	}
	if !assetHref.MatchString(href) {
		t.Fatalf("href = %q", href)
	}
	for _, u := range []string{srv.URL + "/page", srv.URL + "/missing.png", "file:///etc/passwd", "ftp://x/a.png"} {
		if _, err := s.DownloadImage("My Plan.md", u); err == nil {
			t.Errorf("DownloadImage(%q) succeeded, want error", u)
		}
	}
}

func TestImagesAndDelete(t *testing.T) {
	s := newTestService(t, "n.md", "assets/b.png", "Sub/assets/A.JPG", ".hidden/c.png", "d.txt", "Sub/x.md")
	got, err := s.Images()
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, img := range got {
		paths = append(paths, img.Path)
	}
	if want := "assets/b.png,Sub/assets/A.JPG"; strings.Join(paths, ",") != want {
		t.Fatalf("Images = %v, want %s", paths, want)
	}

	for _, p := range []string{"n.md", "../x.png", "d.txt", "missing.png"} {
		if err := s.DeleteImage(p); err == nil {
			t.Errorf("DeleteImage(%q) succeeded, want error", p)
		}
	}
	if err := s.DeleteImage("Sub/assets/A.JPG"); err != nil {
		t.Fatal(err)
	}
	// The emptied assets folder goes too; the note's folder stays.
	if _, err := os.Stat(filepath.Join(s.Root(), "Sub", "assets")); !os.IsNotExist(err) {
		t.Errorf("Sub/assets still exists (err = %v)", err)
	}
	if _, err := os.Stat(filepath.Join(s.Root(), "Sub", "x.md")); err != nil {
		t.Errorf("note removed: %v", err)
	}
}

func TestFileMiddleware(t *testing.T) {
	s := newTestService(t, "n.md")
	if err := os.MkdirAll(filepath.Join(s.Root(), "My Notes", "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	img := pngBytes(t)
	if err := os.WriteFile(filepath.Join(s.Root(), "My Notes", "assets", "a b.png"), img, 0o644); err != nil {
		t.Fatal(err)
	}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	h := FileMiddleware(s)(next)
	get := func(target string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
		return rec
	}

	rec := get("/notes-file/My%20Notes/assets/a%20b.png")
	if rec.Code != http.StatusOK || !bytes.Equal(rec.Body.Bytes(), img) {
		t.Fatalf("image: code %d, %d bytes", rec.Code, rec.Body.Len())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/png" {
		t.Errorf("Content-Type = %q", ct)
	}
	for _, target := range []string{"/notes-file/n.md", "/notes-file/../x.png", "/notes-file/missing.png"} {
		if code := get(target).Code; code != http.StatusNotFound {
			t.Errorf("%s: code %d, want 404", target, code)
		}
	}
	if code := get("/index.html").Code; code != http.StatusTeapot {
		t.Errorf("other path: code %d, want it passed on", code)
	}
}

func TestTreeHidesEmptyAssets(t *testing.T) {
	s := newTestService(t, "a.md", "assets/x.png", "Sub/assets/notes.md", "Mine/assets/notes.txt")
	// A user-made, still-empty "assets" folder.
	if _, err := s.CreateFolder("Empty/assets"); err != nil {
		t.Fatal(err)
	}
	nodes, err := s.Tree()
	if err != nil {
		t.Fatal(err)
	}
	paths := map[string]bool{}
	var collect func([]Node)
	collect = func(ns []Node) {
		for _, n := range ns {
			paths[n.Path] = true
			collect(n.Children)
		}
	}
	collect(nodes)
	for _, p := range []string{"Sub/assets", "Mine/assets", "Empty/assets", "a.md"} {
		if !paths[p] {
			t.Errorf("Tree is missing %s", p)
		}
	}
	if paths["assets"] {
		t.Error("Tree shows the image-only assets folder")
	}
}
