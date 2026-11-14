package notes

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// ResolveLink turns the href of a markdown link in the note at from into the
// root-relative path of the note it points to. Like markdown links elsewhere,
// hrefs are relative to the linking note's folder; a leading "/" makes them
// relative to the notes root. A "#heading" or "?query" suffix is ignored,
// percent-escapes are decoded, and ".md" may be left off. The target must be
// an existing markdown file inside the notes folder.
func (s *NotesService) ResolveLink(from, href string) (string, error) {
	target := href
	if i := strings.IndexAny(target, "#?"); i >= 0 {
		target = target[:i]
	}
	if unescaped, err := url.PathUnescape(target); err == nil {
		target = unescaped
	}
	notFound := fmt.Errorf("no note at %q", href)
	if target == "" {
		return "", notFound
	}
	if strings.HasPrefix(target, "/") {
		target = strings.TrimLeft(target, "/")
	} else {
		target = path.Join(path.Dir(filepath.ToSlash(from)), target)
	}
	abs, err := s.resolve(target)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	// "Projects" may name both a folder and the note "Projects.md".
	if (err != nil || info.IsDir()) && !isMarkdown(abs) {
		abs += ".md"
		info, err = os.Stat(abs)
	}
	if err != nil || !info.Mode().IsRegular() || !isMarkdown(abs) {
		return "", notFound
	}
	return s.realCase(s.rel(abs)), nil
}

// realCase returns rel (an existing root-relative path) spelled the way the
// entries are named on disk. On a case-insensitive filesystem a link to
// "plan.md" finds "Plan.md", and the note must be opened by its real name, or
// it would be tracked (and on save, renamed) as "plan.md".
func (s *NotesService) realCase(rel string) string {
	dir := s.Root()
	segs := strings.Split(rel, "/")
	for i, seg := range segs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return rel
		}
		match := ""
		for _, e := range entries {
			if e.Name() == seg {
				match = seg
				break
			}
			if match == "" && strings.EqualFold(e.Name(), seg) {
				match = e.Name()
			}
		}
		if match == "" {
			return rel
		}
		segs[i] = match
		dir = filepath.Join(dir, match)
	}
	return strings.Join(segs, "/")
}

// hrefEscaper escapes the characters that would otherwise end the path part
// of an href ("#", "?") or start an escape ("%").
var hrefEscaper = strings.NewReplacer("%", "%25", "#", "%23", "?", "%3F")

// LinkPath returns the href that links the note at from to the note at to,
// relative to from's folder (e.g. "Projects/A.md" -> "Plan.md" is "../Plan.md").
// Both paths are root-relative. Spaces are kept; "%", "#" and "?" are escaped
// so ResolveLink reads them as part of the name.
func (s *NotesService) LinkPath(from, to string) string {
	dir := filepath.Dir(filepath.FromSlash(from))
	rel, err := filepath.Rel(dir, filepath.FromSlash(to))
	if err != nil {
		rel = to
	}
	return hrefEscaper.Replace(filepath.ToSlash(rel))
}
