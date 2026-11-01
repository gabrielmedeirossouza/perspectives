package graph

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"tools/internal/md"
	"tools/internal/meta"
	"tools/internal/repo"
)

// ReviewDiff describes the difference between a spec's reviewed version in Git
// and its current working-tree content.
type ReviewDiff struct {
	Spec          *Node
	ReviewedHash  string
	CurrentHash   string
	FoundBase     bool
	SameHash      bool
	Delta         meta.Delta
	ChromeChanged bool
	AffectedRules []string
}

// ReviewCandidateState is the live validation state of one candidate in a
// review checkpoint.
type ReviewCandidateState struct {
	Candidate     meta.ReviewCandidate
	Node          *Node
	Removed       bool
	RemovedClean  bool
	VerdictValid  bool
	Applied       bool
	Blocked       bool
	InvalidReason string
}

// Pending reports whether this candidate still needs judgment, application,
// cleanup, or unblock before the review can close.
func (s ReviewCandidateState) Pending() bool {
	if s.Removed {
		return !s.RemovedClean
	}
	return !s.VerdictValid || s.Candidate.Verdict == "" || s.Blocked || !s.Applied
}

// Writer names the skill responsible for judging or resolving this candidate.
func (s ReviewCandidateState) Writer() string {
	if s.Candidate.Kind == "technical" || s.Blocked {
		return "/sdd-design"
	}
	return "/sdd-audit"
}

// ReviewSummary is the status of one spec's owed applicability review.
type ReviewSummary struct {
	Spec          *Node
	Owner         string // "/sdd-spec" for perspective, "/sdd-design" for technical
	CheckpointRel string
	HasCheckpoint bool
	SourceStale   bool // checkpoint Hash != Spec.Hash
	Scope         string
	AffectedRules []string
	Candidates    []ReviewCandidateState
	Total         int
	Resolved      int
	Blocked       int
	NextDesign    string
	NextAudit     string
}

// CanClose reports whether every candidate is resolved and the checkpoint matches
// the current spec hash.
func (s ReviewSummary) CanClose() bool {
	if !s.HasCheckpoint || s.SourceStale {
		return false
	}
	for _, c := range s.Candidates {
		if c.Pending() {
			return false
		}
	}
	return true
}

func (g *Graph) loadReviews(all []string) {
	revDir := filepath.Join(g.Root, filepath.FromSlash(meta.ReviewsDir))
	seen := map[string]bool{}
	var rels []string
	if entries, err := os.ReadDir(revDir); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			rel := meta.ReviewsDir + "/" + e.Name()
			seen[rel] = true
			rels = append(rels, rel)
		}
	}
	for _, f := range all {
		if strings.HasPrefix(f, meta.ReviewsDir+"/") && !seen[f] {
			if st, err := os.Stat(filepath.Join(g.Root, filepath.FromSlash(f))); err == nil && st.Mode().IsRegular() {
				seen[f] = true
				rels = append(rels, f)
			}
		}
	}
	sort.Strings(rels)
	owners := map[string]*Node{}
	for _, n := range g.Nodes {
		if n.spec() {
			owners[meta.ReviewRel(n.Rel)] = n
		}
	}
	for _, rel := range rels {
		if n := owners[rel]; n != nil {
			n.reviewInvalid = "unreadable"
			if raw, err := os.ReadFile(filepath.Join(g.Root, filepath.FromSlash(rel))); err == nil {
				n.reviewInvalid = meta.Sum(raw)
			}
		}
		if !strings.HasSuffix(rel, ".json") {
			g.reviewDrift = append(g.reviewDrift, loose{
				file: rel,
				msg:  fmt.Sprintf("`%s` is no review checkpoint; delete it", rel),
			})
			continue
		}
		r, err := meta.ReadReview(filepath.Join(g.Root, filepath.FromSlash(rel)))
		if err != nil {
			g.reviewDrift = append(g.reviewDrift, loose{
				file: rel,
				msg:  fmt.Sprintf("`%s`: %v", rel, err),
			})
			continue
		}
		wantRel := meta.ReviewRel(cleanRel(r.Source))
		if rel != wantRel {
			g.reviewDrift = append(g.reviewDrift, loose{
				file: rel,
				msg:  fmt.Sprintf("`%s` names source `%s`, whose checkpoint path is `%s`; delete or rename it", rel, r.Source, wantRel),
			})
			continue
		}
		src := g.byRel[cleanRel(r.Source)]
		if src == nil || !src.spec() || !src.HasFile {
			g.reviewDrift = append(g.reviewDrift, loose{
				file: rel,
				msg:  fmt.Sprintf("`%s` is an orphan review checkpoint for `%s`; delete it", rel, r.Source),
			})
			continue
		}
		src.review = r
		src.reviewInvalid = ""
	}
}

