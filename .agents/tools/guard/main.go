// guard: is this framework commit sound, and which version does it make? Flags and output: .agents/tools/AGENTS.md.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const (
	usage      = "usage: go run -C .agents/tools ./guard [--check] [--root <dir>]"
	configPath = ".sdd/templates/.sdd/.sdd-config.json"
	marker     = ".sdd/templates"
)

// modules are the Go modules the guard vets and tests, root-relative.
var modules = []string{".sdd/templates/.agents/tools", ".agents/tools"}

var versionRe = regexp.MustCompile(`"version"\s*:\s*(\d+)`)

func main() {
	check, root := false, ""
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--check":
			check = true
		case args[i] == "--root" && i+1 < len(args):
			i++
			root = args[i]
		default:
			fail("unknown argument: "+args[i]+"\n"+usage, 2)
		}
	}
	root, err := findRoot(root)
	if err != nil {
		fail(err.Error(), 1)
	}
	if err := run(root, check, os.Stdout); err != nil {
		fail(err.Error(), 1)
	}
}

// findRoot returns flagRoot when it holds the marker, otherwise the nearest
// directory at or above the working directory that does.
func findRoot(flagRoot string) (string, error) {
	if flagRoot != "" {
		abs, err := filepath.Abs(flagRoot)
		if err != nil {
			return "", err
		}
		if !isRoot(abs) {
			return "", errors.New("no " + marker + " under " + abs)
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
			return "", errors.New("no framework root (" + marker + ") at or above " + cwd)
		}
	}
}

func isRoot(dir string) bool {
	st, err := os.Stat(filepath.Join(dir, filepath.FromSlash(marker)))
	return err == nil && st.IsDir()
}

// run vets and tests every module, then reports or bumps the version.
func run(root string, check bool, w io.Writer) error {
	for _, m := range modules {
		dir := filepath.Join(root, filepath.FromSlash(m))
		for _, verb := range []string{"vet", "test"} {
			cmd := exec.Command("go", "-C", dir, verb, "./...")
			if out, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("go %s in %s failed:\n%s", verb, m, strings.TrimSpace(string(out)))
			}
		}
		fmt.Fprintf(w, "ok %s\n", m)
	}
	file := filepath.Join(root, filepath.FromSlash(configPath))
	raw, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	v, err := version(raw)
	if err != nil {
		return fmt.Errorf("%s: %v", configPath, err)
	}
	if check {
		fmt.Fprintf(w, "version %d\n", v)
		return nil
	}
	if b := branch(root); b != "main" {
		fmt.Fprintf(w, "version %d (branch %s, no bump)\n", v, b)
		return nil
	}
	if committed, ok := headVersion(root); ok && v > committed {
		fmt.Fprintf(w, "version %d already bumped from %d, staged\n", v, committed)
		return stage(root)
	}
	if err := os.WriteFile(file, bump(raw, v+1), 0o644); err != nil {
		return err
	}
	if err := stage(root); err != nil {
		return err
	}
	fmt.Fprintf(w, "version %d -> %d, staged\n", v, v+1)
	return nil
}

// stage adds the config to the index.
func stage(root string) error {
	if out, err := exec.Command("git", "-C", root, "add", "--", configPath).CombinedOutput(); err != nil {
		return fmt.Errorf("git add: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

// headVersion is the version the last commit holds; ok is false without one.
func headVersion(root string) (int, bool) {
	out, err := exec.Command("git", "-C", root, "show", "HEAD:"+configPath).Output()
	if err != nil {
		return 0, false
	}
	v, err := version(out)
	return v, err == nil
}

// version reads the one "version" field of the config.
func version(raw []byte) (int, error) {
	m := versionRe.FindAllSubmatchIndex(raw, -1)
	if len(m) != 1 {
		return 0, fmt.Errorf("expected one \"version\" field, found %d", len(m))
	}
	return strconv.Atoi(string(raw[m[0][2]:m[0][3]]))
}

// bump rewrites the "version" field to v, byte for byte otherwise.
func bump(raw []byte, v int) []byte {
	m := versionRe.FindSubmatchIndex(raw)
	out := append([]byte(nil), raw[:m[2]]...)
	out = append(out, strconv.Itoa(v)...)
	return append(out, raw[m[3]:]...)
}

// branch is the current branch, or "detached" when HEAD is detached.
func branch(root string) string {
	out, err := exec.Command("git", "-C", root, "symbolic-ref", "--short", "-q", "HEAD").Output()
	if err != nil {
		return "detached"
	}
	return strings.TrimSpace(string(out))
}

func fail(msg string, code int) {
	fmt.Fprintf(os.Stderr, "guard: %s\n", msg)
	os.Exit(code)
}
