package notes

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Images are saved to an "assets" folder beside the note that shows them.
const assetsDir = "assets"

// FileRoute is the URL prefix under which the webview loads images from the
// notes folder: "/notes-file/Projects/assets/a.png".
const FileRoute = "/notes-file/"

const (
	maxImageBytes   = 25 << 20 // largest image saved or downloaded
	downloadTimeout = 30 * time.Second
)

// imageExts are the file types treated as images (listed, served, saved).
var imageExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
	".webp": true, ".avif": true, ".bmp": true, ".svg": true,
}

// extByType maps a sniffed or declared content type to the extension to save.
var extByType = map[string]string{
	"image/png":     ".png",
	"image/jpeg":    ".jpg",
	"image/gif":     ".gif",
	"image/webp":    ".webp",
	"image/avif":    ".avif",
	"image/bmp":     ".bmp",
	"image/svg+xml": ".svg",
}

func isImage(name string) bool {
	return imageExts[strings.ToLower(filepath.Ext(name))]
}

// hasImages reports whether the folder at abs directly holds an image file.
func hasImages(abs string) bool {
	entries, err := os.ReadDir(abs)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.Type().IsRegular() && isImage(e.Name()) {
			return true
		}
	}
	return false
}

// Image is an image file in the notes folder. Path is root-relative and uses
// forward slashes.
type Image struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Size int64  `json:"size"`
}

// Images lists every image under the notes root, sorted by path
// (case-insensitive). Hidden files and folders are skipped.
func (s *NotesService) Images() ([]Image, error) {
	root := s.Root()
	// Follow a symlinked root, like Tags.
	if resolved, err := filepath.EvalSymlinks(root); err == nil {
		root = resolved
	}
	images := []Image{}
	err := filepath.WalkDir(root, func(abs string, d fs.DirEntry, err error) error {
		if err != nil {
			if abs == root {
				return err
			}
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if abs != root && strings.HasPrefix(d.Name(), ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() || !d.Type().IsRegular() || !isImage(d.Name()) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		rel, err := filepath.Rel(root, abs)
		if err != nil {
			return nil
		}
		images = append(images, Image{Name: d.Name(), Path: filepath.ToSlash(rel), Size: info.Size()})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(images, func(i, j int) bool {
		return strings.ToLower(images[i].Path) < strings.ToLower(images[j].Path)
	})
	return images, nil
}

// SaveImage saves base64-encoded image data (pasted, dropped or picked) to the
// "assets" folder beside the note at note, and returns the href that links the
// note to it (e.g. "assets/plan-20261115-142301.png"). name is the original
// file name, if any; its extension is used when the data alone does not say.
func (s *NotesService) SaveImage(note, name, dataBase64 string) (string, error) {
	if base64.StdEncoding.DecodedLen(len(dataBase64)) > maxImageBytes {
		return "", errTooLarge
	}
	data, err := base64.StdEncoding.DecodeString(dataBase64)
	if err != nil {
		return "", errors.New("image data is not valid base64")
	}
	ext, err := imageExt(data, filepath.Ext(name), "")
	if err != nil {
		return "", err
	}
	return s.saveAsset(note, ext, data)
}

// DownloadImage downloads the image at rawURL (http or https) into the
// "assets" folder beside the note at note, and returns the href that links
// the note to it.
func (s *NotesService) DownloadImage(note, rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", errors.New("only http and https images can be downloaded")
	}
	// Fail before the download if the note is not valid.
	if _, err := s.noteFile(note); err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), downloadTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("couldn't download image: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("couldn't download image: %s", resp.Status)
	}
	declared, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if declared != "" && !strings.HasPrefix(declared, "image/") && declared != "application/octet-stream" {
		return "", errors.New("that link is not an image")
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxImageBytes+1))
	if err != nil {
		return "", fmt.Errorf("couldn't download image: %w", err)
	}
	if len(data) > maxImageBytes {
		return "", errTooLarge
	}
	ext, err := imageExt(data, path.Ext(u.Path), declared)
	if err != nil {
		return "", err
	}
	return s.saveAsset(note, ext, data)
}

// DeleteImage deletes the image at path. An "assets" folder left empty is
// removed too, so it does not linger beside the note.
func (s *NotesService) DeleteImage(p string) error {
	abs, err := s.resolve(p)
	if err != nil {
		return err
	}
	if !isImage(abs) {
		return errors.New("only images can be deleted here")
	}
	info, err := os.Lstat(abs)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("only images can be deleted here")
	}
	if err := os.Remove(abs); err != nil {
		return err
	}
	if dir := filepath.Dir(abs); filepath.Base(dir) == assetsDir {
		_ = os.Remove(dir) // fails, harmlessly, unless the folder is empty
	}
	return nil
}