// OwesReview reports whether spec has an open review checkpoint or has changed
// since its last applicability review while having selective consumers.
func (n *Node) OwesReview() bool {
	if n == nil || !n.spec() || !n.HasFile {
		return false
	}
	if n.review != nil || n.reviewInvalid != "" {
		return true
	}
	if !n.selective || n.Meta == nil || n.MetaErr != nil {
		return false
	}
	if n.Meta.Reviewed != "" {
		return n.Meta.Reviewed != n.Hash
	}
	return n.syncedSelective || n.Meta.Hash != n.Hash
}

// ReviewDiff computes the diff between a spec's reviewed hash in Git and its
// current content.
func (g *Graph) ReviewDiff(spec *Node) ReviewDiff {
	d := ReviewDiff{Spec: spec, CurrentHash: spec.Hash}
	if spec.Meta != nil {
		d.ReviewedHash = spec.Meta.Reviewed
	}
	if d.ReviewedHash == "" {
		d.AffectedRules = append([]string(nil), spec.Order...)
		return d
	}
	if d.ReviewedHash == spec.Hash {
		d.FoundBase = true
		d.SameHash = true
		return d
	}
	blob, ok := g.findSpecBlob(spec.Rel, d.ReviewedHash)
	if !ok {
		d.AffectedRules = append([]string(nil), spec.Order...)
		return d
	}
	oldHashes, _, err := meta.RuleHashes(blob)
	if err != nil {
		d.AffectedRules = append([]string(nil), spec.Order...)
		return d
	}
	d.FoundBase = true
	var oldAdopted string
	if spec.Kind == Technical {
		oldAdopted = meta.AdoptedHash(blob)
	}
	d.Delta = meta.Diff(oldHashes, spec.Rules, spec.Order)
	if spec.Kind == Technical {
		d.Delta.Adopted = meta.DiffAdopted(oldAdopted, spec.Adopted)
	}
	d.ChromeChanged = md.Chrome(blob) != md.Chrome(spec.text)
	affected := map[string]bool{}
	for _, t := range d.Delta.Added {
		affected[t] = true
	}
	for _, t := range d.Delta.Changed {
		affected[t] = true
	}
	for _, r := range d.Delta.Renamed {
		affected[r[1]] = true
	}
	for _, t := range spec.Order {
		if affected[t] {
			d.AffectedRules = append(d.AffectedRules, t)
		}
	}
	for _, t := range d.Delta.Removed {
		if !affected[t] {
			d.AffectedRules = append(d.AffectedRules, t)
		}
	}
	if len(d.AffectedRules) == 0 && (d.ChromeChanged || d.Delta.Adopted != "") {
		d.AffectedRules = append([]string(nil), spec.Order...)
	}
	return d
}

func singleFileNodeHash(base string, b []byte) string {
	return meta.Sum([]byte(fmt.Sprintf("%s\x00%s\n", base, meta.Sum(b))))
}

func (g *Graph) findSpecBlob(rel, wantHash string) (string, bool) {
	base := path.Base(rel)
	if out, err := exec.Command("git", "-C", g.Root, "show", ":"+rel).Output(); err == nil {
		if singleFileNodeHash(base, out) == wantHash {
			return string(out), true
		}
	}
	revs, err := repo.Git(g.Root, "rev-list", "--all", "--", rel)
	if err != nil {
		return "", false
	}
	for _, commit := range strings.Split(strings.TrimSpace(revs), "\n") {
		commit = strings.TrimSpace(commit)
		if commit == "" {
			continue
		}
		out, err := exec.Command("git", "-C", g.Root, "show", commit+":"+rel).Output()
		if err == nil && singleFileNodeHash(base, out) == wantHash {
			return string(out), true
		}
	}
	return "", false
}

// Reviews returns the summary of every spec that currently owes an
// applicability review or has an open checkpoint.
func (g *Graph) Reviews() []ReviewSummary {
	var out []ReviewSummary
	for _, n := range g.Nodes {
		if n.OwesReview() {
			out = append(out, g.ReviewState(n))
		}
	}
	return out
}

