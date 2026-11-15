package notes

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

// renameNoReplace renames from to to, failing with errExists if to exists.
// The check and the rename are one call, so a file created meanwhile (by a
// sync client, say) is never overwritten.
func renameNoReplace(from, to string) error {
	err := unix.RenamexNp(from, to, unix.RENAME_EXCL)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, unix.EEXIST):
		return errExists
	case errors.Is(err, unix.ENOTSUP) || errors.Is(err, unix.EINVAL):
		// The disk can't do it atomically; check first instead.
		return renameIfAbsent(from, to)
	}
	return &os.LinkError{Op: "rename", Old: from, New: to, Err: err}
}
