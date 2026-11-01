package graph

import (
	"fmt"
	"net/url"
	"path"
	"regexp"
	"slices"
	"sort"
	"strings"

	"tools/internal/md"
)

// AuditState is where a code node stands against the audit of its sources:
// the approvals it holds at its current hash, the drift that blocks a judge
// from reading it, whether /sdd-implement owes it first, and the ## Audit gap
// that parks it.
type AuditState struct {
	Node    *Node
	Audited int
	Blocked []string // blocking drift, each line naming its fix
	Unit    string   // why the node is an /sdd-implement unit; empty when it is none
	Parked  string   // reason of the ## Audit gap item naming it; empty when none
}

// Owed reports whether a judge is due now: no approval at the current hash,
// nothing blocking, no /sdd-implement unit, not parked.
func (s AuditState) Owed() bool {
	return s.Audited == 0 && len(s.Blocked) == 0 && s.Unit == "" && s.Parked == ""
}

// auditedOf is the approvals a node holds at its current hash: the recorded
// count while the hash is the recorded one and every file a directory node
// links has a spec, 0 otherwise.
func auditedOf(n *Node) int {
	if n.Meta == nil || n.Meta.Audited == nil || n.Meta.Hash != n.Hash {
		return 0
	}
	if n.IsDir {
		for _, specs := range n.Meta.Files {
			if len(specs) == 0 {
				return 0
			}
		}
	}
	return *n.Meta.Audited
}

