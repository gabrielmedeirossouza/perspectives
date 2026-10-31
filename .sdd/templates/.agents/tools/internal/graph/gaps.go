package graph

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"tools/internal/config"
	"tools/internal/md"
	"tools/internal/meta"
)

// Gap is one item of a gap heading in TODO.md of the specs root (**TODO** in
// specs/AGENTS.md): `- **<title>** (`<owner>`) <reason>`, a leading `[ ]`
// tolerated.
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
			if heading != "audit" && !config.Under(owner, p.Src) && !config.Under(owner, p.Technical) && !config.Under(owner, p.Specs) {
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
		for _, n := range g.CodeNodes(gap.Owner) {
			if n.Rel == rel {
				return true
			}
		}
		return false
	}
	return gap.Owner == rel || gap.Owner+"/"+meta.SpecFile == rel
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
// every runnable rule is parked waits on /sdd-design.
func (g *Graph) park(u Unit) Unit {
	parkRule := func(r *RuleState) bool {
		for _, gap := range g.gapsFor("design", r.Source) {
			if strings.EqualFold(gap.Title, r.Title) {
				r.Parked = gap.Reason
				gap.matched = true
				return true
			}
		}
		return false
	}
	parked := 0
	for i := range u.Rules {
		if parkRule(&u.Rules[i]) {
			parked++
		}
	}
	canRun := 0
	for i := range u.Nodes {
		unparked := 0
		for j := range u.Nodes[i].Rules {
			if !parkRule(&u.Nodes[i].Rules[j]) {
				unparked++
			}
		}
		if u.Nodes[i].Wait == "" && (len(u.Nodes[i].Rules) == 0 || unparked > 0) {
			canRun++
		}
	}
	if u.Wait == "" {
		if len(u.Nodes) > 0 {
			if canRun == 0 {
				u.Wait = "every rule is parked under ## Design gap (/sdd-design)"
			}
		} else if len(u.Rules) > 0 && parked == len(u.Rules) {
			u.Wait = "every rule is parked under ## Design gap (/sdd-design)"
		}
	}
	return u
}

func hasRule(rules map[string]string, title string) bool {
	for k := range rules {
		if strings.EqualFold(k, title) {
			return true
		}
	}
	return false
}

func hasTitle(titles []string, title string) bool {
	for _, t := range titles {
		if strings.EqualFold(t, title) {
			return true
		}
	}
	return false
}

func (g *Graph) hasRemovedRule(n *Node, title string) bool {
	if n == nil || n.Kind != Perspective {
		return false
	}
	for _, t := range g.consumers(n, Technical) {
		for _, e := range t.Sources {
			if e.Valid() && e.To == n && hasTitle(e.Delta.Removed, title) {
				return true
			}
		}
	}
	return false
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
			if n == nil || n.Kind != Perspective || (!hasRule(n.Rules, gap.Title) && !g.hasRemovedRule(n, gap.Title)) {
				d = append(d, fmt.Sprintf(todo+": spec gap %q (`%s`) matches no rule", gap.Title, gap.Owner))
			}
		case "audit":
			owed := false
			for _, n := range g.CodeNodes(gap.Owner) {
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
