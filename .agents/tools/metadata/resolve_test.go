package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"tools/internal/graph"
)

func TestResolveListsEachNodeOnce(t *testing.T) {
	root := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", root).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
	for rel, text := range map[string]string{
		"AGENTS.md":              "# Root\n",
		"perspectives/AGENTS.md": "# P\n",
		"src/x/a.go":             "package x\n",
		"src/x/.metadata.json":   `{"version": 1}` + "\n",
		"src/y/.metadata.json":   `{"version": 1}` + "\n",
	} {
		p := filepath.Join(root, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(text), 0o644)
	}
	g, err := graph.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := resolve(g, root, []string{"src/x", "./src/x/", filepath.Join(root, "src", "x"), "src/y"})
	if err != nil || len(nodes) != 2 || nodes[0].Rel != "src/x" || nodes[1].Rel != "src/y" {
		t.Fatalf("resolve = %v %v", nodes, err)
	}
	if _, err := resolve(g, root, []string{"src"}); err == nil {
		t.Fatal("a directory that is no node resolved")
	}
}
