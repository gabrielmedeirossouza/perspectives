// rules: do the always-active editor rules match UNIVERSAL RULES in the root
// AGENTS.md? With --format antigravity|cursor, print one generated rule instead.
//
// Usage: go run -C .agents/tools ./rules [--format antigravity|cursor] [--root <dir>]
//
// --root defaults to the nearest ancestor holding .sdd/.sdd-config.json (a
// project) or .sdd/templates (the framework). No file is written: redirect
// --format output to the corresponding rule file when regenerating it.
// Output: ok <path> per matching rule, or the rule body with --format. Exit 1
// on a missing or stale rule or an invalid source; 2 on a usage error.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type editor struct {
	name, path, activation string
}

var editors = []editor{
	{"antigravity", ".agents/rules/universal.md", "trigger: always_on"},
	{"cursor", ".cursor/rules/universal.mdc", "alwaysApply: true"},
}

func main() {
	format := flag.String("format", "", "print a rule: antigravity or cursor")
	root := flag.String("root", "", "repository root")
	flag.Parse()
	if flag.NArg() != 0 {
		flag.Usage()
		os.Exit(2)
	}
	if *format != "" && *format != "antigravity" && *format != "cursor" {
		fmt.Fprintln(os.Stderr, "rules: --format must be antigravity or cursor")
		os.Exit(2)
	}
	dir, err := findRoot(*root)
	if err == nil {
		err = run(dir, *format, os.Stdout)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "rules:", err)
		os.Exit(1)
	}
}

func findRoot(root string) (string, error) {
	explicit := root != ""
	if !explicit {
		var err error
		root, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	for dir := root; ; dir = filepath.Dir(dir) {
		for _, marker := range []string{".sdd/.sdd-config.json", ".sdd/templates"} {
			if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(marker))); err == nil {
				return dir, nil
			}
		}
		if explicit || filepath.Dir(dir) == dir {
			return "", fmt.Errorf("no project or framework root under or above %s", root)
		}
	}
}

func source(raw []byte) (string, error) {
	const heading = "# UNIVERSAL RULES\n\n"
	text := string(raw)
	if !strings.HasPrefix(text, heading) {
		return "", errors.New("AGENTS.md must start with # UNIVERSAL RULES followed by a blank line")
	}
	end := strings.Index(text, "\n---\n")
	if end < 0 {
		return "", errors.New("UNIVERSAL RULES in AGENTS.md must end with a --- separator")
	}
	if end < len(heading) || strings.TrimSpace(text[len(heading):end]) == "" {
		return "", errors.New("UNIVERSAL RULES in AGENTS.md is empty")
	}
	return strings.TrimSpace(text[:end]) + "\n", nil
}

func render(e editor, body string) []byte {
	return []byte("---\n" + e.activation + "\n---\n\n" +
		"<!-- Generated from UNIVERSAL RULES in root AGENTS.md; regenerate with go run -C .agents/tools ./rules --format " + e.name + ". -->\n\n" + body)
}

func run(root, format string, out io.Writer) error {
	raw, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		return err
	}
	body, err := source(raw)
	if err != nil {
		return err
	}
	var problems []error
	for _, e := range editors {
		want := render(e, body)
		if format != "" {
			if format == e.name {
				_, err := out.Write(want)
				return err
			}
			continue
		}
		got, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(e.path)))
		if err != nil || !bytes.Equal(got, want) {
			problems = append(problems, fmt.Errorf("%s missing or stale; regenerate with go run -C .agents/tools ./rules --format %s > %s (from the repository root)", e.path, e.name, e.path))
			continue
		}
		fmt.Fprintf(out, "ok %s\n", e.path)
	}
	return errors.Join(problems...)
}
