package notes

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

// renameNoReplace renames from to to, failing with errExists if to exists.
// Unlike os.Rename, MoveFileEx without MOVEFILE_REPLACE_EXISTING never
// overwrites, so a file created meanwhile (by a sync client, say) is kept.
func renameNoReplace(from, to string) error {
	f, err := windows.UTF16PtrFromString(from)
	if err != nil {
		return err
	}
	t, err := windows.UTF16PtrFromString(to)
	if err != nil {
		return err
	}
	err = windows.MoveFileEx(f, t, 0)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, windows.ERROR_ALREADY_EXISTS) || errors.Is(err, windows.ERROR_FILE_EXISTS):
		return errExists
	}
	return &os.LinkError{Op: "rename", Old: from, New: to, Err: err}
}
