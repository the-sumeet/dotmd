//go:build !windows

package notes

import (
	"errors"
	"io/fs"
	"os"
)

// renameIfAbsent renames from to to unless to exists. The check and the
// rename are separate steps, so it is only the fallback where the system
// can't do both at once.
func renameIfAbsent(from, to string) error {
	if _, err := os.Lstat(to); err == nil {
		return errExists
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return os.Rename(from, to)
}