// AuditOf returns the audit state of code node n.
func (g *Graph) AuditOf(n *Node) AuditState {
	s := AuditState{Node: n, Audited: auditedOf(n), Blocked: g.blocking(n)}
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

// Candidate is a technical node a code node does not list, with what makes
// it worth the judge's read: its spec names the node's file or directory, or
// it is a global spec, which every code node it constrains lists.
type Candidate struct {
	Node    *Node
	Names   bool   // its spec.md names the node's file, its directory, or a parent of it below the code root
	Global  bool   // it serves no perspective node
	Purpose string // the purpose sentence of its spec.md
}

// Tier orders candidates for the judge: 1 names the node, 2 global, 3 the rest.
func (c Candidate) Tier() int {
	switch {
	case c.Names:
		return 1
	case c.Global:
		return 2
	}
	return 3
}

// ListedSpec describes a technical or perspective spec a code node lists,
// including which rules are selected via uses and which remain unselected.
type ListedSpec struct {
	Node       *Node
	Selective  bool
	Selected   []string // in spec Order
	Unselected []string // in spec Order
}

// ReviewNote summarizes an open applicability review targeting a code node.
type ReviewNote struct {
	Source  string
	Rules   []string
	Verdict string
	Uses    []string
	Reason  string
	Valid   bool
	Applied bool
}

// Dossier is what a judge needs about one code node. A served perspective the
// node misses is an /sdd-implement unit, not judged, so only drops are listed.
type Dossier struct {
	State            AuditState
	Files            []string            // the content: the file, or the files a directory node links
	Links            map[string][]string // directory node: each linked file with the sources it is linked to
	Technical        []*Node             // listed technical specs with a valid edge, by path
	Perspectives     []*Node             // listed perspective specs with a valid edge, by path
	ListedTechnical  []ListedSpec        // listed technical specs with selected/unselected rules
	ListedPersp      []ListedSpec        // listed perspective specs with selected/unselected rules
	AnchorHints      []string            // #anchor links from selected rules to unselected rules
	Candidates       []Candidate         // technical specs not listed, tier then path
	DropPerspectives []string            // perspective nodes or rules listed that no listed technical spec serves
	Reviews          []ReviewNote        // active applicability reviews targeting this code node
}

// Dossier gathers what a judge needs about code node n.
func (g *Graph) Dossier(n *Node) Dossier {
	d := Dossier{State: g.AuditOf(n), Files: n.Files()}
	if n.IsDir && n.Meta != nil {
		d.Links = cloneFiles(n.Meta.Files)
	}
	listed := map[*Node]*Edge{}
	for _, e := range sortedEdges(n.Sources) {
		if !e.Valid() || listed[e.To] != nil {
			continue
		}
		listed[e.To] = e
		useSet := map[string]bool{}
		for _, u := range e.Source.Uses {
			useSet[u] = true
		}
		ls := ListedSpec{Node: e.To, Selective: e.Selective()}
		if e.Selective() {
			for _, t := range e.To.Order {
				if useSet[t] {
					ls.Selected = append(ls.Selected, t)
				} else {
					ls.Unselected = append(ls.Unselected, t)
				}
			}
		} else {
			ls.Selected = append([]string(nil), e.To.Order...)
		}
		switch e.To.Kind {
		case Technical:
			d.Technical = append(d.Technical, e.To)
			d.ListedTechnical = append(d.ListedTechnical, ls)
		case Perspective:
			d.Perspectives = append(d.Perspectives, e.To)
			d.ListedPersp = append(d.ListedPersp, ls)
		}
	}
	d.AnchorHints = g.anchorHints(n, listed)
	var linked []string
	if n.IsDir {
		linked = d.Files
	}
	names := namesRe(n.Rel, g.Paths.Src, linked...)
	for _, t := range g.Nodes {
		if t.Kind != Technical || listed[t] != nil || !t.HasFile {
			continue
		}
		d.Candidates = append(d.Candidates, Candidate{Node: t, Names: names.MatchString(t.text), Global: g.global(t), Purpose: t.Purpose})
	}
	sort.SliceStable(d.Candidates, func(i, j int) bool {
		a, b := d.Candidates[i], d.Candidates[j]
		if a.Tier() != b.Tier() {
			return a.Tier() < b.Tier()
		}
		return a.Node.Rel < b.Node.Rel
	})
	for _, u := range g.unservedCov(n) {
		d.DropPerspectives = append(d.DropPerspectives, u.drop)
	}
	sort.Strings(d.DropPerspectives)
	d.Reviews = g.reviewNotesFor(n)
	return d
}

func (g *Graph) anchorHints(n *Node, listed map[*Node]*Edge) []string {
	var hints []string
	seen := map[string]bool{}
	for _, e := range sortedEdges(n.Sources) {
		if !e.Valid() || !e.Selective() || e.To.ruleLinks == nil {
			continue
		}
		useSet := map[string]bool{}
		for _, u := range e.Source.Uses {
			useSet[u] = true
		}
		for _, u := range e.To.Order {
			if !useSet[u] {
				continue
			}
			for _, raw := range e.To.ruleLinks[u] {
				filePart, anchor, ok := strings.Cut(raw, "#")
				if !ok || anchor == "" {
					continue
				}
				filePart, _, _ = strings.Cut(filePart, "?")
				if t, err := url.PathUnescape(filePart); err == nil {
					filePart = t
				}
				if f, err := url.PathUnescape(anchor); err == nil {
					anchor = f
				}
				targetSpec := e.To
				if filePart != "" {
					var rel string
					if strings.HasPrefix(filePart, "/") {
						rel = cleanRel(strings.TrimLeft(filePart, "/"))
					} else {
						rel = cleanRel(path.Join(path.Dir(e.To.Rel), filePart))
					}
					targetSpec = g.byRel[rel]
					if targetSpec == nil {
						targetSpec = g.byRel[cleanRel(filePart)]
					}
					if targetSpec == nil || !targetSpec.spec() {
						continue
					}
				}
				targetTitle := targetSpec.slugToRule[md.Slug(anchor)]
				if targetTitle == "" {
					continue
				}
				if te := listed[targetSpec]; te != nil {
					if !te.Selective() || slices.Contains(te.Source.Uses, targetTitle) {
						continue
					}
				}
				var msg string
				if targetSpec == e.To {
					msg = fmt.Sprintf("`%s` %q links to unselected `#%s` (%q)", e.To.Rel, u, anchor, targetTitle)
				} else {
					msg = fmt.Sprintf("`%s` %q links to `%s#%s` (%q, unselected)", e.To.Rel, u, targetSpec.Rel, anchor, targetTitle)
				}
				if !seen[msg] {
					seen[msg] = true
					hints = append(hints, msg)
				}
			}
		}
	}
	return hints
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

// namesRe matches a mention of the node's path — the file of a file node,
// the directory and linked files of a directory node — or of a parent of it
// below the code root src, as a whole path token — optionally led by ./ or /
// and closed by a slash or a sentence period; a sibling (src/backend/invitee),
// a file with a longer name (src/backend/invite.ts beside src/backend/invite),
// or a child path does not count. For the code root itself the token is the
// root, which most placement rules mention.
func namesRe(rel, src string, files ...string) *regexp.Regexp {
	var dirs []string
	for _, f := range files {
		dirs = append(dirs, regexp.QuoteMeta(rel+"/"+f))
	}
	for dir := rel; dir != "." && dir != ""; dir = path.Dir(dir) {
		dirs = append(dirs, regexp.QuoteMeta(dir))
		if dir == src {
			break
		}
	}
	if rel != src {
		dirs = dirs[:len(dirs)-1] // the code root alone names every code node
	}
	return regexp.MustCompile(`(^|[^\w/.-])(?:\.?/)?(` + strings.Join(dirs, "|") + `)/?(?:[^\w/.-]|\.(?:[^\w]|$)|$)`)
}

// purpose is the first paragraph after the # title of a spec file — its
// purpose sentence, wrapped lines joined — or empty when a heading, an out-of-
// scope line, or a code fence follows the title.
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
			case strings.HasPrefix(t, "**Out of scope:**"):
				return strings.Join(lines, " ")
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
