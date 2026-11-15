//go:build linux && !android

package notes

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// trash moves abs to the Trash with gio, which follows the freedesktop.org
// Trash spec (and handles other disks) on GNOME, KDE and most desktops.
func trash(abs string) error {
	gio, err := exec.LookPath("gio")
	if err != nil {
		return errors.New("couldn't move to Trash: the gio command (part of GLib) isn't installed")
	}
	out, err := exec.Command(gio, "trash", "--", abs).CombinedOutput()
	if err != nil {
		if msg := strings.TrimSpace(string(out)); msg != "" {
			return fmt.Errorf("couldn't move to Trash: %s", msg)
		}
		return fmt.Errorf("couldn't move to Trash: %w", err)
	}
	return nil
}
