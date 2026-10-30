// pending: what work is pending, for whom — the specs /sdd-design owes, the code
// /sdd-implement owes, the next /sdd-implement continue batch, and the code nodes
// /sdd-audit owes, with the dossier a judge needs.
//
// Usage: go run -C .agents/tools ./pending [--batch | --audit [--all] [node|dir...]] [--root <dir>]
//
//	--batch       print only the next batch (or what /sdd-design owes when nothing
//	              can be implemented) and the drift
//	--audit       print the dossier of every code node that owes an audit — or
//	              of the code nodes at or under each target — grouped by
//	              directory, and the drift
//	--all         with --audit, also the nodes already approved or parked
//	--root <dir>  repository root, absolute (default: nearest directory at or
//	              above the working directory with .sdd/.sdd-config.json)
//
// Output, plain text:
//
//	# Design      perspective specs no technical spec serves; technical specs
//	              stale against a source, with the rules that changed and the
//	              ## Spec gap that parks one; dangling sources of technical
//	              specs, with the rules recorded for them
//	# Implement   the backlog, sources first: technical specs with no code node,
//	              and directories with code nodes stale against a source, holding
//	              a dangling one, or missing a perspective node their technical
//	              specs serve — each with the union of its rules (new, changed,
//	              removed, renamed; park when a ## Design gap holds it) and
//	              whether it runs or waits
//	# Next batch  the units /sdd-implement continue runs now, at most 5, then the
//	              units that wait and why
//	# Audit       code nodes with no approval at their current content: owed
//	              to /sdd-audit, blocked by drift, after /sdd-implement, or parked
//	              under ## Audit gap. With --audit, the targeted nodes grouped by
//	              directory — one judge per directory — the standing of each
//	              node not judged, and one dossier per node judged: its file or
//	              the files it links with their specs, listed technical and
//	              perspective specs, candidate technical specs not listed
//	              (naming the node, global, the rest), and the perspective
//	              corrections that follow
//	# Drift       metadata that does not match its content, missing or orphan
//	              metadata, a metadata file that is no node, invalid sources, a
//	              bad link of a directory node, a code node with no technical
//	              spec, files under the specs root or the code root in no node,
//	              duplicate titles, cycles, gaps in TODO.md that match no rule
//	              or node
//
// With nothing to design or implement it prints "Nothing to design or
// implement." before the audit and the drift, and "No pending work." when
// those are empty too. Exit 0
// with the answer; non-zero with the reason on stderr.
//
// It implements **Continue** of .agents/skills/sdd-implement/SKILL.md and
// **Inventory** of .agents/skills/sdd-audit/SKILL.md over the states of
// **Metadata** in the root AGENTS.md; they win over it.
package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
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
		return g.CodeNodesUnder(g.Paths.Src), nil
	}
	seen := map[*graph.Node]bool{}
	var out []*graph.Node
	for _, d := range dirs {
		if filepath.IsAbs(d) {
			d = repo.Rel(g.Root, d)
		}
		rel := repo.Clean(d)
		var nodes []*graph.Node
		switch base := path.Base(rel); {
		case base == meta.FileName:
			if n := g.Node(path.Dir(rel)); n != nil && n.IsDir {
				nodes = []*graph.Node{n}
			}
		case strings.HasSuffix(base, meta.Suffix):
			if n := g.Node(strings.TrimSuffix(rel, meta.Suffix)); n != nil && n.Kind == graph.Code {
				nodes = []*graph.Node{n}
			}
		default:
			nodes = g.CodeNodesUnder(rel)
		}
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
	var audit []string
	if !batchOnly {
		audit = auditLines(g.Audits())
	}
	if len(design) == 0 && len(units) == 0 {
		if len(audit) == 0 && len(drift) == 0 {
			out = append(out, "No pending work.")
		} else {
			out = append(out, "Nothing to design or implement.")
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
			out = append(out, list(run)...)
		} else {
			out = append(out, "# Next batch: none", "", "Every unit waits:")
		}
		if len(wait) > 0 {
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
	if len(audit) > 0 {
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
			line += ": " + quoted(t.Order)
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

func quoted(titles []string) string {
	q := make([]string, len(titles))
	for i, t := range titles {
		q[i] = fmt.Sprintf("%q", t)
	}
	return strings.Join(q, ", ")
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
		if u.Wait == "" {
			for _, nu := range u.Nodes {
				if nu.Wait != "" {
					out = append(out, fmt.Sprintf("   wait %s — %s", nu.Node.Label(), nu.Wait))
				}
			}
		}
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
