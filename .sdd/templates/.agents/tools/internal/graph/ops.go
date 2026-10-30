package graph

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"tools/internal/meta"
)

// Refresh returns n's metadata with the current hash and rules; modified moves
// only when the hash differs from the recorded one. Sources and files stay as
// recorded. On a code node, audited keeps its count while the hash is the
// recorded one and returns to 0 when the hash moved; other kinds carry no
// audited. An orphan metadata, a spec node that is not spec.md, a technical
// spec without its skeleton, and a directory node with a bad link are
// refused: they are fixed by hand, not refreshed.
func Refresh(n *Node, now time.Time) (*meta.Metadata, error) {
	if msg := malformed(n); msg != "" {
		return nil, errors.New(msg)
	}
	if !n.HasMeta && n.Kind == Technical {
		return nil, fmt.Errorf("`%s` has no %s; write its skeleton `{\"version\": %d, \"sources\": [...]}` (/sdd-design)", n.Rel, path.Base(n.MetaRel), meta.Version)
	}
	if n.MetaErr != nil {
		return nil, fmt.Errorf("`%s`: %v", n.MetaRel, n.MetaErr)
	}
	if n.RuleErr != nil {
		return nil, fmt.Errorf("`%s`: %v", n.Rel, n.RuleErr)
	}
	for _, f := range n.linkErrs {
		if f.blocking {
			return nil, errors.New(f.msg)
		}
	}
	if n.Meta != nil && !n.IsDir && len(n.Meta.Files) > 0 {
		return nil, fmt.Errorf("`%s` carries files, which only a directory node has; drop it", n.MetaRel)
	}
	m := &meta.Metadata{Version: meta.Version, Algorithm: meta.Algorithm, Hash: n.Hash, Rules: n.Rules}
	if n.Meta != nil {
		m.Modified = n.Meta.Modified
		m.Sources = make([]meta.Source, len(n.Meta.Sources))
		copy(m.Sources, n.Meta.Sources)
		m.Files = copyFiles(n.Meta.Files)
	}
	if n.Meta == nil || n.Meta.Hash != n.Hash || m.Modified == "" {
		m.Modified = meta.Stamp(now)
	}
	if n.Kind == Code {
		audited := 0
		if n.Meta != nil && n.Meta.Hash == n.Hash && n.Meta.Audited != nil {
			audited = *n.Meta.Audited
		}
		m.Audited = &audited
	}
	return m, nil
}

// Sync is Refresh plus the current hash and rules of every source. It refuses
// a spec with a link of its own that names nothing (its writer syncs it only
// once the link is fixed), a dangling or invalid source, a source whose
// content or metadata is not what it claims — a broken link of a source is
// its writer's drift and refuses nothing here — a code
// node that lists no technical spec, a code node whose sources do not match
// the perspective nodes its technical specs serve, and a code node that waits
// on /sdd-design. Recording a source listed since the last sync returns audited
// to 0: no judge has seen that edge.
func (g *Graph) Sync(n *Node, now time.Time) (*meta.Metadata, error) {
	m, err := Refresh(n, now)
	if err != nil {
		return nil, err
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
		case len(g.blocking(e.To)) > 0:
			return nil, fmt.Errorf("`%s` has drift; fix it first: %s", e.To.Rel, g.blocking(e.To)[0])
		}
		if e.Source.Hash == "" && m.Audited != nil {
			*m.Audited = 0
		}
		m.Sources[i].Hash = e.To.Hash
		m.Sources[i].Rules = copyRules(e.To.Rules)
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
			for _, e := range n.Sources {
				if e.Valid() && e.To.Kind == Perspective {
					if _, ok := served[e.To]; !ok {
						problems = append(problems, fmt.Sprintf("`%s` lists `%s`, but none of its technical specs serves it", n.Rel, e.To.Rel))
					}
				}
			}
		}
		if w := g.codeWait(n); w != "" {
			problems = append(problems, fmt.Sprintf("`%s` waits: %s", n.Rel, w))
		}
		if len(problems) > 0 {
			return nil, errors.New(strings.Join(problems, "\n"))
		}
	}
	return m, nil
}

// Approve is Refresh plus one more approval on audited: a judge confirmed that
// every technical spec the code node lists governs its code and that no
// technical spec it obeys is missing. It refuses a node that is no code node,
// one with drift (a hash that moved since the metadata was last written
// included), one with a dangling source, and one that is an /sdd-implement unit —
// stale against a source or missing a served perspective — so an approval is
// recorded only on a node that has no drift and is no unit; the count then
// persists until the hash moves or a sync records a new source. A node that
// only waits on /sdd-design is approved.
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

func copyRules(r map[string]string) map[string]string {
	if r == nil {
		return nil
	}
	out := make(map[string]string, len(r))
	for k, v := range r {
		out[k] = v
	}
	return out
}

func copyFiles(f map[string][]string) map[string][]string {
	if f == nil {
		return nil
	}
	out := make(map[string][]string, len(f))
	for k, v := range f {
		out[k] = append([]string{}, v...)
	}
	return out
}