// ReviewState evaluates a spec's applicability review and validates its
// checkpoint candidates against the current repository state.
func (g *Graph) ReviewState(spec *Node) ReviewSummary {
	owner := "/sdd-design"
	if spec.Kind == Perspective {
		owner = "/sdd-spec"
	}
	diff := g.ReviewDiff(spec)
	sum := ReviewSummary{
		Spec:          spec,
		Owner:         owner,
		CheckpointRel: meta.ReviewRel(spec.Rel),
		AffectedRules: diff.AffectedRules,
	}
	if spec.review == nil {
		return sum
	}
	r := spec.review
	sum.HasCheckpoint = true
	sum.SourceStale = r.Hash != spec.Hash
	sum.Scope = r.Scope
	if len(r.Rules) > 0 {
		var rules []string
		for _, t := range spec.Order {
			if _, ok := r.Rules[t]; ok {
				rules = append(rules, t)
			}
		}
		for _, t := range meta.SortedKeys(r.Rules) {
			if !slices.Contains(rules, t) {
				rules = append(rules, t)
			}
		}
		if len(rules) > 0 {
			sum.AffectedRules = rules
		}
	}
	existing := map[string]bool{}
	for _, c := range r.Candidates {
		existing[cleanRel(c.Path)] = true
		st := g.evalCandidate(spec, r, c)
		sum.Candidates = append(sum.Candidates, st)
	}
	// Any newly created allowed node on disk within the review's scope enters the queue.
	switch {
	case r.Scope == "unknown":
		for _, n := range g.Nodes {
			if n == spec || !n.Kind.MaySource(spec.Kind) || existing[n.Rel] {
				continue
			}
			kindStr := "code"
			if n.Kind == Technical {
				kindStr = "technical"
			}
			sum.Candidates = append(sum.Candidates, ReviewCandidateState{
				Candidate: meta.ReviewCandidate{Path: n.Rel, Kind: kindStr},
				Node:      n,
			})
		}
	case r.Scope == "expanded" && len(r.Paths) > 0:
		for _, p := range r.Paths {
			for _, n := range g.Resolve(cleanRel(p)) {
				if n == spec || !n.Kind.MaySource(spec.Kind) || existing[n.Rel] {
					continue
				}
				existing[n.Rel] = true
				kindStr := "code"
				if n.Kind == Technical {
					kindStr = "technical"
				}
				sum.Candidates = append(sum.Candidates, ReviewCandidateState{
					Candidate: meta.ReviewCandidate{Path: n.Rel, Kind: kindStr},
					Node:      n,
				})
			}
		}
	}
	for _, st := range sum.Candidates {
		sum.Total++
		if st.Blocked {
			sum.Blocked++
		}
		if !st.Pending() {
			sum.Resolved++
			continue
		}
		switch st.Writer() {
		case "/sdd-design":
			if sum.NextDesign == "" {
				sum.NextDesign = st.Candidate.Path
			}
		case "/sdd-audit":
			if sum.NextAudit == "" {
				sum.NextAudit = st.Candidate.Path
			}
		}
	}
	return sum
}

func (g *Graph) evalCandidate(spec *Node, r *meta.Review, c meta.ReviewCandidate) ReviewCandidateState {
	cp := cleanRel(c.Path)
	n := g.byRel[cp]
	st := ReviewCandidateState{Candidate: c, Node: n}
	if n == nil || (!n.IsDir && !n.HasFile) {
		st.Removed = true
		st.RemovedClean = g.removedCandidateClean(cp, n)
		if !st.RemovedClean {
			st.InvalidReason = "removed node still has orphan metadata or links"
		}
		return st
	}
	if c.Verdict == "" {
		return st
	}
	if n == spec || !n.Kind.MaySource(spec.Kind) || n.Kind.String() != c.Kind {
		st.InvalidReason = "invalid candidate kind or self-reference"
		return st
	}
	if b := g.blocking(n); len(b) > 0 {
		st.InvalidReason = "candidate drift: " + b[0]
		return st
	}
	if drift := g.blockedSource(n); drift != "" {
		st.InvalidReason = "support drift: " + drift
		return st
	}
	if c.Hash != n.Hash {
		st.InvalidReason = "candidate content changed"
		return st
	}
	if !maps.Equal(c.Support, g.candidateSupportMap(n, spec)) {
		st.InvalidReason = "judgment support sources or links changed"
		return st
	}
	if !g.sourceCompatible(spec, r, c, n.Kind) {
		st.InvalidReason = "source rules or context changed"
		return st
	}
	appliedOnDisk := g.verdictAppliedOnDisk(n, spec, c)
	curLinks := g.candidateLinksHash(n, spec)
	if c.Applied {
		if !appliedOnDisk || c.LinksHash != curLinks {
			st.InvalidReason = "candidate links changed after application"
			return st
		}
		st.VerdictValid = true
		st.Applied = true
		st.Blocked = c.Verdict == "block"
		return st
	}
	// Verdict was recorded before application finished (or is a block verdict).
	if c.Verdict == "block" {
		st.VerdictValid = true
		st.Blocked = true
		return st
	}
	if c.LinksHash != curLinks && (c.AppliedLinksHash != curLinks || !appliedOnDisk) {
		st.InvalidReason = "candidate links changed before application"
		return st
	}
	st.VerdictValid = true
	st.Applied = appliedOnDisk
	return st
}

func (g *Graph) removedCandidateClean(rel string, n *Node) bool {
	if n != nil && n.HasMeta {
		return false
	}
	for _, m := range g.Nodes {
		for _, e := range m.Sources {
			if cleanRel(e.Source.Path) == rel {
				return false
			}
		}
	}
	return true
}

func specChromeHash(spec *Node) string {
	return meta.Sum([]byte(md.Chrome(spec.text)))
}

