package repo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tools/internal/config"
)

func TestRootWalksUpToTheConfig(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(root, filepath.FromSlash(config.File))
	os.MkdirAll(filepath.Dir(cfg), 0o755)
	os.WriteFile(cfg, []byte("{}\n"), 0o644)
	deep := filepath.Join(root, "a", "b")
	os.MkdirAll(deep, 0o755)
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(deep); err != nil {
		t.Fatal(err)
	}
	if got, err := Root(""); err != nil || got != root {
		t.Fatalf("Root = %q, %v; want %q", got, err, root)
	}
	if got, err := Root(root); err != nil || got != root {
		t.Fatalf("Root(flag) = %q, %v", got, err)
	}
	if _, err := Root(deep); err == nil || !strings.Contains(err.Error(), config.File) {
		t.Fatalf("Root(deep) = %v; want an error naming the config", err)
	}
}

func TestRootFailsWithoutTheConfig(t *testing.T) {
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if _, err := Root(""); err == nil || !strings.Contains(err.Error(), "at or above") {
		t.Fatalf("Root = %v", err)
	}
}

func TestClean(t *testing.T) {
	if Clean("./src/x/") != "src/x" || Clean("src") != "src" {
		t.Fatal("Clean")
	}
}
