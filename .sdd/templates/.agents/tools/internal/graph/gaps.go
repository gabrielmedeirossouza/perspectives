package graph

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"tools/internal/config"
	"tools/internal/md"
	"tools/internal/meta"
)

// Gap is one item parked in TODO.md of the specs root under ## Spec gap (a product
// rule /sdd-design cannot design against), ## Design gap (a rule /sdd-implement
// cannot implement), or ## Audit gap (code no technical spec governs).
// Shape: `- **<title>** (`<owner>`) <reason>`, with an optional leading `[ ]`
// tolerated. A spec or design gap's owner is a spec directory or its spec.md;
// an audit gap's title names what the missing spec would decide and its owner
// is a code node (a file, or <dir>/.metadata.json for a directory node), a
// linked file — which parks the directory node that links it — or a
// directory, which parks every code node at or under it.
type Gap struct {
	Heading string // "spec", "design", or "audit"
	Title   string
	Owner   string // root-relative path the owner names
	Reason  string
	matched bool
}

var gapRe = regexp.MustCompile("^\\s*-\\s*(?:\\[[ xX]\\]\\s*)?\\*\\*(.+?)\\*\\*\\s*\\(`([^`]+)`\\)\\s*(.*)$")

// loadGaps reads TODO.md of the specs root; items out of shape become drift.
// An owner under no root names a perspective node.
func loadGaps(root string, p config.Paths) ([]Gap, []string) {
	todo := p.Specs + "/TODO.md"
	text, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(todo)))
	if err != nil {
		return nil, nil
	}
	var gaps []Gap
	var drift []string
	for _, s := range md.Sections(string(text)) {
		var heading string
		switch strings.ToLower(s.Title) {
		case "spec gap":
			heading = "spec"
		case "design gap":
			heading = "design"
		case "audit gap":
			heading = "audit"
		}
		if s.Level != 2 || heading == "" {
			continue
		}
		for _, line := range s.Lines {
			if !strings.HasPrefix(strings.TrimSpace(line), "-") {
				continue
			}
			m := gapRe.FindStringSubmatch(line)
			if m == nil {
				drift = append(drift, todo+": "+heading+" gap out of shape: "+strings.TrimSpace(line))
				continue
			}
			owner := cleanRel(m[2])
			if !config.Under(owner, p.Src) && !config.Under(owner, p.Technical) && !config.Under(owner, p.Specs) {
				owner = p.Specs + "/" + owner
			}
			reason := strings.TrimRight(strings.TrimSpace(m[3]), ".")
			if reason == "" {
				reason = heading + " gap"
			}
			gaps = append(gaps, Gap{Heading: heading, Title: m[1], Owner: owner, Reason: reason})
		}
	}
	return gaps, drift
}

// covers reports whether the gap's owner names node rel: the spec node
// itself or its directory for a spec or design gap; for an audit gap, one of
// the code nodes the owner parks.
func (g *Graph) covers(gap *Gap, rel string) bool {
	if gap.Heading == "audit" {
		for _, n := range g.auditNodes(gap.Owner) {
			if n.Rel == rel {
				return true
			}
		}
		return false
	}
	return gap.Owner == rel || gap.Owner+"/"+meta.SpecFile == rel
}

// auditNodes lists the code nodes an audit gap's owner parks: the node a
// file path or a metadata file names, the directory node that links a file,
// or every code node at or under a directory.
func (g *Graph) auditNodes(owner string) []*Node {
	dir, base := path.Dir(owner), path.Base(owner)
	switch {
	case base == meta.FileName:
		if n := g.byRel[dir]; n != nil && n.IsDir {
			return []*Node{n}
		}
		return nil
	case strings.HasSuffix(base, meta.Suffix):
		if n := g.byRel[strings.TrimSuffix(owner, meta.Suffix)]; n != nil && n.Kind == Code {
			return []*Node{n}
		}
		return nil
	}
	if n := g.byRel[owner]; n != nil && !n.IsDir {
		if n.Kind == Code {
			return []*Node{n}
		}
		return nil
	}
	if d := g.byRel[dir]; d != nil && d.IsDir && d.links(base) {
		return []*Node{d}
	}
	return g.CodeNodesUnder(owner)
}

// gapsFor returns the gaps of one heading parked on one node.
func (g *Graph) gapsFor(heading, rel string) []*Gap {
	var out []*Gap
	for i := range g.gaps {
		if g.gaps[i].Heading == heading && g.covers(&g.gaps[i], rel) {
			out = append(out, &g.gaps[i])
		}
	}
	return out
}

// park marks the rules of a unit that a ## Design gap item parks; a unit whose
// every rule is parked waits on /sdd-design.
func (g *Graph) park(u Unit) Unit {
	parked := 0
	for i, r := range u.Rules {
		for _, gap := range g.gapsFor("design", r.Source) {
			if strings.EqualFold(gap.Title, r.Title) {
				u.Rules[i].Parked = gap.Reason
				gap.matched = true
				parked++
				break
			}
		}
	}
	if u.Wait == "" && len(u.Rules) > 0 && parked == len(u.Rules) {
		u.Wait = "every rule is parked under ## Design gap (/sdd-design)"
	}
	return u
}

// specNode returns the spec node an owner names: the node itself or the
// spec.md of the directory.
func (g *Graph) specNode(owner string) *Node {
	if n := g.byRel[owner]; n != nil && n.spec() {
		return n
	}
	return g.byRel[owner+"/"+meta.SpecFile]
}

// unmatchedGaps lists the gaps that park nothing: a design gap matching no
// pending rule, a spec gap matching no rule of the node it names, an audit
// gap naming no code node that owes an audit.
func (g *Graph) unmatchedGaps() []string {
	g.Units() // marks design gaps
	todo := g.Paths.Specs + "/TODO.md"
	var d []string
	for _, gap := range g.gaps {
		switch gap.Heading {
		case "design":
			if !gap.matched {
				d = append(d, fmt.Sprintf(todo+": design gap %q (`%s`) matches no pending rule", gap.Title, gap.Owner))
			}
		case "spec":
			n := g.specNode(gap.Owner)
			if n == nil || n.Rules[gap.Title] == "" {
				d = append(d, fmt.Sprintf(todo+": spec gap %q (`%s`) matches no rule", gap.Title, gap.Owner))
			}
		case "audit":
			owed := false
			for _, n := range g.auditNodes(gap.Owner) {
				if auditedOf(n) == 0 {
					owed = true
					break
				}
			}
			if !owed {
				d = append(d, fmt.Sprintf(todo+": audit gap %q (`%s`) names no code node that owes an audit", gap.Title, gap.Owner))
			}
		}
	}
	return d
}