func (g *Graph) candidateSupportMap(n, spec *Node) map[string]string {
	m := map[string]string{"chrome": specChromeHash(spec)}
	if spec.Kind == Perspective && n != nil && n.Kind == Code {
		m["persp"] = g.candidateSupportHash(n, spec)
	}
	if n != nil {
		for _, e := range n.Sources {
			if e.To != nil && e.To != spec {
				m["source:"+e.To.Rel] = e.To.Hash
				m["links:"+e.To.Rel] = g.candidateLinksHash(e.To, spec)
			}
		}
	}
	return m
}

func (g *Graph) sourceCompatible(spec *Node, r *meta.Review, c meta.ReviewCandidate, candKind Kind) bool {
	if c.SourceHash == spec.Hash && r.Hash == spec.Hash {
		return true
	}
	if r.Hash != spec.Hash {
		return false
	}
	if c.Support["chrome"] != specChromeHash(spec) {
		return false
	}
	if candKind == Technical && spec.Kind == Technical && c.Adopted != spec.Adopted {
		return false
	}
	if len(c.JudgedRules) == 0 {
		return false
	}
	for title, wantHash := range r.Rules {
		curHash := spec.Rules[title]
		if wantHash != curHash {
			return false
		}
		if judged, ok := c.JudgedRules[title]; !ok || judged != curHash {
			return false
		}
	}
	for _, u := range c.Uses {
		curHash, ok := spec.Rules[u]
		if !ok {
			return false
		}
		if judged, ok := c.JudgedRules[u]; !ok || judged != curHash {
			return false
		}
	}
	return true
}

func (g *Graph) candidateLinksHash(n, spec *Node) string {
	if n == nil || n.Meta == nil {
		return meta.Sum(nil)
	}
	return linksHash(n.Meta)
}

func linksHash(m *meta.Metadata) string {
	sources := make([]meta.Source, 0, len(m.Sources))
	for _, s := range m.Sources {
		sources = append(sources, meta.Source{Path: s.Path, Uses: s.Uses})
	}
	b, _ := json.Marshal(struct {
		Sources []meta.Source
		Files   map[string][]string
	}{sources, m.Files})
	return meta.Sum(b)
}

func (g *Graph) candidateSupportHash(c, persp *Node) string {
	if c == nil || c.Kind != Code || persp == nil || persp.Kind != Perspective {
		return ""
	}
	cov := g.coverage(c)
	if cov.served[persp] == nil {
		return "none"
	}
	if cov.integral[persp] {
		return "integral:" + strings.Join(persp.Order, "\x00")
	}
	return "selective:" + strings.Join(cov.reqTitles(persp), "\x00")
}

func (g *Graph) verdictAppliedOnDisk(n, spec *Node, c meta.ReviewCandidate) bool {
	if n == nil || n.Meta == nil {
		return false
	}
	var found *meta.Source
	for i := range n.Meta.Sources {
		if cleanRel(n.Meta.Sources[i].Path) == spec.Rel {
			found = &n.Meta.Sources[i]
			break
		}
	}
	switch c.Verdict {
	case "exclude":
		if found != nil {
			return false
		}
		if n.IsDir {
			for _, specs := range n.Meta.Files {
				if slices.Contains(specs, spec.Rel) {
					return false
				}
			}
		}
		return true
	case "keep":
		if found == nil {
			return false
		}
		if len(c.Uses) == 0 {
			return len(found.Uses) == 0
		}
		return slices.Equal(found.Uses, c.Uses)
	case "add":
		return found != nil && slices.Equal(found.Uses, c.Uses)
	case "reduce":
		if found == nil || !slices.Equal(found.Uses, c.Uses) {
			return false
		}
		if n.Kind == Code && auditedOf(n) > 0 {
			if found.Hash != "" {
				return false
			}
			for k := range found.Rules {
				if !slices.Contains(c.Uses, k) {
					return false
				}
			}
		}
		return true
	}
	return false
}

func (g *Graph) reviewNotesFor(n *Node) []ReviewNote {
	var out []ReviewNote
	for _, spec := range g.Nodes {
		if spec.review == nil {
			continue
		}
		for _, c := range spec.review.Candidates {
			if cleanRel(c.Path) != n.Rel {
				continue
			}
			st := g.evalCandidate(spec, spec.review, c)
			out = append(out, ReviewNote{
				Source:  spec.Rel,
				Rules:   meta.SortedKeys(spec.review.Rules),
				Verdict: c.Verdict,
				Uses:    append([]string(nil), c.Uses...),
				Reason:  c.Reason,
				Valid:   st.VerdictValid,
				Applied: st.Applied,
			})
		}
	}
	return out
}

