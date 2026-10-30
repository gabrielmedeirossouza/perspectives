package graph

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tools/internal/meta"
)

// Refresh returns n's metadata with the current hash and rules; modified moves
// only when the hash differs from the recorded one. Sources stay as recorded.
// A spec-less perspective or technical node is refused: its metadata is to be
// deleted or its spec.md written, not refreshed.
func Refresh(n *Node, now time.Time) (*meta.Metadata, error) {
	if msg := specless(n); msg != "" {
		return nil, errors.New(msg)
	}
	if n.MetaErr != nil {
		return nil, fmt.Errorf("`%s/%s`: %v", n.Rel, meta.FileName, n.MetaErr)
	}
	if n.RuleErr != nil {
		return nil, fmt.Errorf("`%s/%s`: %v", n.Rel, meta.SpecFile, n.RuleErr)
	}
	m := &meta.Metadata{Version: meta.Version, Algorithm: meta.Algorithm, Hash: n.Hash, Rules: n.Rules}
	if n.Meta != nil {
		m.Modified = n.Meta.Modified
		m.Sources = make([]meta.Source, len(n.Meta.Sources))
		copy(m.Sources, n.Meta.Sources)
	}
	if n.Meta == nil || n.Meta.Hash != n.Hash || m.Modified == "" {
		m.Modified = meta.Stamp(now)
	}
	return m, nil
}

// Sync is Refresh plus the current hash and rules of every source. It refuses
// a dangling or invalid source, a source whose own metadata has drift, a code
// node whose sources do not match the perspective nodes its technical specs
// serve, and a code node that waits on /design.
func (g *Graph) Sync(n *Node, now time.Time) (*meta.Metadata, error) {
	m, err := Refresh(n, now)
	if err != nil {
		return nil, err
	}
	for i, e := range n.Sources {
		switch {
		case e.Dangling():
			return nil, fmt.Errorf("`%s` lists `%s`, which is no node; drop it from sources", n.Rel, e.Source.Path)
		case !e.Valid():
			return nil, fmt.Errorf("`%s` (%s) lists `%s` (%s); %s", n.Rel, n.Kind, e.To.Rel, e.To.Kind, allowed(n.Kind))
		case g.HasDrift(e.To):
			return nil, fmt.Errorf("`%s` has drift; run metadata --write %s first", e.To.Rel, e.To.Rel)
		}
		m.Sources[i].Hash = e.To.Hash
		m.Sources[i].Rules = copyRules(e.To.Rules)
	}
	if n.Kind == Code {
		var problems []string
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
		if w := g.codeWait(n); w != "" {
			problems = append(problems, fmt.Sprintf("`%s` waits: %s", n.Rel, w))
		}
		if len(problems) > 0 {
			return nil, errors.New(strings.Join(problems, "\n"))
		}
	}
	return m, nil
}

// Save writes m to n's .metadata.json when its encoding differs from the file,
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
	file := filepath.Join(n.Dir, meta.FileName)
	if old, err := os.ReadFile(file); err == nil && bytes.Equal(old, b) {
		return false, nil
	}
	return true, os.WriteFile(file, b, 0o644)
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
