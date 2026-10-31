// pending: what work is pending, and for whom? Flags and output: .agents/tools/AGENTS.md.
package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"tools/internal/graph"
	"tools/internal/meta"
	"tools/internal/repo"
)

const (
	usage = "usage: go run -C .agents/tools ./pending [--batch | --audit [--all] [node|dir...]] [--root <dir>]"
	cap   = 5
)

func main() {
	batch, audit, all := false, false, false
	root := ""
	var dirs []string
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--batch":
			batch = true
		case args[i] == "--audit":
			audit = true
		case args[i] == "--all":
			all = true
		case args[i] == "--root" && i+1 < len(args):
			i++
			root = args[i]
		case strings.HasPrefix(args[i], "-"):
			fail("unknown argument: "+args[i]+"\n"+usage, 2)
		default:
			dirs = append(dirs, args[i])
		}
	}
	if (batch && audit) || (!audit && (all || len(dirs) > 0)) {
		fail("--all and node|dir... go with --audit, which excludes --batch\n"+usage, 2)
	}
	dir, err := repo.Root(root)
	if err != nil {
		fail(err.Error(), 1)
	}
	g, err := graph.Load(dir)
	if err != nil {
		fail(err.Error(), 1)
	}
	if audit {
		targets, err := codeNodes(g, dirs)
		if err != nil {
			fail(err.Error(), 1)
		}
		fmt.Println(renderAudit(g, all, targets))
		return
	}
	fmt.Println(render(g, batch))
}

// codeNodes resolves targets to the code nodes at or under each; every code
// node when none is given.
func codeNodes(g *graph.Graph, dirs []string) ([]*graph.Node, error) {
	if len(dirs) == 0 {
		return g.CodeNodes(g.Paths.Src), nil
	}
	seen := map[*graph.Node]bool{}
	var out []*graph.Node
	for _, d := range dirs {
		rel := repo.Target(g.Root, d)
		nodes := g.CodeNodes(rel)
		if len(nodes) == 0 {
			return nil, fmt.Errorf("no code node at or under `%s`", rel)
		}
		for _, n := range nodes {
			if !seen[n] {
				seen[n] = true
				out = append(out, n)
			}
		}
	}
	return out, nil
}