// ClassifyReview classifies the applicability scope of a spec change since
// reviewed ("preserved", "expanded", or "unknown"). When scope is "preserved"
// and no checkpoint is open, it records reviewed on the spec immediately.
// Otherwise it opens or updates .sdd/reviews/<id>.json atomically, preserving
// valid candidate judgments when the judged rules and context are unchanged.
func (g *Graph) ClassifyReview(spec *Node, scope string, paths []string, now time.Time) (closed bool, err error) {
	if spec == nil || !spec.spec() || !spec.HasFile {
		return false, fmt.Errorf("review target must be a perspective or technical spec")
	}
	if b := g.blocking(spec); len(b) > 0 {
		return false, fmt.Errorf("`%s` has drift: %s", spec.Rel, b[0])
	}
	if spec.reviewInvalid != "" && scope != "unknown" {
		return false, fmt.Errorf("`%s` has an invalid checkpoint; reconstruct lost evidence with --scope unknown", spec.Rel)
	}
	switch scope {
	case "preserved", "expanded", "unknown":
	default:
		return false, fmt.Errorf("invalid review scope %q; use preserved, expanded, or unknown", scope)
	}
	diff := g.ReviewDiff(spec)
	if scope == "preserved" && !diff.FoundBase {
		return false, fmt.Errorf("`%s` has no previous reviewed version in git; classify with --scope expanded or --scope unknown", spec.Rel)
	}
	if scope == "preserved" && spec.review == nil {
		m, err := Refresh(spec, now)
		if err != nil {
			return false, err
		}
		m.Reviewed = spec.Hash
		if err := meta.CompleteReview(g.Root, spec.Rel, "", func() error {
			_, err := Save(spec, m)
			return err
		}); err != nil {
			return false, err
		}
		return true, nil
	}
	rules := map[string]string{}
	for _, t := range diff.AffectedRules {
		rules[t] = spec.Rules[t]
	}
	if len(rules) == 0 {
		for t, h := range spec.Rules {
			rules[t] = h
		}
	}
	if spec.review != nil && spec.review.Hash != spec.Hash {
		for t := range spec.review.Rules {
			if _, ok := rules[t]; !ok {
				rules[t] = spec.Rules[t]
			}
		}
	}
	candNodes := map[string]*Node{}
	addNode := func(n *Node) {
		if n != nil && n != spec && n.Kind.MaySource(spec.Kind) {
			candNodes[n.Rel] = n
		}
	}
	effScope := scope
	if spec.review != nil && scope == "preserved" {
		switch spec.review.Scope {
		case "unknown":
			effScope = "unknown"
		case "expanded":
			effScope = "expanded"
		}
	}
	var effPaths []string
	seenPaths := map[string]bool{}
	addScopePath := func(rel string) {
		rel = cleanRel(rel)
		if rel != "" && !seenPaths[rel] {
			seenPaths[rel] = true
			effPaths = append(effPaths, rel)
		}
	}
	if spec.review != nil && effScope == "expanded" {
		for _, p := range spec.review.Paths {
			addScopePath(p)
		}
	}
	for _, p := range paths {
		rel := repo.Target(g.Root, p)
		matches := g.Resolve(rel)
		if len(matches) == 0 {
			return false, fmt.Errorf("`%s` matches no node", rel)
		}
		addScopePath(rel)
		for _, n := range matches {
			addNode(n)
		}
	}
	switch {
	case effScope == "unknown" || (effScope == "expanded" && len(effPaths) == 0):
		effScope = "unknown"
		effPaths = nil
		for _, n := range g.Nodes {
			addNode(n)
		}
	case effScope == "expanded":
		sort.Strings(effPaths)
		for _, p := range effPaths {
			for _, n := range g.Resolve(p) {
				addNode(n)
			}
		}
	}
	prevByPath := map[string]meta.ReviewCandidate{}
	rawSum := spec.reviewInvalid
	if spec.review != nil {
		rawSum = spec.review.RawSum
		for _, c := range spec.review.Candidates {
			prevByPath[cleanRel(c.Path)] = c
		}
	}
	allPaths := map[string]bool{}
	for rel := range candNodes {
		allPaths[rel] = true
	}
	for rel := range prevByPath {
		allPaths[rel] = true
	}
	rev := &meta.Review{
		Version: meta.ReviewVersion,
		Source:  spec.Rel,
		Hash:    spec.Hash,
		Scope:   effScope,
		Rules:   rules,
		Paths:   effPaths,
		RawSum:  rawSum,
	}
	var sortedPaths []string
	for rel := range allPaths {
		sortedPaths = append(sortedPaths, rel)
	}
	sort.Strings(sortedPaths)
	for _, rel := range sortedPaths {
		n := g.byRel[rel]
		if prev, ok := prevByPath[rel]; ok {
			if n != nil && g.canReuseOnReclassify(spec, rev, prev, n) {
				prev.SourceHash = spec.Hash
				prev.Support = g.candidateSupportMap(n, spec)
				if n.Kind == Technical && spec.Kind == Technical {
					prev.Adopted = spec.Adopted
				}
				prev.JudgedRules = maps.Clone(rules)
				rev.Candidates = append(rev.Candidates, prev)
				continue
			}
			kindStr := prev.Kind
			if n != nil {
				if n.Kind == Technical {
					kindStr = "technical"
				} else {
					kindStr = "code"
				}
			}
			rev.Candidates = append(rev.Candidates, meta.ReviewCandidate{Path: rel, Kind: kindStr})
			continue
		}
		if n == nil {
			continue
		}
		kindStr := "code"
		if n.Kind == Technical {
			kindStr = "technical"
		}
		rev.Candidates = append(rev.Candidates, meta.ReviewCandidate{Path: rel, Kind: kindStr})
	}
	if err := meta.WriteReviewAtomic(g.Root, rev); err != nil {
		return false, err
	}
	loaded, err := meta.ReadReview(filepath.Join(g.Root, filepath.FromSlash(meta.ReviewRel(spec.Rel))))
	if err != nil {
		return false, err
	}
	spec.review = loaded
	spec.reviewInvalid = ""
	return false, nil
}

