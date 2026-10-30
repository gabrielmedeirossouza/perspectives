package graph

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"tools/internal/md"
)

// Gap is one item parked in perspectives/TODO.md under ## Spec gap (a product
// rule /design cannot design against) or ## Design gap (a rule /implement
// cannot implement). Shape: `- **<title>** (`<owner>`) <reason>`, with an
// optional leading `[ ]` tolerated.
type Gap struct {
	Heading string // "spec" or "design"
	Title   string
	Node    string // root-relative node directory the owner names
	Reason  string
	matched bool
}

var gapRe = regexp.MustCompile("^\\s*-\\s*(?:\\[[ xX]\\]\\s*)?\\*\\*(.+?)\\*\\*\\s*\\(`([^`]+)`\\)\\s*(.*)$")

// loadGaps reads perspectives/TODO.md; items out of shape become drift.
func loadGaps(root string) ([]Gap, []string) {
	text, err := os.ReadFile(filepath.Join(root, "perspectives", "TODO.md"))
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
				drift = append(drift, "perspectives/TODO.md: "+heading+" gap out of shape: "+strings.TrimSpace(line))
				continue
			}
			owner := cleanRel(m[2])
			if !strings.HasPrefix(owner, "src/") && !strings.HasPrefix(owner, "perspectives/") {
				owner = "perspectives/" + owner
			}
			reason := strings.TrimRight(strings.TrimSpace(m[3]), ".")
			if reason == "" {
				reason = heading + " gap"
			}
			gaps = append(gaps, Gap{Heading: heading, Title: m[1], Node: owner, Reason: reason})
		}
	}
	return gaps, drift
}

// gapsFor returns the gaps of one heading parked on one node.
func (g *Graph) gapsFor(heading, node string) []*Gap {
	var out []*Gap
	for i := range g.gaps {
		if g.gaps[i].Heading == heading && g.gaps[i].Node == node {
			out = append(out, &g.gaps[i])
		}
	}
	return out
}

// park marks the rules of a unit that a ## Design gap item parks; a unit whose
// every rule is parked waits on /design.
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
		u.Wait = "every rule is parked under ## Design gap (/design)"
	}
	return u
}

// unmatchedGaps lists the gaps that park no rule: a design gap matching no
// pending rule, a spec gap matching no rule of the node it names.
func (g *Graph) unmatchedGaps() []string {
	g.Units() // marks design gaps
	var d []string
	for _, gap := range g.gaps {
		switch gap.Heading {
		case "design":
			if !gap.matched {
				d = append(d, fmt.Sprintf("perspectives/TODO.md: design gap %q (`%s`) matches no pending rule", gap.Title, gap.Node))
			}
		case "spec":
			n := g.byRel[gap.Node]
			if n == nil || n.Rules[gap.Title] == "" {
				d = append(d, fmt.Sprintf("perspectives/TODO.md: spec gap %q (`%s`) matches no rule", gap.Title, gap.Node))
			}
		}
	}
	return d
}
