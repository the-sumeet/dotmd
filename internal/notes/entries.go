package notes

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// moveToTrash moves the file or folder at an absolute path to the system
// Trash. It is a variable so tests can stub it.
var moveToTrash = trash

var errExists = errors.New("a file or folder with that name already exists")

// Rename gives the note, folder or image at path a new name in the same
// folder and returns its new root-relative path. A note or image keeps its
// extension if name leaves it out ("Plan" -> "Plan.md"). Renaming onto an
// existing entry is an error, except to change only the case of the name.
func (s *NotesService) Rename(path, name string) (string, error) {
	abs, info, err := s.entry(path)
	if err != nil {
		return "", err
	}
	name = strings.TrimSpace(name)
	if name == "" || strings.HasPrefix(name, ".") {
		return "", errors.New("names cannot be empty or start with \".\"")
	}
	if strings.ContainsAny(name, `/\`) {
		return "", errors.New("names cannot contain \"/\" or \"\\\"")
	}
	if !info.IsDir() {
		old := filepath.Ext(abs)
		switch {
		case isMarkdown(abs) && !isMarkdown(name):
			name += old
		case isImage(abs) && !strings.EqualFold(filepath.Ext(name), old):
			name += old
		}
	}
	target := filepath.Join(filepath.Dir(abs), name)
	if target == abs {
		return s.rel(abs), nil
	}
	// On a case-insensitive disk "a.md" -> "A.md" finds the note itself, so
	// the rename has to replace "the target". A hard link with another name
	// is also the same file, but must not be replaced.
	if strings.EqualFold(name, filepath.Base(abs)) {
		if existing, err := os.Lstat(target); err == nil && os.SameFile(info, existing) {
			if err := os.Rename(abs, target); err != nil {
				return "", err
			}
			return s.rel(target), nil
		}
	}
	if err := renameNoReplace(abs, target); err != nil {
		return "", err
	}
	return s.rel(target), nil
}

// Delete moves the note, folder (with everything in it) or image at path to
// the system Trash. An "assets" folder left empty by deleting an image is
// removed too, so it does not linger beside the note.
func (s *NotesService) Delete(path string) error {
	abs, info, err := s.entry(path)
	if err != nil {
		return err
	}
	if err := moveToTrash(abs); err != nil {
		return err
	}
	if !info.IsDir() && isImage(abs) {
		if dir := filepath.Dir(abs); filepath.Base(dir) == assetsDir {
			_ = os.Remove(dir) // fails, harmlessly, unless the folder is empty
		}
	}
	return nil
}

// entry resolves path to an entry shown in the Files tree: a folder, a note
// or an image that is not hidden and not a symlink.
func (s *NotesService) entry(path string) (string, fs.FileInfo, error) {
	abs, err := s.resolve(path)
	if err != nil {
		return "", nil, err
	}
	for seg := range strings.SplitSeq(s.rel(abs), "/") {
		if strings.HasPrefix(seg, ".") {
			return "", nil, errors.New("hidden files and folders can't be changed here")
		}
	}
	info, err := os.Lstat(abs)
	if err != nil {
		return "", nil, err
	}
	if !info.IsDir() && !(info.Mode().IsRegular() && (isMarkdown(abs) || isImage(abs))) {
		return "", nil, errors.New("only notes, folders and images can be changed here")
	}
	return abs, info, nil
}