var errTooLarge = fmt.Errorf("images can be at most %d MB", maxImageBytes>>20)

// imageExt checks that data is an image and returns the extension to save it
// with. The sniffed type wins; nameExt (from a file name or URL) and declared
// (a Content-Type) are used for types Go cannot sniff (SVG, AVIF).
func imageExt(data []byte, nameExt, declared string) (string, error) {
	nameExt = strings.ToLower(nameExt)
	if nameExt == ".jpeg" {
		nameExt = ".jpg"
	}
	sniffed, _, _ := mime.ParseMediaType(http.DetectContentType(data))
	if ext, ok := extByType[sniffed]; ok {
		return ext, nil
	}
	if (nameExt == ".svg" || declared == "image/svg+xml") && looksLikeSVG(data) {
		return ".svg", nil
	}
	if (nameExt == ".avif" || declared == "image/avif") && len(data) >= 12 &&
		string(data[4:8]) == "ftyp" && (string(data[8:12]) == "avif" || string(data[8:12]) == "avis") {
		return ".avif", nil
	}
	return "", errors.New("that file is not a supported image")
}

func looksLikeSVG(data []byte) bool {
	head := data
	if len(head) > 1024 {
		head = head[:1024]
	}
	return bytes.Contains(bytes.ToLower(head), []byte("<svg"))
}

// noteFile resolves note and checks it is a markdown path.
func (s *NotesService) noteFile(note string) (string, error) {
	abs, err := s.resolve(note)
	if err != nil {
		return "", err
	}
	if !isMarkdown(abs) {
		return "", errors.New("images can only be added to a note")
	}
	return abs, nil
}

// saveAsset writes data to <note folder>/assets/<note-slug>-<timestamp><ext>,
// adding "-2", "-3", … if that name is taken, and returns the href from the
// note to the new file.
func (s *NotesService) saveAsset(note, ext string, data []byte) (string, error) {
	abs, err := s.noteFile(note)
	if err != nil {
		return "", err
	}
	dir := filepath.Join(filepath.Dir(abs), assetsDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	base := slug(strings.TrimSuffix(filepath.Base(abs), filepath.Ext(abs))) +
		"-" + time.Now().Format("20060102-150405")
	for n := 1; ; n++ {
		name := base
		if n > 1 {
			name += "-" + strconv.Itoa(n)
		}
		target := filepath.Join(dir, name+ext)
		f, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		if _, err := f.Write(data); err != nil {
			f.Close()
			os.Remove(target)
			return "", err
		}
		if err := f.Close(); err != nil {
			os.Remove(target)
			return "", err
		}
		return s.LinkPath(s.rel(abs), s.rel(target)), nil
	}
}

// slug lowercases name and turns every run of other characters than letters
// and digits into a single "-" ("My Plan!" -> "my-plan").
func slug(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(name) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if dash && b.Len() > 0 {
				b.WriteByte('-')
			}
			b.WriteRune(r)
			dash = false
		} else {
			dash = true
		}
	}
	if b.Len() == 0 {
		return "image"
	}
	return b.String()
}

// FileMiddleware serves images from the notes folder at FileRoute, so a note's
// relative image links can be shown in the webview. Only images are served;
// any other request goes on to next.
func FileMiddleware(s *NotesService) application.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rel, ok := strings.CutPrefix(r.URL.Path, FileRoute)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}
			s.serveImage(w, r, rel)
		})
	}
}

func (s *NotesService) serveImage(w http.ResponseWriter, r *http.Request, rel string) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	abs, err := s.resolve(rel)
	if err != nil || !isImage(abs) {
		http.NotFound(w, r)
		return
	}
	f, err := os.Open(abs)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		http.NotFound(w, r)
		return
	}
	h := w.Header()
	if ct := mime.TypeByExtension(strings.ToLower(filepath.Ext(abs))); ct != "" {
		h.Set("Content-Type", ct)
	}
	h.Set("X-Content-Type-Options", "nosniff")
	// An SVG opened directly must not run scripts with the app's origin.
	h.Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; sandbox")
	h.Set("Cache-Control", "no-cache")
	http.ServeContent(w, r, info.Name(), info.ModTime(), f)
}
