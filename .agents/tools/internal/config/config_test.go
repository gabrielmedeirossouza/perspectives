package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, root, text string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(File))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestMissingFileIsTheDefaultLayout(t *testing.T) {
	c, err := Load(t.TempDir())
	if err != nil || c.Paths != Default().Paths || c.Version != 0 {
		t.Fatalf("Load = %+v, %v", c, err)
	}
}

func TestLoadReadsAndCleansTheLayout(t *testing.T) {
	root := t.TempDir()
	write(t, root, `{"version": 3, "remote": "r", "product": "p", "paths": {"specs": "./product/", "src": "apps/code", "technical": "apps/design"}, "perspectives": [{"name": "player", "from": "user"}, {"name": "coach"}], "skills": ["sdd-spec"]}`)
	c, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	want := Paths{Specs: "product", Src: "apps/code", Technical: "apps/design"}
	if c.Version != 3 || c.Remote != "r" || c.Product != "p" || c.Paths != want || len(c.Perspectives) != 2 || c.Perspectives[1].From != "" || len(c.Skills) != 1 {
		t.Fatalf("Load = %+v", c)
	}
}

func TestEmptyPathsTakeTheirDefault(t *testing.T) {
	root := t.TempDir()
	write(t, root, `{"version": 1, "paths": {"specs": "product"}}`)
	c, err := Load(root)
	if err != nil || c.Paths != (Paths{Specs: "product", Src: "src", Technical: "src/specs"}) {
		t.Fatalf("Load = %+v, %v", c.Paths, err)
	}
}

func TestLoadRefusesInvalidLayouts(t *testing.T) {
	for _, tc := range []struct{ text, want string }{
		{`{"version": 1, "path": {}}`, "unknown field"},
		{`{"version": "1"}`, "invalid JSON"},
		{`{"paths": {"specs": "/abs"}}`, "not a directory inside"},
		{`{"paths": {"specs": "../up"}}`, "not a directory inside"},
		{`{"paths": {"specs": "src/specs"}}`, "overlap"},
		{`{"paths": {"specs": "x", "src": "x/code"}}`, "overlap"},
		{`{"paths": {"technical": "specs/tech"}}`, "overlap"},
		{`{"paths": {"technical": "src"}}`, "holds paths.src"},
		{`{"paths": {"src": "design/code", "technical": "design"}}`, "holds paths.src"},
	} {
		root := t.TempDir()
		write(t, root, tc.text)
		if _, err := Load(root); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: err = %v, want %q", tc.text, err, tc.want)
		}
	}
}

func TestUnder(t *testing.T) {
	if !Under("src", "src") || !Under("src/a/b", "src") || Under("srcx", "src") || Under("a/src", "src") {
		t.Fatal("Under")
	}
}
