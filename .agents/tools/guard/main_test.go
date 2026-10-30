package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestVersionAndBumpKeepEveryOtherByte(t *testing.T) {
	raw := []byte("{\n  \"version\": 3,\n  \"remote\": \"r\"\n}\n")
	v, err := version(raw)
	if err != nil || v != 3 {
		t.Fatalf("version = %d, %v", v, err)
	}
	if got := string(bump(raw, 4)); got != "{\n  \"version\": 4,\n  \"remote\": \"r\"\n}\n" {
		t.Fatalf("bump = %q", got)
	}
	for _, bad := range []string{"{}", `{"version": 1, "x": {"version": 2}}`} {
		if _, err := version([]byte(bad)); err == nil {
			t.Errorf("%s: no error", bad)
		}
	}
}

func git(t *testing.T, root string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v %s", args, err, out)
	}
	return string(out)
}

func write(t *testing.T, root, rel, text string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	os.MkdirAll(filepath.Dir(p), 0o755)
	if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRunBumpsOnMainOnly(t *testing.T) {
	root := t.TempDir()
	git(t, root, "init", "-q", "-b", "main")
	for _, m := range modules {
		write(t, root, m+"/go.mod", "module x\n\ngo 1.22\n")
		write(t, root, m+"/x.go", "package x\n")
	}
	write(t, root, configPath, "{\n  \"version\": 3\n}\n")
	var out bytes.Buffer
	if err := run(root, true, &out); err != nil || !strings.HasSuffix(out.String(), "version 3\n") {
		t.Fatalf("check: %v\n%s", err, out.String())
	}
	out.Reset()
	if err := run(root, false, &out); err != nil || !strings.HasSuffix(out.String(), "version 3 -> 4, staged\n") {
		t.Fatalf("bump: %v\n%s", err, out.String())
	}
	if staged := git(t, root, "diff", "--cached", "--name-only"); !strings.Contains(staged, configPath) {
		t.Fatalf("staged = %q", staged)
	}
	git(t, root, "checkout", "-q", "-b", "other")
	out.Reset()
	if err := run(root, false, &out); err != nil || !strings.HasSuffix(out.String(), "version 4 (branch other, no bump)\n") {
		t.Fatalf("off main: %v\n%s", err, out.String())
	}
	if got, err := findRoot(filepath.Join(root, ".agents")); err == nil || got != "" {
		t.Fatalf("findRoot outside: %q, %v", got, err)
	}
	if got, err := findRoot(root); err != nil || got != root {
		t.Fatalf("findRoot: %q, %v", got, err)
	}
}

func TestRunReportsAFailingModule(t *testing.T) {
	root := t.TempDir()
	git(t, root, "init", "-q", "-b", "main")
	for _, m := range modules {
		write(t, root, m+"/go.mod", "module x\n\ngo 1.22\n")
		write(t, root, m+"/x.go", "package x\n")
	}
	write(t, root, modules[0]+"/x.go", "package x\n\nfunc broken() { return 1 }\n")
	write(t, root, configPath, "{\"version\": 1}\n")
	if err := run(root, true, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "go vet in "+modules[0]) {
		t.Fatalf("err = %v", err)
	}
}
