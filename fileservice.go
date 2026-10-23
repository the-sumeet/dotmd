package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/adrg/xdg"
)

// FileNode is a file or directory inside the data directory.
type FileNode struct {
	Name     string     `json:"name"`
	Path     string     `json:"path"` // relative to the data directory
	IsDir    bool       `json:"isDir"`
	Children []FileNode `json:"children"`
}

// FileService exposes the app's data directory to the frontend.
type FileService struct {
	dir string
}

// NewFileService resolves the platform data directory (via xdg) and ensures it exists.
func NewFileService() (*FileService, error) {
	dir := filepath.Join(xdg.DataHome, "dotmd")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &FileService{dir: dir}, nil
}

// DataDir returns the absolute path of the data directory.
func (f *FileService) DataDir() string {
	return f.dir
}

// ListTree returns the contents of the data directory, folders first.
func (f *FileService) ListTree() ([]FileNode, error) {
	return readTree(f.dir, "")
}

// resolve maps a slash-separated relative path to an absolute path inside the data directory.
func (f *FileService) resolve(rel string) (string, error) {
	abs := filepath.Join(f.dir, filepath.FromSlash(rel))
	inside, err := filepath.Rel(f.dir, abs)
	if err != nil || inside == "." || inside == ".." || strings.HasPrefix(inside, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid path %q", rel)
	}
	return abs, nil
}

// ReadFile returns the contents of a file in the data directory.
func (f *FileService) ReadFile(rel string) (string, error) {
	abs, err := f.resolve(rel)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// WriteFile atomically replaces the contents of a file in the data directory.
func (f *FileService) WriteFile(rel, content string) error {
	abs, err := f.resolve(rel)
	if err != nil {
		return err
	}
	if info, err := os.Stat(abs); err == nil && info.IsDir() {
		return fmt.Errorf("%q is a directory", rel)
	}
	tmp, err := os.CreateTemp(filepath.Dir(abs), ".dotmd-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), abs)
}

// AbsPath returns the absolute filesystem path of an entry in the data directory.
func (f *FileService) AbsPath(rel string) (string, error) {
	return f.resolve(rel)
}

// Rename moves a file or folder to a new path. It fails if the destination already exists.
func (f *FileService) Rename(from, to string) error {
	src, err := f.resolve(from)
	if err != nil {
		return err
	}
	dst, err := f.resolve(to)
	if err != nil {
		return err
	}
	srcInfo, err := os.Lstat(src)
	if err != nil {
		return err
	}
	// A case-only rename on a case-insensitive filesystem finds the source itself.
	if dstInfo, err := os.Lstat(dst); err == nil && !os.SameFile(srcInfo, dstInfo) {
		return fmt.Errorf("%q already exists", to)
	}
	return os.Rename(src, dst)
}

// Delete permanently removes a file, or a folder and everything in it.
func (f *FileService) Delete(rel string) error {
	abs, err := f.resolve(rel)
	if err != nil {
		return err
	}
	if _, err := os.Lstat(abs); err != nil {
		return err
	}
	return os.RemoveAll(abs)
}

// CreateFile creates an empty file called name inside the folder dir ("" is the data directory)
// and returns its relative path. A name without an extension gets ".md". It never overwrites.
func (f *FileService) CreateFile(dir, name string) (string, error) {
	return f.create(dir, name, false)
}

// CreateFolder creates an empty folder called name inside the folder dir ("" is the data directory)
// and returns its relative path. It never overwrites.
func (f *FileService) CreateFolder(dir, name string) (string, error) {
	return f.create(dir, name, true)
}

func (f *FileService) create(dir, name string, isDir bool) (string, error) {
	name = strings.TrimSpace(name)
	switch {
	case name == "":
		return "", errors.New("name can't be empty")
	case strings.ContainsAny(name, `/\`):
		return "", errors.New(`name can't contain / or \`)
	case strings.HasPrefix(name, "."):
		return "", errors.New("names starting with . are hidden from the file list")
	}
	if !isDir && filepath.Ext(name) == "" {
		name += ".md"
	}
	if len(name) > maxNameBytes {
		return "", fmt.Errorf("name is too long (max %d bytes)", maxNameBytes)
	}
	parent := f.dir
	if dir != "" {
		var err error
		if parent, err = f.resolve(dir); err != nil {
			return "", err
		}
	}
	if info, err := os.Stat(parent); err != nil {
		return "", err
	} else if !info.IsDir() {
		return "", fmt.Errorf("%q is not a folder", dir)
	}
	abs := filepath.Join(parent, name)
	perm := os.FileMode(0o644)
	if isDir {
		perm = 0o755
	}
	if err := claim(abs, isDir, perm); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return "", fmt.Errorf("%q already exists", name)
		}
		return "", err
	}
	out, err := filepath.Rel(f.dir, abs)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(out), nil
}

// Duplicate copies a file or folder next to itself as "name copy.ext" (or "name copy 2.ext", …)
// and returns the new relative path.
func (f *FileService) Duplicate(rel string) (string, error) {
	src, err := f.resolve(rel)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(src)
	if err != nil {
		return "", err
	}
	ext := ""
	if !info.IsDir() {
		ext = filepath.Ext(src)
	}
	dir := filepath.Dir(src)
	stem := strings.TrimSuffix(filepath.Base(src), ext)
	// Claim the destination atomically, so the cleanup below only ever removes what we created.
	var dst string
	for i := 1; ; i++ {
		suffix := " copy" + ext
		if i > 1 {
			suffix = fmt.Sprintf(" copy %d%s", i, ext)
		}
		dst = filepath.Join(dir, fitName(stem, suffix)+suffix)
		err = claim(dst, info.IsDir(), info.Mode().Perm())
		if err == nil {
			break
		}
		if !errors.Is(err, fs.ErrExist) {
			return "", err
		}
	}
	if info.IsDir() {
		err = copyDir(src, dst)
	} else {
		err = copyFile(src, dst, info.Mode())
	}
	if err != nil {
		os.RemoveAll(dst)
		return "", err
	}
	out, err := filepath.Rel(f.dir, dst)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(out), nil
}

// imageExts are the file types SaveImage accepts.
var imageExts = map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true, ".svg": true}

