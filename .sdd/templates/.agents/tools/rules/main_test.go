package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, root, rel, text string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestGenerateAndDetectDrift(t *testing.T) {
	root := t.TempDir()
	law := "# UNIVERSAL RULES\n\n- Be direct.\n\n---\n\n# Project\n\nLocal law.\n"
	write(t, root, "AGENTS.md", law)
	for _, e := range editors {
		var out bytes.Buffer
		if err := run(root, e.name, &out); err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(out.String(), "---\n"+e.activation+"\n---\n") ||
			!strings.Contains(out.String(), "- Be direct.\n") || strings.Contains(out.String(), "Local law.") {
			t.Fatalf("%s: %s", e.name, out.String())
		}
		write(t, root, e.path, out.String())
	}
	if err := run(root, "", &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	write(t, root, "AGENTS.md", strings.Replace(law, "Be direct.", "Avoid repetition.", 1))
	err := run(root, "", &bytes.Buffer{})
	for _, e := range editors {
		if err == nil || !strings.Contains(err.Error(), e.path+" missing or stale") {
			t.Fatalf("changed source: %v", err)
		}
	}
	if err := os.Remove(filepath.Join(root, filepath.FromSlash(editors[0].path))); err != nil {
		t.Fatal(err)
	}
	if err := run(root, "", &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), editors[0].path) {
		t.Fatalf("missing rule: %v", err)
	}
}

func TestRejectInvalidSource(t *testing.T) {
	for _, raw := range []string{
		"# Project\n",
		"# UNIVERSAL RULES\n\nBe direct.\n",
		"# UNIVERSAL RULES\n\n---\n",
	} {
		if _, err := source([]byte(raw)); err == nil {
			t.Errorf("accepted %q", raw)
		}
	}
}

func TestFindRoot(t *testing.T) {
	for _, marker := range []string{".sdd/.sdd-config.json", ".sdd/templates/AGENTS.md"} {
		root := t.TempDir()
		write(t, root, marker, "")
		if got, err := findRoot(root); err != nil || got != root {
			t.Fatalf("%s: %q, %v", marker, got, err)
		}
		if _, err := findRoot(filepath.Join(root, ".sdd")); err == nil {
			t.Fatal("explicit root searched its parent")
		}
	}
}
