package graph

import (
	"bytes"
	"errors"
	"fmt"
	"maps"
	"os"
	"strings"
	"time"

	"tools/internal/meta"
)

// Refresh computes tool-owned fields without changing sources or file links.
// It refuses blocking drift that only a hand repair fixes.
func Refresh(n *Node, now time.Time) (*meta.Metadata, error) {
	for _, f := range selfFindings(n) {
		if f.blocking && !f.written {
			return nil, errors.New(f.msg)
		}
	}
	m := &meta.Metadata{
		Version:   meta.Version,
		Algorithm: meta.Algorithm,
		Hash:      n.Hash,
		Adopted:   n.Adopted,
		Rules:     n.Rules,
	}
	if n.Meta != nil {
		m.Modified = n.Meta.Modified
		m.Sources = make([]meta.Source, len(n.Meta.Sources))
		for i, s := range n.Meta.Sources {
			s.Path = cleanRel(s.Path)
			s.Rules = maps.Clone(s.Rules)
			if s.Uses != nil {
				s.Uses = append([]string(nil), s.Uses...)
			}
			m.Sources[i] = s
		}
		m.Files = cloneFiles(n.Meta.Files)
		if n.spec() {
			switch {
			case n.selective || n.review != nil || n.reviewInvalid != "":
				if n.Meta.Reviewed != "" {
					m.Reviewed = n.Meta.Reviewed
				} else if n.review == nil && n.reviewInvalid == "" && !n.syncedSelective && n.Meta.Hash == n.Hash {
					m.Reviewed = n.Hash
				}
			case n.Meta.Reviewed != "":
				m.Reviewed = n.Hash
			}
		}
	}
	if n.Meta == nil || n.Meta.Hash != n.Hash || m.Modified == "" {
		m.Modified = meta.Stamp(now)
	}
	if n.Kind == Code {
		audited := auditedOf(n)
		m.Audited = &audited
	}
	return m, nil
}

// ensureReviewed initializes reviewed to the current spec hash when an intact
// spec without reviewed and without an open review checkpoint gains its first
// selective consumer via --sync or --approve.
func (g *Graph) ensureReviewed(spec *Node) error {
	if !spec.spec() || spec.Meta == nil || spec.MetaErr != nil {
		return nil
	}
	if spec.Meta.Reviewed != "" || spec.review != nil || spec.reviewInvalid != "" || spec.syncedSelective || spec.Meta.Hash != spec.Hash {
		return nil
	}
	cp := *spec.Meta
	cp.Reviewed = spec.Hash
	_, err := Save(spec, &cp)
	return err
}

// Sync refreshes and records source versions after dependency validation.
// Newly recorded edges invalidate the audit of the previous source list.
func (g *Graph) Sync(n *Node, now time.Time) (*meta.Metadata, error) {
	m, err := Refresh(n, now)
	if err != nil {
		return nil, err
	}
	if g.inCycle[n] {
		return nil, errors.New(g.cycleMsg())
	}
	if n.IsDir && len(n.content) == 0 {
		return nil, fmt.Errorf("`%s` links no file; a directory node links at least one", n.MetaRel)
	}
	if len(n.refErrs) > 0 {
		return nil, errors.New(strings.Join(n.refErrs, "\n"))
	}
	for i, e := range n.Sources {
		switch {
		case e.Dangling():
			return nil, fmt.Errorf("`%s` lists `%s`, which is no node%s; drop it from sources", n.Rel, e.Source.Path, g.hint(e.Source.Path))
		case !e.Valid():
			return nil, fmt.Errorf("`%s` (%s) lists `%s` (%s); %s", n.Rel, n.Kind, e.To.Rel, e.To.Kind, allowed(n.Kind))
		}
		if line := g.driftLine(e.To); line != "" {
			return nil, fmt.Errorf("%s; fix it first", line)
		}
		if e.Selective() {
			ci := map[string]string{}
			for _, t := range e.To.Order {
				ci[strings.ToLower(t)] = t
			}
			renamedTo := map[string]string{}
			for _, r := range e.Delta.Renamed {
				renamedTo[r[0]] = r[1]
			}
			for _, u := range e.Source.Uses {
				if _, ok := e.To.Rules[u]; ok {
					continue
				}
				if newTitle, ok := renamedTo[u]; ok {
					return nil, fmt.Errorf("`%s` selects %q from `%s`, which was renamed to %q; update uses first", n.Rel, u, e.To.Rel, newTitle)
				}
				if _, inSnap := e.Source.Rules[u]; inSnap {
					return nil, fmt.Errorf("`%s` selects %q from `%s`, which was removed; update uses first", n.Rel, u, e.To.Rel)
				}
				if actual, ok := ci[strings.ToLower(u)]; ok {
					return nil, fmt.Errorf("`%s` lists %q in uses for `%s`, which differs in case from %q; use %q", n.MetaRel, u, e.To.Rel, actual, actual)
				}
				return nil, fmt.Errorf("`%s` lists unknown rule %q in uses for `%s`", n.MetaRel, u, e.To.Rel)
			}
			if m.Audited != nil {
				goneHashes := map[string]int{}
				for oldTitle, oldHash := range e.Source.Rules {
					if _, ok := e.To.Rules[oldTitle]; !ok {
						goneHashes[oldHash]++
					}
				}
				for _, u := range e.Source.Uses {
					if _, hadSnap := e.Source.Rules[u]; hadSnap {
						continue
					}
					curHash := e.To.Rules[u]
					if goneHashes[curHash] > 0 {
						goneHashes[curHash]--
					} else {
						*m.Audited = 0
					}
				}
			}
			m.Sources[i].Hash = ""
			pruned := make(map[string]string, len(e.Source.Uses))
			for _, u := range e.Source.Uses {
				pruned[u] = e.To.Rules[u]
			}
			m.Sources[i].Rules = pruned
		} else {
			if e.Source.Hash == "" && m.Audited != nil {
				*m.Audited = 0
			}
			m.Sources[i].Hash = e.To.Hash
			m.Sources[i].Rules = maps.Clone(e.To.Rules)
		}
		if n.Kind == Technical && e.To.Kind == Technical {
			m.Sources[i].Adopted = e.To.Adopted
		} else {
			m.Sources[i].Adopted = ""
		}
	}
	if n.Kind == Code {
		var problems []string
		if msg := g.noTechnical(n); msg != "" {
			problems = append(problems, msg) // which perspectives are served is unknown without a technical spec
		} else {
			for _, mc := range g.missingCov(n) {
				if mc.unlisted {
					problems = append(problems, fmt.Sprintf("`%s` lists `%s` but not `%s`, which it serves; add it to sources", n.Rel, mc.servedBy.Rel, mc.node.Rel))
				} else {
					problems = append(problems, fmt.Sprintf("`%s` lists `%s` (which serves %s of `%s`), but omits %s in uses; add %s to uses", n.Rel, mc.servedBy.Rel, meta.Quoted(mc.titles), mc.node.Rel, meta.Quoted(mc.titles), meta.Quoted(mc.titles)))
				}
			}
			for _, uc := range g.unservedCov(n) {
				problems = append(problems, uc.msg)
			}
		}
		if w := g.codeWait(n); w != "" {
			problems = append(problems, fmt.Sprintf("`%s` waits: %s", n.Rel, w))
		}
		if u, ok := g.codeUnit(n); ok {
			for _, r := range u.Rules {
				for _, gap := range g.gapsFor("design", r.Source) {
					if strings.EqualFold(gap.Title, r.Title) {
						problems = append(problems, fmt.Sprintf("`%s` waits: %q (`%s`) is parked under ## Design gap: %s (/sdd-design)", n.Rel, r.Title, r.Source, gap.Reason))
						break
					}
				}
			}
		}
		if len(problems) > 0 {
			return nil, errors.New(strings.Join(problems, "\n"))
		}
	}
	for _, e := range n.Sources {
		if e.Valid() && e.Selective() {
			if err := g.ensureReviewed(e.To); err != nil {
				return nil, err
			}
		}
	}
	return m, nil
}

