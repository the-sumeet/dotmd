package notes

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Tag is a frontmatter tag and the notes (root-relative paths) that carry it.
type Tag struct {
	Name  string   `json:"name"`
	Notes []string `json:"notes"`
}

// Tags collects the `tags` frontmatter field of every note, sorted by tag name
// (case-insensitive). Notes that cannot be read or parsed are skipped.
func (s *NotesService) Tags() ([]Tag, error) {
	root := s.Root()
	byName := map[string]*Tag{}
	err := filepath.WalkDir(root, func(abs string, d fs.DirEntry, err error) error {
		if err != nil {
			// Only a missing or unreadable root is fatal; skip anything else.
			if abs == root {
				return err
			}
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if abs != root && strings.HasPrefix(d.Name(), ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() || !d.Type().IsRegular() || !isMarkdown(d.Name()) {
			return nil
		}
		data, err := os.ReadFile(abs)
		if err != nil {
			return nil
		}
		rel, err := filepath.Rel(root, abs)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		for _, name := range parseTags(data) {
			key := strings.ToLower(name)
			t := byName[key]
			if t == nil {
				t = &Tag{Name: name}
				byName[key] = t
			}
			if len(t.Notes) == 0 || t.Notes[len(t.Notes)-1] != rel {
				t.Notes = append(t.Notes, rel)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	tags := make([]Tag, 0, len(byName))
	for _, t := range byName {
		sort.Slice(t.Notes, func(i, j int) bool {
			return strings.ToLower(t.Notes[i]) < strings.ToLower(t.Notes[j])
		})
		tags = append(tags, *t)
	}
	sort.Slice(tags, func(i, j int) bool {
		return strings.ToLower(tags[i].Name) < strings.ToLower(tags[j].Name)
	})
	return tags, nil
}

// TagsOf returns the frontmatter tags of a note's content, so the frontend
// can update the tag list after a save without rescanning every note.
func (s *NotesService) TagsOf(content string) []string {
	tags := parseTags([]byte(content))
	if tags == nil {
		return []string{}
	}
	return tags
}

// parseTags reads the `tags` field from a note's YAML frontmatter. It accepts a
// list (`tags: [a, b]` or a block list) or a string separated by commas or
// spaces (`tags: a, b`). A leading `#` is dropped.
func parseTags(note []byte) []string {
	fm, ok := frontmatter(note)
	if !ok {
		return nil
	}
	var meta struct {
		Tags any `yaml:"tags"`
	}
	if yaml.Unmarshal(fm, &meta) != nil {
		return nil
	}

	var raw []string
	switch v := meta.Tags.(type) {
	case string:
		raw = strings.FieldsFunc(v, func(r rune) bool { return r == ',' || r == ' ' })
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok {
				raw = append(raw, s)
			}
		}
	}

	seen := map[string]bool{}
	var tags []string
	for _, t := range raw {
		t = strings.TrimPrefix(strings.TrimSpace(t), "#")
		if t == "" || seen[strings.ToLower(t)] {
			continue
		}
		seen[strings.ToLower(t)] = true
		tags = append(tags, t)
	}
	return tags
}

// frontmatter returns the YAML between an opening `---` line at the very start
// of the note and the next `---` line.
func frontmatter(note []byte) ([]byte, bool) {
	note = bytes.ReplaceAll(note, []byte("\r\n"), []byte("\n"))
	first, rest, ok := bytes.Cut(note, []byte("\n"))
	if !ok || string(bytes.TrimRight(first, " \t")) != "---" {
		return nil, false
	}
	for start := 0; start <= len(rest); {
		end := bytes.IndexByte(rest[start:], '\n')
		line := rest[start:]
		if end >= 0 {
			line = rest[start : start+end]
		}
		if string(bytes.TrimRight(line, " \t")) == "---" {
			return rest[:start], true
		}
		if end < 0 {
			break
		}
		start += end + 1
	}
	return nil, false
}