func render(g *graph.Graph, batchOnly bool) string {
	var out []string
	design := g.Design()
	units := g.Units()
	drift := g.Drift()
	audit := auditLines(g.Audits())
	if len(design) == 0 && len(units) == 0 {
		if len(audit) == 0 && len(drift) == 0 {
			out = append(out, "No pending work.")
		} else {
			out = append(out, "Nothing stale, dangling, or unconsumed.")
		}
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
			if batchOnly {
				out = append(out, list(run)...)
			} else {
				for _, u := range run {
					out = append(out, fmt.Sprintf("- `%s`", u.Rel()))
				}
			}
		} else {
			waits := "Every unit waits."
			if batchOnly {
				waits = "Every unit waits:"
			}
			out = append(out, "# Next batch: none", "", waits)
		}
		if batchOnly && len(wait) > 0 {
			if len(run) > 0 {
				out = append(out, "", "Waiting:")
			}
			for _, u := range wait {
				out = append(out, fmt.Sprintf("- `%s` — %s", u.Rel(), u.Wait))
			}
		}
		out = append(out, "")
	} else if len(design) > 0 && batchOnly {
		out = append(out, "# Next batch: none", "", "Nothing to implement; /sdd-design owes:")
		for _, line := range design {
			out = append(out, "- "+line)
		}
		out = append(out, "")
	}
	if len(audit) > 0 && !batchOnly {
		out = append(out, "# Audit", "")
		for _, line := range audit {
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

// auditLines names every code node with no approval at its current content
// and where it stands: owed, blocked, after /sdd-implement, or parked.
func auditLines(states []graph.AuditState) []string {
	var out []string
	for _, s := range states {
		if s.Audited > 0 {
			continue
		}
		out = append(out, standing(s, "owes an audit"))
	}
	return out
}

// standing renders one node's audit standing after its path: blocking drift
// first, then a parked gap, then the /sdd-implement unit.
func standing(s graph.AuditState, owed string) string {
	switch {
	case len(s.Blocked) > 0:
		return fmt.Sprintf("`%s` %s — blocked: %s", s.Node.Rel, owed, strings.Join(s.Blocked, "; "))
	case s.Parked != "":
		return fmt.Sprintf("`%s` parked: %s (/sdd-design)", s.Node.Rel, s.Parked)
	case s.Unit != "":
		return fmt.Sprintf("`%s` %s — after /sdd-implement: %s", s.Node.Rel, owed, s.Unit)
	}
	return fmt.Sprintf("`%s` %s (/sdd-audit)", s.Node.Rel, owed)
}

// renderAudit prints the targeted code nodes grouped by directory — one
// judge per directory — with the standing of each node not judged and a
// dossier for each one a judge reads now, then the drift.
func renderAudit(g *graph.Graph, all bool, targets []*graph.Node) string {
	groups := map[string][]*graph.Node{}
	var dirs []string
	for _, n := range targets {
		dir := n.UnitDir()
		if _, ok := groups[dir]; !ok {
			dirs = append(dirs, dir)
		}
		groups[dir] = append(groups[dir], n)
	}
	sort.Strings(dirs)
	var out []string
	for _, dir := range dirs {
		var lines, dossiers []string
		for _, n := range groups[dir] {
			s := g.AuditOf(n)
			judge := s.Owed() || (all && len(s.Blocked) == 0 && s.Unit == "")
			switch {
			case judge:
				dossiers = append(dossiers, dossier(g.Dossier(n))...)
			case len(s.Blocked) > 0 || s.Unit != "" || s.Parked != "":
				lines = append(lines, standing(s, "not judged"))
			default:
				lines = append(lines, fmt.Sprintf("`%s` audited %d; skipped (--all re-audits)", n.Rel, s.Audited))
			}
		}
		out = append(out, fmt.Sprintf("## `%s/`", dir), "")
		for _, line := range lines {
			out = append(out, "- "+line)
		}
		if len(lines) > 0 {
			out = append(out, "")
		}
		out = append(out, dossiers...)
	}
	if len(out) == 0 {
		out = append(out, "No audit owed.")
	} else {
		out = append([]string{"# Audit", ""}, out...)
	}
	if drift := g.Drift(); len(drift) > 0 {
		out = append(out, "# Drift", "")
		for _, line := range drift {
			out = append(out, "- "+line)
		}
	}
	return strings.TrimRight(strings.Join(out, "\n"), "\n")
}

// dossier renders what a judge needs about one code node.
func dossier(d graph.Dossier) []string {
	n := d.State.Node
	out := []string{fmt.Sprintf("### `%s` — audited %d", n.Rel, d.State.Audited), ""}
	if d.State.Parked != "" {
		out = append(out, "Parked: "+d.State.Parked+" (/sdd-design)", "")
	}
	if n.IsDir {
		out = append(out, "Files:")
		var links []string
		for _, f := range d.Files {
			to := "none"
			if specs := d.Links[f]; len(specs) > 0 {
				to = "`" + strings.Join(specs, "`, `") + "`"
			}
			links = append(links, f+" → "+to)
		}
		out = append(out, bullets(links)...)
	} else {
		out = append(out, "File: "+n.Rel)
	}
	out = append(out, "", "Listed technical specs:")
	var tech []string
	for _, t := range d.Technical {
		line := "`" + t.Rel + "`"
		if len(t.Order) > 0 {
			line += ": " + meta.Quoted(t.Order)
		}
		tech = append(tech, line)
	}
	out = append(out, bullets(tech)...)
	out = append(out, "", "Listed perspective specs:")
	var persp []string
	for _, p := range d.Perspectives {
		persp = append(persp, "`"+p.Rel+"`")
	}
	out = append(out, bullets(persp)...)
	out = append(out, "", "Candidates:")
	var cands []string
	for _, c := range d.Candidates {
		var why []string
		if c.Names {
			why = append(why, "names this node")
		}
		if c.Global {
			why = append(why, "global")
		}
		if c.Purpose != "" {
			why = append(why, c.Purpose)
		}
		cands = append(cands, "`"+c.Node.Rel+"` — "+strings.Join(why, "; "))
	}
	out = append(out, bullets(cands)...)
	if len(d.DropPerspectives) > 0 {
		out = append(out, "", "Perspectives to drop:")
		out = append(out, bullets(d.DropPerspectives)...)
	}
	return append(out, "")
}

func bullets(items []string) []string {
	if len(items) == 0 {
		return []string{"- none"}
	}
	out := make([]string, len(items))
	for i, s := range items {
		out[i] = "- " + s
	}
	return out
}

// list renders units numbered, each with its rules: run, wait, or park per
// line; a node that waits inside a unit that runs is listed and skipped.
func list(units []graph.Unit) []string {
	var out []string
	for i, u := range units {
		verb, reason := "run ", u.Reason
		if u.Wait != "" {
			verb, reason = "wait", u.Reason+" — waits: "+u.Wait
		}
		out = append(out, fmt.Sprintf("%d. `%s` — %s", i+1, u.Rel(), reason))
		runnable := map[[2]string]bool{}
		if u.Wait == "" {
			for _, nu := range u.Nodes {
				if nu.Wait != "" {
					out = append(out, fmt.Sprintf("   wait %s — %s", nu.Node.Label(), nu.Wait))
					continue
				}
				for _, r := range nu.Rules {
					runnable[[2]string{r.Source, r.Title}] = true
				}
			}
		}
		for _, r := range u.Rules {
			if r.Parked != "" {
				out = append(out, fmt.Sprintf("   park %s — %s (%s) — parked: %s", r.Title, r.State, r.Source, r.Parked))
				continue
			}
			rv := verb
			if u.Wait == "" && len(u.Nodes) > 0 && !runnable[[2]string{r.Source, r.Title}] {
				rv = "wait"
			}
			out = append(out, fmt.Sprintf("   %s %s — %s (%s)", rv, r.Title, r.State, r.Source))
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
