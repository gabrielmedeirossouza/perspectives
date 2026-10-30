package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"tools/internal/graph"
)

func TestResolveNamesNodesMetadataFilesAndDirectories(t *testing.T) {
	root := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", root).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
	for rel, text := range map[string]string{
		"AGENTS.md":                "# Root\n",
		".sdd/.sdd-config.json":    `{"version": 0, "paths": {"specs": "specs", "src": "src", "technical": "src/specs"}}` + "\n",
		"specs/AGENTS.md":          "# P\n",
		"src/x/a.go":               "package x\n",
		"src/x/a.go.metadata.json": `{"version": 2}` + "\n",
		"src/x/.metadata.json":     `{"version": 2}` + "\n",
		"src/y/b.go":               "package y\n",
		"src/y/b.go.metadata.json": `{"version": 2}` + "\n",
		"specs/business/i/spec.md": "# I\n",
		"src/loose/c.go":           "package loose\n",
	} {
		p := filepath.Join(root, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(text), 0o644)
	}
	g, err := graph.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	rels := func(nodes []*graph.Node) string {
		var out []string
		for _, n := range nodes {
			out = append(out, n.Rel)
		}
		return strings.Join(out, " ")
	}
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"src/x", "./src/x/", filepath.Join(root, "src", "x"), "src/y/b.go"}, "src/x src/x/a.go src/y/b.go"},
		{[]string{"src/x/.metadata.json"}, "src/x"},
		{[]string{"src/x/a.go.metadata.json"}, "src/x/a.go"},
		{[]string{"src/y"}, "src/y/b.go"},
		{[]string{"src"}, "src/x src/x/a.go src/y/b.go"},
		{[]string{"specs/business/i"}, "specs/business/i/spec.md"},
		{[]string{"specs/business/i/spec.md"}, "specs/business/i/spec.md"},
	} {
		nodes, err := resolve(g, root, c.args, false, false)
		if err != nil || rels(nodes) != c.want {
			t.Errorf("resolve(%v) = %q %v, want %q", c.args, rels(nodes), err, c.want)
		}
	}
	for _, bad := range []string{"src/z", "src/x/nope.go", "src/y/.metadata.json", "src/x/a.go/.metadata.json"} {
		if _, err := resolve(g, root, []string{bad}, false, false); err == nil || !strings.Contains(err.Error(), "is no node") {
			t.Errorf("resolve(%s) = %v, want an error", bad, err)
		}
	}
	// --sync and --approve take one node per target.
	for _, one := range []string{"src/x/.metadata.json", "src/x/a.go", "src/y", "specs/business/i"} {
		if nodes, err := resolve(g, root, []string{one}, true, false); err != nil || len(nodes) != 1 {
			t.Errorf("single resolve(%s) = %v %v", one, rels(nodes), err)
		}
	}
	if _, err := resolve(g, root, []string{"src/x"}, true, false); err == nil || !strings.Contains(err.Error(), "`src/x` names 2 nodes; --sync and --approve take one node per target: a file path, or src/x/.metadata.json for the directory node") {
		t.Errorf("single resolve(src/x) = %v", err)
	}
	if nodes, err := resolve(g, root, []string{"src/loose"}, false, true); err != nil || len(nodes) != 0 {
		t.Errorf("a check of a directory holding only files in no node = %v %v", nodes, err)
	}
	if _, err := resolve(g, root, []string{"src/loose"}, false, false); err == nil {
		t.Errorf("a write of a directory with no node resolved")
	}
}