// Approve records a judgment only for unchanged, valid code with no pending
// implementation. Waiting on design alone does not prevent approval.
func (g *Graph) Approve(n *Node, now time.Time) (*meta.Metadata, error) {
	if n.Kind != Code {
		return nil, fmt.Errorf("`%s` is a %s node; only a code node is approved", n.Rel, n.Kind)
	}
	if n.Meta != nil && n.MetaErr == nil && n.Meta.Hash != "" && n.Meta.Hash != n.Hash {
		return nil, fmt.Errorf("`%s` changed since its metadata was last written; run metadata --write %s and judge it again", n.Rel, n.Rel)
	}
	if d := g.NodeDrift(n); len(d) > 0 {
		return nil, errors.New(strings.Join(d, "\n"))
	}
	if n.IsDir && n.Meta != nil {
		for _, k := range meta.SortedKeys(n.Meta.Files) {
			if len(n.Meta.Files[k]) == 0 {
				return nil, fmt.Errorf("`%s` links `%s` to no spec; every linked file needs a governing technical spec before approval", n.MetaRel, k)
			}
		}
	}
	if u, ok := g.codeUnit(n); ok {
		return nil, fmt.Errorf("`%s` is an /sdd-implement unit (%s); approve it once synced", n.Rel, u.Reason)
	}
	m, err := Refresh(n, now)
	if err != nil {
		return nil, err
	}
	for i, e := range n.Sources {
		if e.Valid() && e.Selective() {
			m.Sources[i].Hash = ""
			pruned := make(map[string]string, len(e.Source.Uses))
			for _, u := range e.Source.Uses {
				if h, ok := e.Source.Rules[u]; ok {
					pruned[u] = h
				}
			}
			m.Sources[i].Rules = pruned
			if err := g.ensureReviewed(e.To); err != nil {
				return nil, err
			}
		}
	}
	*m.Audited++
	return m, nil
}

// Save writes m to n's metadata file when its encoding differs from the file,
// and makes n and its edges carry m so later operations in the same run see it.
func Save(n *Node, m *meta.Metadata) (bool, error) {
	b, err := meta.Encode(m)
	if err != nil {
		return false, err
	}
	n.Meta, n.MetaErr, n.HasMeta = m, nil, true
	for i, e := range n.Sources {
		if i < len(m.Sources) {
			e.Source = m.Sources[i]
			e.computeDelta()
		}
	}
	if n.g != nil {
		n.g.rebuildIndex()
	}
	if old, err := os.ReadFile(n.MetaPath); err == nil && bytes.Equal(old, b) {
		return false, nil
	}
	return true, os.WriteFile(n.MetaPath, b, 0o644)
}

// cloneFiles copies a files map deeply, nil staying nil.
func cloneFiles(f map[string][]string) map[string][]string {
	if f == nil {
		return nil
	}
	out := make(map[string][]string, len(f))
	for k, v := range f {
		cp := make([]string, len(v))
		for i, p := range v {
			cp[i] = cleanRel(p)
		}
		out[k] = cp
	}
	return out
}
