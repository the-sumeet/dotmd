//go:build !darwin && !linux && !windows

package notes

// renameNoReplace renames from to to, failing with errExists if to exists.
func renameNoReplace(from, to string) error {
	return renameIfAbsent(from, to)
}
