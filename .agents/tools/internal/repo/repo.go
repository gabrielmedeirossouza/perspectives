// Package repo locates the repository root and renders paths relative to it.
package repo

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// Root returns the repository root: flagRoot when given, otherwise the
// nearest directory at or above the working directory that holds AGENTS.md
// and perspectives/. go run -C .agents/tools starts two levels below it.
func Root(flagRoot string) (string, error) {
	if flagRoot != "" {
		abs, err := filepath.Abs(flagRoot)
		if err != nil {
			return "", err
		}
		if !isRoot(abs) {
			return "", errors.New("no AGENTS.md and perspectives/ under " + abs)
		}
		return abs, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for dir := cwd; ; dir = filepath.Dir(dir) {
		if isRoot(dir) {
			return dir, nil
		}
		if filepath.Dir(dir) == dir {
			return "", errors.New("no repository root (AGENTS.md + perspectives/) at or above " + cwd)
		}
	}
}

func isRoot(dir string) bool {
	if st, err := os.Stat(filepath.Join(dir, "AGENTS.md")); err != nil || st.IsDir() {
		return false
	}
	st, err := os.Stat(filepath.Join(dir, "perspectives"))
	return err == nil && st.IsDir()
}

// Rel returns path relative to root, with forward slashes.
func Rel(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

// Clean normalizes a root-relative path a user typed: forward slashes, no
// leading "./", no trailing slash.
func Clean(p string) string {
	p = filepath.ToSlash(filepath.Clean(p))
	return strings.TrimSuffix(strings.TrimPrefix(p, "./"), "/")
}
