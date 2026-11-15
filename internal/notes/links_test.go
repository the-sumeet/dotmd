package notes

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveLink(t *testing.T) {
	root := t.TempDir()
	for _, f := range []string{
		"A.md", "My Note.md", "Plan.md", "Projects/B.md", "Projects/Deep/C.markdown", "image.png", "Folder.md/x.md",
		"Projects.md", "C# Tips.md", "What?.md", "100% done.md", "Mixed/Case Note.md",
	} {
		p := filepath.Join(root, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	s := &NotesService{root: root}

	cases := []struct {
		from, href string
		want       string // "" means an error is expected
	}{
		{"A.md", "Plan.md", "Plan.md"},
		{"A.md", "./Plan.md", "Plan.md"},
		{"A.md", "Projects/B.md", "Projects/B.md"},
		{"A.md", "Projects/Deep/C.markdown", "Projects/Deep/C.markdown"},
		{"Projects/B.md", "../A.md", "A.md"},
		{"Projects/Deep/C.markdown", "../../A.md", "A.md"},
		{"Projects/Deep/C.markdown", "../B.md", "Projects/B.md"},
		{"Projects/B.md", "/A.md", "A.md"},
		{"Projects/B.md", "/Projects/Deep/C.markdown", "Projects/Deep/C.markdown"},
		{"A.md", "My%20Note.md", "My Note.md"},
		{"A.md", "My Note.md", "My Note.md"},
		{"A.md", "Plan.md#goals", "Plan.md"},
		{"A.md", "Plan.md?x=1", "Plan.md"},
		{"A.md", "Plan", "Plan.md"},
		{"Projects/B.md", "../My Note", "My Note.md"},
		{"", "Plan.md", "Plan.md"},
		{"A.md", "Missing.md", ""},
		{"A.md", "Missing", ""},
		{"A.md", "#heading", ""},
		{"A.md", "", ""},
		{"A.md", "../A.md", ""},
		{"Projects/B.md", "../../A.md", ""},
		{"A.md", "Folder.md", ""},
		{"A.md", "image.png", ""},
		{"A.md", "Projects", "Projects.md"},
		{"A.md", "C%23%20Tips.md", "C# Tips.md"},
		{"A.md", "What%3F.md", "What?.md"},
		{"A.md", "100%25 done.md", "100% done.md"},
		{"A.md", "100% done.md", "100% done.md"},
	}
	for _, c := range cases {
		got, err := s.ResolveLink(c.from, c.href)
		if c.want == "" {
			if err == nil {
				t.Errorf("ResolveLink(%q, %q) = %q, want error", c.from, c.href, got)
			}
			continue
		}
		if err != nil || got != c.want {
			t.Errorf("ResolveLink(%q, %q) = %q, %v; want %q", c.from, c.href, got, err, c.want)
		}
	}
}

func TestResolveLinkCase(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "Mixed", "Case Note.md")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	s := &NotesService{root: root}
	got, err := s.ResolveLink("A.md", "mixed/case note.md")
	if _, statErr := os.Stat(filepath.Join(root, "mixed", "case note.md")); statErr != nil {
		// Case-sensitive filesystem: the differently cased link has no target.
		if err == nil {
			t.Errorf("ResolveLink = %q, want error on a case-sensitive filesystem", got)
		}
		return
	}
	if err != nil || got != "Mixed/Case Note.md" {
		t.Errorf("ResolveLink = %q, %v; want the on-disk name %q", got, err, "Mixed/Case Note.md")
	}
}

func TestLinkPathResolves(t *testing.T) {
	root := t.TempDir()
	notes := []string{"A.md", "Sub/C# Tips.md", "What?.md", "100% done.md", "Sub/Deep/x.md", "Re: plan.md"}
	for _, f := range notes {
		p := filepath.Join(root, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	s := &NotesService{root: root}
	for _, from := range notes {
		for _, to := range notes {
			href := s.LinkPath(from, to)
			if got, err := s.ResolveLink(from, href); err != nil || got != to {
				t.Errorf("ResolveLink(%q, LinkPath(...) = %q) = %q, %v; want %q", from, href, got, err, to)
			}
		}
	}
}

func TestLinkPath(t *testing.T) {
	s := &NotesService{root: t.TempDir()}
	cases := []struct{ from, to, want string }{
		{"A.md", "Plan.md", "Plan.md"},
		{"A.md", "A.md", "A.md"},
		{"A.md", "Projects/B.md", "Projects/B.md"},
		{"Projects/B.md", "A.md", "../A.md"},
		{"Projects/B.md", "Projects/Other.md", "Other.md"},
		{"Projects/Deep/C.md", "Notes/My Note.md", "../../Notes/My Note.md"},
		{"A.md", "C# Tips.md", "C%23 Tips.md"},
		{"A.md", "What?.md", "What%3F.md"},
		{"A.md", "100% done.md", "100%25 done.md"},
		{"A.md", "Re: plan.md", "./Re: plan.md"},
		{"A.md", "Re: x/B.md", "./Re: x/B.md"},
		{"Sub/A.md", "Re: plan.md", "../Re: plan.md"},
		{"A.md", "Time 10:30.md", "Time 10:30.md"},
	}
	for _, c := range cases {
		if got := s.LinkPath(c.from, c.to); got != c.want {
			t.Errorf("LinkPath(%q, %q) = %q, want %q", c.from, c.to, got, c.want)
		}
	}
}
