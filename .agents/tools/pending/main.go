// pending: what work is pending, for whom — the specs /design owes, the code
// /implement owes, and the next /implement continue batch.
//
// Usage: go run -C .agents/tools ./pending [--batch] [--root <dir>]
//
//	--batch       print only the next batch (or what /design owes when nothing
//	              can be implemented) and the drift
//	--root <dir>  repository root, absolute (default: nearest directory at or
//	              above the working directory with AGENTS.md and perspectives/)
//
// Output, plain text:
//
//	# Design      perspective specs no technical spec serves; technical specs
//	              stale against a source, with the rules that changed and the
//	              ## Spec gap that parks one; dangling sources of technical
//	              specs, with the rules recorded for them
//	# Implement   the backlog, sources first: technical specs with no code node,
//	              and code nodes stale against a source, holding a dangling one,
//	              or missing a perspective node their technical specs serve —
//	              each with its rules (new, changed, removed, renamed; park when
//	              a ## Design gap holds it) and whether it runs or waits
//	# Next batch  the units /implement continue runs now, at most 5, then the
//	              units that wait and why
//	# Drift       metadata that does not match its content, missing or orphan
//	              metadata, invalid sources, files under src/ outside any node,
//	              duplicate titles, cycles, gaps in TODO.md that match no rule
//
// With nothing pending it prints "No pending work." and the drift. Exit 0
// with the answer; non-zero with the reason on stderr.
//
// It implements **Continue** of .agents/skills/implement/SKILL.md over the
// states of **Metadata** in the root AGENTS.md; they win over it.
package main

import (
	"fmt"
	"os"
	"strings"

	"tools/internal/graph"
	"tools/internal/repo"
)

const (
	usage = "usage: go run -C .agents/tools ./pending [--batch] [--root <dir>]"
	cap   = 5
)

func main() {
	batch := false
	root := ""
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--batch":
			batch = true
		case args[i] == "--root" && i+1 < len(args):
			i++
			root = args[i]
		default:
			fail("unknown argument: "+args[i]+"\n"+usage, 2)
		}
	}
	dir, err := repo.Root(root)
	if err != nil {
		fail(err.Error(), 1)
	}
	g, err := graph.Load(dir)
	if err != nil {
		fail(err.Error(), 1)
	}
	fmt.Println(render(g, batch))
}

func render(g *graph.Graph, batchOnly bool) string {
	var out []string
	design := g.Design()
	units := g.Units()
	drift := g.Drift()
	if len(design) == 0 && len(units) == 0 {
		out = append(out, "No pending work.")
	}
	if !batchOnly {
		if len(design) > 0 {
			out = append(out, "# Design", "")
			for _, line := range design {
				out = append(out, "- "+line)
			}
			out = append(out, "")
		}
		if len(units) > 0 {
			out = append(out, "# Implement", "")
			out = append(out, list(units)...)
			out = append(out, "")
		}
	}
	if len(units) > 0 {
		run, wait := g.Batch(cap)
		if len(run) > 0 {
			out = append(out, fmt.Sprintf("# Next batch: %d unit%s", len(run), plural(len(run))), "")
			out = append(out, list(run)...)
		} else {
			out = append(out, "# Next batch: none", "", "Every unit waits:")
		}
		if len(wait) > 0 {
			if len(run) > 0 {
				out = append(out, "", "Waiting:")
			}
			for _, u := range wait {
				out = append(out, fmt.Sprintf("- `%s` — %s", u.Node.Rel, u.Wait))
			}
		}
		out = append(out, "")
	} else if len(design) > 0 && batchOnly {
		out = append(out, "# Next batch: none", "", "Nothing to implement; /design owes:")
		for _, line := range design {
			out = append(out, "- "+line)
		}
		out = append(out, "")
	}
	if len(drift) > 0 {
		out = append(out, "# Drift", "")
		for _, line := range drift {
			out = append(out, "- "+line)
		}
	}
	return strings.TrimRight(strings.Join(out, "\n"), "\n")
}

// list renders units numbered, each with its rules: run, wait, or park per line.
func list(units []graph.Unit) []string {
	var out []string
	for i, u := range units {
		verb, reason := "run ", u.Reason
		if u.Wait != "" {
			verb, reason = "wait", u.Reason+" — waits: "+u.Wait
		}
		out = append(out, fmt.Sprintf("%d. `%s` — %s", i+1, u.Node.Rel, reason))
		for _, r := range u.Rules {
			if r.Parked != "" {
				out = append(out, fmt.Sprintf("   park %s — %s (%s) — parked: %s", r.Title, r.State, r.Source, r.Parked))
				continue
			}
			out = append(out, fmt.Sprintf("   %s %s — %s (%s)", verb, r.Title, r.State, r.Source))
		}
	}
	return out
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func fail(msg string, code int) {
	fmt.Fprintf(os.Stderr, "pending: %s\n", msg)
	os.Exit(code)
}
