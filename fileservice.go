package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

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
