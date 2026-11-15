// Package notes exposes the user's notes folder to the frontend: listing it
// as a tree and reading/writing markdown files inside it.
package notes

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Node is one entry in the notes tree: a folder, a note or an image. Path is
// relative to the notes root and always uses forward slashes.
type Node struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	IsDir    bool   `json:"isDir"`
	IsImage  bool   `json:"isImage"`
	Children []Node `json:"children"`
}

// NotesService gives the frontend access to the notes folder. Every path it
// accepts is relative to the root and may not escape it.
type NotesService struct {
	mu   sync.RWMutex
	root string
}

// New returns a service rooted at the default notes folder (~/Documents/dotmd).
func New() *NotesService {
	return NewAt(DefaultRoot())
}

// NewAt returns a service rooted at root, which is created on startup if it
// does not exist.
func NewAt(root string) *NotesService {
	return &NotesService{root: root}
}

// DefaultRoot is the notes folder used until the user picks their own.
func DefaultRoot() string {
	docs := application.Path(application.PathDocuments)
	if docs == "" {
		home, _ := os.UserHomeDir()
		docs = filepath.Join(home, "Documents")
	}
	return filepath.Join(docs, "dotmd")
}

// ServiceStartup creates the notes folder if it does not exist yet.
func (s *NotesService) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	return os.MkdirAll(s.Root(), 0o755)
}

// Root returns the absolute path of the notes folder.
func (s *NotesService) Root() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.root
}

// SetRoot switches to another notes folder, which must already exist. The
// frontend changes folders through the settings service, which saves it.
//
//wails:ignore
func (s *NotesService) SetRoot(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("not a folder: " + abs)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.root = abs
	return nil
}

// Tree lists the folders, markdown notes and images under the root; image
// nodes have IsImage set, so callers must not treat every file as a note.
// Hidden entries (names starting with ".") are skipped. Folders sort before
// files.
func (s *NotesService) Tree() ([]Node, error) {
	return walk(s.Root(), "")
}

// Read returns the contents of the note at path.
func (s *NotesService) Read(path string) (string, error) {
	abs, err := s.resolve(path)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Write saves content to the note at path, creating parent folders as needed.
// The file is replaced atomically so a crash never leaves a half-written note.
func (s *NotesService) Write(path, content string) error {
	abs, err := s.resolve(path)
	if err != nil {
		return err
	}
	if !isMarkdown(abs) {
		return errors.New("only markdown files can be saved")
	}
	// Write through a symlink to the note it points to; replacing the link
	// itself would turn it into a separate copy.
	if info, err := os.Lstat(abs); err == nil && info.Mode()&os.ModeSymlink != 0 {
		if abs, err = filepath.EvalSymlinks(abs); err != nil {
			return err
		}
		if !isMarkdown(abs) {
			return errors.New("only markdown files can be saved")
		}
	}
	dir := filepath.Dir(abs)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".dotmd-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no-op after a successful rename
	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	// CreateTemp makes the file 0600; keep the note's existing mode instead.
	mode := os.FileMode(0o644)
	if info, err := os.Stat(abs); err == nil {
		mode = info.Mode().Perm()
	}
	if err := os.Chmod(tmp.Name(), mode); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), abs)
}

// CreateNote creates an empty note at path (".md" is appended unless it
// already has a markdown extension) and returns its root-relative path.
// Missing parent folders are created; an existing note is an error.
func (s *NotesService) CreateNote(path string) (string, error) {
	path, err := cleanNewPath(path)
	if err != nil {
		return "", err
	}
	if !isMarkdown(path) {
		path += ".md"
	}
	abs, err := s.resolve(path)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return "", err
	}
	f, err := os.OpenFile(abs, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			return "", errors.New("a note with that name already exists")
		}
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	return s.rel(abs), nil
}

// CreateFolder creates a folder (and any missing parents) at path and returns
// its root-relative path. An existing entry is an error.
func (s *NotesService) CreateFolder(path string) (string, error) {
	path, err := cleanNewPath(path)
	if err != nil {
		return "", err
	}
	abs, err := s.resolve(path)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(abs); err == nil {
		return "", errors.New("a file or folder with that name already exists")
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return "", err
	}
	return s.rel(abs), nil
}

// cleanNewPath tidies a path typed by the user for a new entry: spaces around
// each segment are trimmed ("Projects / Idea" -> "Projects/Idea"), and empty or
// hidden segments are rejected, since hidden entries would not show in the tree.
func cleanNewPath(rel string) (string, error) {
	segs := strings.Split(filepath.ToSlash(rel), "/")
	for i, seg := range segs {
		seg = strings.TrimSpace(seg)
		if seg == "" || strings.HasPrefix(seg, ".") {
			return "", errors.New("names cannot be empty or start with \".\"")
		}
		segs[i] = seg
	}
	return strings.Join(segs, "/"), nil
}

// rel converts an absolute path under the root to a root-relative slash path.
func (s *NotesService) rel(abs string) string {
	r, _ := filepath.Rel(s.Root(), abs)
	return filepath.ToSlash(r)
}

// resolve turns a root-relative path into an absolute one, rejecting paths
// that are absolute or would escape the root.
func (s *NotesService) resolve(rel string) (string, error) {
	if rel == "" || filepath.IsAbs(rel) || strings.HasPrefix(rel, "/") {
		return "", errors.New("path must be inside the notes folder")
	}
	root := s.Root()
	abs := filepath.Join(root, filepath.FromSlash(rel))
	r, err := filepath.Rel(root, abs)
	if err != nil || r == "." || r == ".." || strings.HasPrefix(r, ".."+string(filepath.Separator)) {
		return "", errors.New("path must be inside the notes folder")
	}
	return abs, nil
}

func walk(root, rel string) ([]Node, error) {
	entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return nil, err
	}
	nodes := []Node{}
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		path := name
		if rel != "" {
			path = rel + "/" + name
		}
		if e.IsDir() {
			// An unreadable subfolder (e.g. no permission) is listed as empty
			// rather than failing the whole tree.
			children, err := walk(root, path)
			if err != nil {
				children = []Node{}
			}
			nodes = append(nodes, Node{Name: name, Path: path, IsDir: true, Children: children})
		} else if e.Type().IsRegular() && isMarkdown(name) {
			nodes = append(nodes, Node{Name: name, Path: path, Children: []Node{}})
		} else if e.Type().IsRegular() && isImage(name) {
			nodes = append(nodes, Node{Name: name, Path: path, IsImage: true, Children: []Node{}})
		}
	}
	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i].IsDir != nodes[j].IsDir {
			return nodes[i].IsDir
		}
		return strings.ToLower(nodes[i].Name) < strings.ToLower(nodes[j].Name)
	})
	return nodes, nil
}

func isMarkdown(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".md", ".markdown":
		return true
	}
	return false
}