// maxImageBytes caps the size of a saved image.
const maxImageBytes = 25 << 20

// SaveImage writes data as a new "image-YYYYMMDD-HHMMSS<ext>" file inside the folder dir ("" is the
// data directory) and returns its relative path. It never overwrites; a taken name gets " 2", " 3", ….
func (f *FileService) SaveImage(dir, ext string, data []byte) (string, error) {
	ext = strings.ToLower(ext)
	if !imageExts[ext] {
		return "", fmt.Errorf("unsupported image type %q", ext)
	}
	if len(data) > maxImageBytes {
		return "", fmt.Errorf("image is too large (max %d MB)", maxImageBytes>>20)
	}
	parent := f.dir
	if dir != "" {
		var err error
		if parent, err = f.resolve(dir); err != nil {
			return "", err
		}
	}
	if info, err := os.Stat(parent); err != nil {
		return "", err
	} else if !info.IsDir() {
		return "", fmt.Errorf("%q is not a folder", dir)
	}
	stem := "image-" + time.Now().Format("20060102-150405")
	var dst string
	for i := 1; ; i++ {
		suffix := ext
		if i > 1 {
			suffix = fmt.Sprintf(" %d%s", i, ext)
		}
		dst = filepath.Join(parent, stem+suffix)
		out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err == nil {
			_, err = out.Write(data)
			if cerr := out.Close(); err == nil {
				err = cerr
			}
			if err != nil {
				os.Remove(dst)
				return "", err
			}
			break
		}
		if !errors.Is(err, fs.ErrExist) {
			return "", err
		}
	}
	rel, err := filepath.Rel(f.dir, dst)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(rel), nil
}

// FilesPrefix is the URL path under which handler serves images from the data directory.
const FilesPrefix = "/files/"

// handler serves the images in the data directory read-only under FilesPrefix, so notes can show
// the ones saved next to them. Hidden (dot-prefixed) entries, other file types, folders and
// symlinks that leave the data directory are not served.
func (f *FileService) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rel := strings.TrimPrefix(r.URL.Path, FilesPrefix)
		for _, seg := range strings.Split(rel, "/") {
			if strings.HasPrefix(seg, ".") {
				http.NotFound(w, r)
				return
			}
		}
		if !imageExts[strings.ToLower(filepath.Ext(rel))] {
			http.NotFound(w, r)
			return
		}
		abs, err := f.resolve(rel)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		real, err := filepath.EvalSymlinks(abs)
		root, rootErr := filepath.EvalSymlinks(f.dir)
		info, statErr := os.Stat(real)
		if err != nil || rootErr != nil || statErr != nil || !info.Mode().IsRegular() ||
			!strings.HasPrefix(real, root+string(filepath.Separator)) {
			http.NotFound(w, r)
			return
		}
		// Without this the webview may keep showing a file that was deleted or replaced.
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, r, real)
	})
}

// maxNameBytes is the longest file name most filesystems allow (NAME_MAX on macOS and Linux).
const maxNameBytes = 255

// fitName trims stem so that stem+suffix fits in maxNameBytes, without splitting a UTF-8 character.
func fitName(stem, suffix string) string {
	n := max(maxNameBytes-len(suffix), 0)
	if len(stem) <= n {
		return stem
	}
	for n > 0 && !utf8.RuneStart(stem[n]) {
		n--
	}
	return stem[:n]
}

// claim creates an empty file or folder at path, failing with fs.ErrExist if anything is already there.
func claim(path string, isDir bool, perm os.FileMode) error {
	if isDir {
		return os.Mkdir(path, perm)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return err
	}
	return f.Close()
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode.Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		info, err := d.Info()
		if err != nil {
			return err
		}
		switch {
		case rel == ".":
			return nil // dst itself was already created by claim
		case d.IsDir():
			return os.Mkdir(target, info.Mode().Perm())
		case info.Mode().IsRegular():
			return copyFile(path, target, info.Mode())
		default:
			return nil // skip symlinks and special files
		}
	})
}

func readTree(root, rel string) ([]FileNode, error) {
	entries, err := os.ReadDir(filepath.Join(root, rel))
	if err != nil {
		return nil, err
	}
	nodes := []FileNode{}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		node := FileNode{Name: e.Name(), Path: filepath.ToSlash(filepath.Join(rel, e.Name())), IsDir: e.IsDir(), Children: []FileNode{}}
		if e.IsDir() {
			// An unreadable subfolder shouldn't hide the rest of the tree.
			if children, err := readTree(root, filepath.FromSlash(node.Path)); err == nil {
				node.Children = children
			}
		}
		nodes = append(nodes, node)
	}
	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i].IsDir != nodes[j].IsDir {
			return nodes[i].IsDir
		}
		return strings.ToLower(nodes[i].Name) < strings.ToLower(nodes[j].Name)
	})
	return nodes, nil
}
