package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"tools/internal/graph"
)

func TestBatchModeNamesDesignWorkWhenNothingCanBeImplemented(t *testing.T) {
	root := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", root).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
	for rel, text := range map[string]string{
		"AGENTS.md":                            "# Root\n",
		"perspectives/AGENTS.md":               "# P\n",
		"perspectives/business/invite/spec.md": "# Invite\n\n## S\n\n### R\n\nx\n",
	} {
		p := filepath.Join(root, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(text), 0o644)
	}
	g, err := graph.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	out := render(g, true)
	for _, want := range []string{"# Next batch: none", "Nothing to implement; /design owes:", "- `perspectives/business/invite` has no technical spec", "# Drift"} {
		if !strings.Contains(out, want) {
			t.Fatalf("batch output lacks %q:\n%s", want, out)
		}
	}
	if full := render(g, false); !strings.Contains(full, "# Design") || strings.Contains(full, "# Next batch") {
		t.Fatalf("full output:\n%s", full)
	}
}