func (g *Graph) canReuseOnReclassify(spec *Node, newRev *meta.Review, prev meta.ReviewCandidate, n *Node) bool {
	if !g.evalCandidate(spec, newRev, prev).VerdictValid {
		return false
	}
	if prev.Verdict == "" || prev.Hash != n.Hash {
		return false
	}
	if spec.Kind == Perspective && n.Kind == Code && prev.Support["persp"] != g.candidateSupportHash(n, spec) {
		return false
	}
	if prev.Support["chrome"] != specChromeHash(spec) {
		return false
	}
	if n.Kind == Technical && spec.Kind == Technical && prev.Adopted != spec.Adopted {
		return false
	}
	if len(prev.JudgedRules) == 0 {
		return false
	}
	for title, newHash := range newRev.Rules {
		if oldHash, ok := prev.JudgedRules[title]; !ok || oldHash != newHash {
			return false
		}
	}
	for _, u := range prev.Uses {
		curHash, ok := spec.Rules[u]
		if !ok {
			return false
		}
		if oldHash, ok := prev.JudgedRules[u]; !ok || oldHash != curHash {
			return false
		}
	}
	if prev.Applied {
		return g.verdictAppliedOnDisk(n, spec, prev)
	}
	return true
}

// RecordVerdict saves a candidate judgment in spec's open review checkpoint
// before applying metadata edits.
func (g *Graph) RecordVerdict(spec, cand *Node, verdict string, uses []string, reason string) error {
	if spec == nil || !spec.spec() {
		return fmt.Errorf("review source must be a spec node")
	}
	if spec.review == nil {
		return fmt.Errorf("`%s` has no open review checkpoint; classify with --scope first", spec.Rel)
	}
	if spec.review.Hash != spec.Hash {
		return fmt.Errorf("`%s` changed since its review checkpoint was classified; re-classify with --scope first", spec.Rel)
	}
	if cand == nil || cand == spec || !cand.Kind.MaySource(spec.Kind) {
		return fmt.Errorf("invalid review candidate for `%s`", spec.Rel)
	}
	if b := g.blocking(cand); len(b) > 0 {
		return fmt.Errorf("`%s` has drift: %s", cand.Rel, b[0])
	}
	switch verdict {
	case "keep", "add", "reduce", "exclude", "block":
	default:
		return fmt.Errorf("invalid verdict %q; use keep, add, reduce, exclude, or block", verdict)
	}
	if (verdict == "exclude" || verdict == "block") && strings.TrimSpace(reason) == "" {
		return fmt.Errorf("verdict %q requires --reason", verdict)
	}
	if (verdict == "add" || verdict == "reduce") && len(uses) == 0 {
		return fmt.Errorf("verdict %q requires --uses", verdict)
	}
	seenUses := map[string]bool{}
	for _, u := range uses {
		if strings.TrimSpace(u) == "" {
			return fmt.Errorf("empty rule title in --uses")
		}
		if _, ok := spec.Rules[u]; !ok {
			return fmt.Errorf("`%s` has no rule %q", spec.Rel, u)
		}
		key := strings.ToLower(u)
		if seenUses[key] {
			return fmt.Errorf("duplicate rule %q in --uses", u)
		}
		seenUses[key] = true
	}
	var foundSrc *meta.Source
	if cand.Meta != nil {
		for i := range cand.Meta.Sources {
			if cleanRel(cand.Meta.Sources[i].Path) == spec.Rel {
				foundSrc = &cand.Meta.Sources[i]
				break
			}
		}
	}
	switch verdict {
	case "keep":
		if foundSrc == nil {
			return fmt.Errorf("`%s` does not list `%s`; use --verdict add or --verdict exclude", cand.Rel, spec.Rel)
		}
		if len(foundSrc.Uses) > 0 {
			if len(uses) == 0 {
				uses = append([]string(nil), foundSrc.Uses...)
			} else if !slices.Equal(uses, foundSrc.Uses) {
				return fmt.Errorf("`%s` changes uses for `%s`; use --verdict add or --verdict reduce", cand.Rel, spec.Rel)
			}
		} else if len(uses) > 0 {
			return fmt.Errorf("`%s` lists `%s` integrally; omit --uses for keep or use --verdict reduce", cand.Rel, spec.Rel)
		}
	case "reduce":
		if foundSrc == nil {
			return fmt.Errorf("`%s` does not list `%s`; use --verdict add", cand.Rel, spec.Rel)
		}
		if len(foundSrc.Uses) > 0 {
			for _, u := range uses {
				if !slices.Contains(foundSrc.Uses, u) {
					return fmt.Errorf("`%s` does not currently select %q from `%s`; use --verdict add to add rules", cand.Rel, u, spec.Rel)
				}
			}
		} else if len(foundSrc.Rules) > 0 {
			for _, u := range uses {
				if _, ok := foundSrc.Rules[u]; !ok {
					return fmt.Errorf("`%s` snapshot for `%s` does not cover rule %q; use --verdict add to add rules", cand.Rel, spec.Rel, u)
				}
			}
		}
	}
	if spec.Kind == Perspective && cand.Kind == Code && (verdict == "add" || verdict == "reduce" || (verdict == "keep" && len(uses) > 0)) {
		cov := g.coverage(cand)
		if cov.served[spec] == nil {
			return fmt.Errorf("`%s` has no technical spec serving `%s`; record --verdict block --reason ... for /sdd-design instead", cand.Rel, spec.Rel)
		}
		if !cov.integral[spec] {
			for _, u := range uses {
				if !cov.rules[spec][u] {
					return fmt.Errorf("`%s` technical specs do not serve rule %q of `%s`; record --verdict block --reason ... for /sdd-design instead", cand.Rel, u, spec.Rel)
				}
			}
		}
	}
	judged := maps.Clone(spec.review.Rules)
	if judged == nil {
		judged = map[string]string{}
	}
	for _, u := range uses {
		judged[u] = spec.Rules[u]
	}
	kindStr := "code"
	if cand.Kind == Technical {
		kindStr = "technical"
	}
	var adopted string
	if cand.Kind == Technical && spec.Kind == Technical {
		adopted = spec.Adopted
	}
	rc := meta.ReviewCandidate{
		Path:        cand.Rel,
		Kind:        kindStr,
		Hash:        cand.Hash,
		LinksHash:   g.candidateLinksHash(cand, spec),
		Support:     g.candidateSupportMap(cand, spec),
		SourceHash:  spec.Hash,
		Adopted:     adopted,
		JudgedRules: judged,
		Verdict:     verdict,
		Uses:        append([]string(nil), uses...),
		Reason:      strings.TrimSpace(reason),
	}
	rc.Applied = g.verdictAppliedOnDisk(cand, spec, rc)
	planned, err := Refresh(cand, time.Now())
	if err != nil {
		return err
	}
	applyVerdict(planned, cand.IsDir, spec.Rel, rc)
	rc.AppliedLinksHash = linksHash(planned)
	replaced := false
	for i := range spec.review.Candidates {
		if cleanRel(spec.review.Candidates[i].Path) == cand.Rel {
			spec.review.Candidates[i] = rc
			replaced = true
			break
		}
	}
	if !replaced {
		spec.review.Candidates = append(spec.review.Candidates, rc)
	}
	if err := meta.WriteReviewAtomic(g.Root, spec.review); err != nil {
		return err
	}
	loaded, err := meta.ReadReview(filepath.Join(g.Root, filepath.FromSlash(meta.ReviewRel(spec.Rel))))
	if err != nil {
		return err
	}
	spec.review = loaded
	return nil
}

