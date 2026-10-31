// Package repo locates the repository root and renders paths relative to it.
package repo

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"tools/internal/config"
)

// Root returns the repository root: flagRoot when given, otherwise the
// nearest directory at or above the working directory that holds
// .sdd/.sdd-config.json. go run -C .agents/tools starts two levels below it.
func Root(flagRoot string) (string, error) {
	if flagRoot != "" {
		abs, err := filepath.Abs(flagRoot)
		if err != nil {
			return "", err
		}
		if !isRoot(abs) {
			return "", errors.New("no " + config.File + " under " + abs)
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
			return "", errors.New("no repository root (" + config.File + ") at or above " + cwd)
		}
	}
}

func isRoot(dir string) bool {
	st, err := os.Stat(filepath.Join(dir, filepath.FromSlash(config.File)))
	return err == nil && st.Mode().IsRegular()
}

// Rel returns path relative to root, with forward slashes.
func Rel(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

// Target turns a path a user typed, absolute or root-relative, into a clean
// root-relative path.
func Target(root, p string) string {
	if filepath.IsAbs(p) {
		p = Rel(root, p)
	}
	return Clean(p)
}

// Git runs git in dir and returns its standard output.
func Git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s in %s: %s", args[0], dir, strings.TrimSpace(errb.String()))
	}
	return out.String(), nil
}

// Clean normalizes a root-relative path a user typed: forward slashes, no
// leading "./", no trailing slash.
func Clean(p string) string {
	p = filepath.ToSlash(filepath.Clean(p))
	return strings.TrimSuffix(strings.TrimPrefix(p, "./"), "/")
}
