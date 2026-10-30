package graph

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"tools/internal/md"
	"tools/internal/meta"
)

// AuditState is where a code node stands against the audit of its sources:
// the approvals it holds at its current hash, the drift that blocks a judge
// from reading it, whether /implement owes it first, and the ## Audit gap
// that parks it.
type AuditState struct {
	Node    *Node
	Audited int
	Blocked []string // blocking drift, each line naming its fix
	Unit    string   // why the node is an /implement unit; empty when it is none
	Parked  string   // reason of the ## Audit gap item naming it; empty when none
}

// Owed reports whether a judge is due now: no approval at the current hash,
// nothing blocking, no /implement unit, not parked.
func (s AuditState) Owed() bool {
	return s.Audited == 0 && len(s.Blocked) == 0 && s.Unit == "" && s.Parked == ""
}

// auditedOf is the approvals a node holds at its current hash: the recorded
// count while the hash is the recorded one, 0 otherwise.
func auditedOf(n *Node) int {
	if n.Meta == nil || n.Meta.Audited == nil || n.Meta.Hash != n.Hash {
		return 0
	}
	return *n.Meta.Audited
}

// AuditOf returns the audit state of code node n.
func (g *Graph) AuditOf(n *Node) AuditState {
	s := AuditState{Node: n, Audited: auditedOf(n)}
	for _, f := range g.findings(n) {
		if f.blocking {
			s.Blocked = append(s.Blocked, f.msg)
		}
	}
	if u, ok := g.codeUnit(n); ok {
		s.Unit = u.Reason
	}
	if gaps := g.gapsFor("audit", n.Rel); len(gaps) > 0 {
		s.Parked = gaps[0].Reason
	}
	return s
}

// Audits lists the audit state of every code node, by path.
func (g *Graph) Audits() []AuditState {
	var out []AuditState
	for _, n := range g.Nodes {
		if n.Kind == Code {
			out = append(out, g.AuditOf(n))
		}
	}
	return out
}

// CodeNodesUnder lists the code nodes at or under a root-relative directory, by path.
func (g *Graph) CodeNodesUnder(rel string) []*Node {
	var out []*Node
	for _, n := range g.Nodes {
		if n.Kind == Code && (n.Rel == rel || strings.HasPrefix(n.Rel, rel+"/")) {
			out = append(out, n)
		}
	}
	return out
}

// Candidate is a technical node a code node does not list, with what makes
// it worth the judge's read: its spec names the node's directory, or it is a
// global spec, which every code node it constrains lists.
type Candidate struct {
	Node    *Node
	Names   bool   // its spec.md names the node's directory or a parent of it below the code root
	Global  bool   // it serves no perspective node
	Purpose string // the purpose sentence of its spec.md
}

// Tier orders candidates for the judge: 1 names the directory, 2 global, 3 the rest.
func (c Candidate) Tier() int {
	switch {
	case c.Names:
		return 1
	case c.Global:
		return 2
	}
	return 3
}

// Dossier is what a judge needs about one code node. A served perspective the
// node misses is an /implement unit, not judged, so only drops are listed.
type Dossier struct {
	State            AuditState
	Files            []string
	Technical        []*Node     // listed technical specs with a valid edge, by path
	Perspectives     []*Node     // listed perspective specs with a valid edge, by path
	Candidates       []Candidate // technical specs not listed, tier then path
	DropPerspectives []string    // perspective nodes listed that no listed technical spec serves
}

// Dossier gathers what a judge needs about code node n.
func (g *Graph) Dossier(n *Node) Dossier {
	d := Dossier{State: g.AuditOf(n), Files: n.Files()}
	listed := map[*Node]bool{}
	for _, e := range sortedEdges(n.Sources) {
		if !e.Valid() {
			continue
		}
		listed[e.To] = true
		switch e.To.Kind {
		case Technical:
			d.Technical = append(d.Technical, e.To)
		case Perspective:
			d.Perspectives = append(d.Perspectives, e.To)
		}
	}
	names := namesRe(n.Rel, g.Paths.Src)
	for _, t := range g.Nodes {
		if t.Kind != Technical || listed[t] || !t.HasSpec {
			continue
		}
		text, _ := os.ReadFile(filepath.Join(t.Dir, meta.SpecFile))
		d.Candidates = append(d.Candidates, Candidate{Node: t, Names: names.Match(text), Global: g.global(t), Purpose: purpose(string(text))})
	}
	sort.SliceStable(d.Candidates, func(i, j int) bool {
		a, b := d.Candidates[i], d.Candidates[j]
		if a.Tier() != b.Tier() {
			return a.Tier() < b.Tier()
		}
		return a.Node.Rel < b.Node.Rel
	})
	served, _ := g.served(n)
	for _, p := range d.Perspectives {
		if _, ok := served[p]; !ok {
			d.DropPerspectives = append(d.DropPerspectives, fmt.Sprintf("`%s` (none of its technical specs serves it)", p.Rel))
		}
	}
	return d
}

// global reports whether technical node t serves no perspective node: a
// constraint that would hold with no feature, which every code node it
// constrains lists.
func (g *Graph) global(t *Node) bool {
	for _, e := range t.Sources {
		if e.Valid() && e.To.Kind == Perspective {
			return false
		}
	}
	return true
}

// namesRe matches a mention of the node's directory, or of a parent of it
// below the code root src, as a whole path token — optionally led by ./ and
// closed by a slash or a sentence period; a sibling (src/backend/invitee), a
// file beside it (src/backend/invite.ts), or a child path does not count. For
// the code root itself the token is the root, which most placement rules
// mention.
func namesRe(rel, src string) *regexp.Regexp {
	var dirs []string
	for dir := rel; dir != "." && dir != ""; dir = path.Dir(dir) {
		dirs = append(dirs, regexp.QuoteMeta(dir))
		if dir == src {
			break
		}
	}
	if rel != src {
		dirs = dirs[:len(dirs)-1] // the code root alone names every code node
	}
	return regexp.MustCompile(`(^|[^\w/.-])(?:\./)?(` + strings.Join(dirs, "|") + `)/?(?:[^\w/.-]|\.(?:[^\w]|$)|$)`)
}

// purpose is the first paragraph after the # title of a spec file — its
// purpose sentence, wrapped lines joined — or empty when a heading or a code
// fence follows the title.
func purpose(text string) string {
	for _, s := range md.Sections(text) {
		if s.Level != 1 {
			return ""
		}
		var lines []string
		for _, line := range s.Lines {
			t := strings.TrimSpace(line)
			switch {
			case t == "" && len(lines) > 0:
				return strings.Join(lines, " ")
			case t == "":
			case len(lines) == 0 && (strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~")):
				return ""
			default:
				lines = append(lines, t)
			}
		}
		return strings.Join(lines, " ")
	}
	return ""
}
