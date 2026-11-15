//go:build !darwin && !windows && !(linux && !android)

package notes

import "errors"

// trash is unavailable where the system has no Trash (e.g. Android).
func trash(abs string) error {
	return errors.New("moving to Trash isn't supported on this device")
}