// ApplyReviewCandidate applies the saved verdict of cand in spec's open review
// checkpoint idempotently to cand's metadata and records Applied in the
// checkpoint.
func (g *Graph) ApplyReviewCandidate(spec, cand *Node, now time.Time) error {
	if spec == nil || spec.review == nil {
		return fmt.Errorf("`%s` has no open review checkpoint", spec.Rel)
	}
	idx := -1
	for i := range spec.review.Candidates {
		if cleanRel(spec.review.Candidates[i].Path) == cand.Rel {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("`%s` is not a candidate in the review of `%s`", cand.Rel, spec.Rel)
	}
	c := spec.review.Candidates[idx]
	st := g.evalCandidate(spec, spec.review, c)
	if !st.VerdictValid {
		return fmt.Errorf("`%s` has no valid verdict in review of `%s` (%s)", cand.Rel, spec.Rel, st.InvalidReason)
	}
	if c.Verdict == "block" {
		return fmt.Errorf("`%s` is blocked (%s); resolve the block before applying", cand.Rel, c.Reason)
	}
	if !g.verdictAppliedOnDisk(cand, spec, c) {
		m, err := Refresh(cand, now)
		if err != nil {
			return err
		}
		applyVerdict(m, cand.IsDir, spec.Rel, c)
		if _, err := Save(cand, m); err != nil {
			return err
		}
		if err := g.rebuildNodeEdges(cand); err != nil {
			return err
		}
	}
	spec.review.Candidates[idx].LinksHash = g.candidateLinksHash(cand, spec)
	spec.review.Candidates[idx].Applied = true
	if err := meta.WriteReviewAtomic(g.Root, spec.review); err != nil {
		return err
	}
	loaded, err := meta.ReadReview(filepath.Join(g.Root, filepath.FromSlash(meta.ReviewRel(spec.Rel))))
	if err != nil {
		return err
	}
	spec.review = loaded
	return nil
}

func applyVerdict(m *meta.Metadata, isDir bool, source string, c meta.ReviewCandidate) {
	switch c.Verdict {
	case "exclude":
		var kept []meta.Source
		for _, s := range m.Sources {
			if cleanRel(s.Path) != source {
				kept = append(kept, s)
			}
		}
		m.Sources = kept
		if isDir && m.Files != nil {
			for f, specs := range m.Files {
				var ks []string
				for _, sp := range specs {
					if cleanRel(sp) != source {
						ks = append(ks, sp)
					}
				}
				m.Files[f] = ks
			}
		}
	case "keep", "add", "reduce":
		if len(c.Uses) > 0 {
			found := false
			for i := range m.Sources {
				if cleanRel(m.Sources[i].Path) == source {
					found = true
					m.Sources[i].Hash = ""
					m.Sources[i].Uses = append([]string(nil), c.Uses...)
					if c.Verdict == "reduce" && m.Sources[i].Rules != nil {
						pruned := make(map[string]string, len(c.Uses))
						for _, u := range c.Uses {
							if h, ok := m.Sources[i].Rules[u]; ok {
								pruned[u] = h
							}
						}
						m.Sources[i].Rules = pruned
					}
					break
				}
			}
			if !found {
				m.Sources = append(m.Sources, meta.Source{
					Path: source,
					Uses: append([]string(nil), c.Uses...),
				})
			}
		}
	}
}

func (g *Graph) rebuildNodeEdges(n *Node) error {
	if n.Meta == nil {
		return nil
	}
	n.Sources = nil
	for _, s := range n.Meta.Sources {
		e := &Edge{From: n, Source: s, To: g.byRel[cleanRel(s.Path)]}
		e.computeDelta()
		n.Sources = append(n.Sources, e)
	}
	g.rebuildIndex()
	return nil
}

// CloseReview verifies that every candidate in spec's open checkpoint is
// judged, valid, applied, and unblocked at spec's current hash, writes
// reviewed = spec.Hash to spec's metadata, and removes .sdd/reviews/<id>.json.
func (g *Graph) CloseReview(spec *Node, now time.Time) error {
	if spec == nil || !spec.spec() || !spec.HasFile {
		return fmt.Errorf("review target must be a perspective or technical spec")
	}
	if spec.reviewInvalid != "" {
		return fmt.Errorf("`%s` has an invalid checkpoint; reconstruct lost evidence with --scope unknown", spec.Rel)
	}
	if b := g.blocking(spec); len(b) > 0 {
		return fmt.Errorf("`%s` has drift: %s", spec.Rel, b[0])
	}
	if spec.review == nil {
		if spec.Meta != nil && spec.Meta.Reviewed == spec.Hash && !spec.OwesReview() {
			return nil
		}
		return fmt.Errorf("`%s` has no open review checkpoint; classify with --scope first", spec.Rel)
	}
	sum := g.ReviewState(spec)
	if sum.SourceStale {
		return fmt.Errorf("`%s` changed since its review checkpoint was classified; re-classify with --scope first", spec.Rel)
	}
	for _, st := range sum.Candidates {
		if st.Removed {
			if !st.RemovedClean {
				return fmt.Errorf("removed candidate `%s` still has orphan metadata or links", st.Candidate.Path)
			}
			continue
		}
		if st.Candidate.Verdict == "" {
			return fmt.Errorf("candidate `%s` has not been judged", st.Candidate.Path)
		}
		if !st.VerdictValid {
			return fmt.Errorf("candidate `%s` verdict is invalid (%s); judge it again", st.Candidate.Path, st.InvalidReason)
		}
		if st.Blocked {
			return fmt.Errorf("candidate `%s` is blocked: %s", st.Candidate.Path, st.Candidate.Reason)
		}
		if !st.Applied {
			return fmt.Errorf("candidate `%s` verdict %q is not applied; run --apply first", st.Candidate.Path, st.Candidate.Verdict)
		}
	}
	rawSum := spec.review.RawSum
	m, err := Refresh(spec, now)
	if err != nil {
		return err
	}
	m.Reviewed = spec.Hash
	if err := meta.CompleteReview(g.Root, spec.Rel, rawSum, func() error {
		_, err := Save(spec, m)
		return err
	}); err != nil {
		return err
	}
	spec.review = nil
	return nil
}
