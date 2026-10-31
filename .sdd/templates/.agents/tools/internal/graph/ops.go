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
	m := &meta.Metadata{Version: meta.Version, Algorithm: meta.Algorithm, Hash: n.Hash, Rules: n.Rules}
	if n.Meta != nil {
		m.Modified = n.Meta.Modified
		m.Sources = make([]meta.Source, len(n.Meta.Sources))
		copy(m.Sources, n.Meta.Sources)
		for i := range m.Sources {
			m.Sources[i].Path = cleanRel(m.Sources[i].Path)
		}
		m.Files = cloneFiles(n.Meta.Files)
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
		if e.Source.Hash == "" && m.Audited != nil {
			*m.Audited = 0
		}
		m.Sources[i].Hash = e.To.Hash
		m.Sources[i].Rules = maps.Clone(e.To.Rules)
	}
	if n.Kind == Code {
		var problems []string
		if msg := g.noTechnical(n); msg != "" {
			problems = append(problems, msg) // which perspectives are served is unknown without a technical spec
		} else {
			served, _ := g.served(n)
			for _, p := range g.missing(n) {
				problems = append(problems, fmt.Sprintf("`%s` lists `%s` but not `%s`, which it serves; add it to sources", n.Rel, served[p].Rel, p.Rel))
			}
			for _, p := range g.unserved(n) {
				problems = append(problems, unservedMsg(n, p))
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
			if e.Valid() {
				e.Delta = meta.Diff(e.Source.Rules, e.To.Rules, e.To.Order)
			}
		}
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
