package notes

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseTags(t *testing.T) {
	cases := []struct {
		name, note string
		want       []string
	}{
		{"flow list", "---\ntags: [notes, markdown]\n---\n# hi\n", []string{"notes", "markdown"}},
		{"block list", "---\ntitle: x\ntags:\n  - a\n  - '#b'\n---\n", []string{"a", "b"}},
		{"string", "---\ntags: a, b c\n---\n", []string{"a", "b", "c"}},
		{"crlf", "---\r\ntags: [a]\r\n---\r\n", []string{"a"}},
		{"dedupe", "---\ntags: [a, A, a]\n---\n", []string{"a"}},
		{"no frontmatter", "# tags: [a]\n", nil},
		{"unclosed", "---\ntags: [a]\n", nil},
		{"bad yaml", "---\ntags: [a\n---\n", nil},
		{"no tags", "---\ntitle: x\n---\n", nil},
	}
	for _, c := range cases {
		if got := parseTags([]byte(c.note)); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: parseTags = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestTags(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"b.md":         "---\ntags: [work, Ideas]\n---\n",
		"a/c.md":       "---\ntags: [ideas]\n---\n",
		"plain.md":     "no frontmatter",
		".hidden/x.md": "---\ntags: [secret]\n---\n",
		"notes.txt":    "---\ntags: [txt]\n---\n",
	}
	for name, body := range files {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	got, err := (&NotesService{root: root}).Tags()
	if err != nil {
		t.Fatal(err)
	}
	want := []Tag{
		{Name: "ideas", Notes: []string{"a/c.md", "b.md"}},
		{Name: "work", Notes: []string{"b.md"}},
	}
	// The first spelling seen wins; WalkDir visits a/c.md before b.md.
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Tags() = %+v, want %+v", got, want)
	}
}
