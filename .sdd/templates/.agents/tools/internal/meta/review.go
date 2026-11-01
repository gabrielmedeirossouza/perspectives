package meta

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

const (
	ReviewVersion = 1
	ReviewsDir    = ".sdd/reviews"
)

// Review is an open applicability-review checkpoint for one source spec in
// .sdd/reviews/<id>.json.
type Review struct {
	Version    int               `json:"version"`
	Source     string            `json:"source"`
	Hash       string            `json:"hash"`
	Scope      string            `json:"scope"` // "preserved", "expanded", or "unknown"
	Rules      map[string]string `json:"rules,omitempty"`
	Paths      []string          `json:"paths,omitempty"`
	Candidates []ReviewCandidate `json:"candidates,omitempty"`
	RawSum     string            `json:"-"` // sha256 of the file when read, for concurrency checks
}

// ReviewCandidate is one potential consumer in an open review checkpoint.
type ReviewCandidate struct {
	Path             string            `json:"path"`
	Kind             string            `json:"kind"` // "technical" or "code"
	Hash             string            `json:"hash,omitempty"`
	SourceHash       string            `json:"source_hash,omitempty"`
	JudgedRules      map[string]string `json:"judged_rules,omitempty"`
	Adopted          string            `json:"adopted,omitempty"`
	Uses             []string          `json:"uses,omitempty"`
	LinksHash        string            `json:"links_hash,omitempty"`
	AppliedLinksHash string            `json:"applied_links_hash,omitempty"`
	Support          map[string]string `json:"support,omitempty"`
	Verdict          string            `json:"verdict,omitempty"` // "keep", "add", "reduce", "exclude", "block"
	Reason           string            `json:"reason,omitempty"`
	Applied          bool              `json:"applied,omitempty"`
}

// ReviewID is the deterministic checkpoint identifier for a spec node path.
func ReviewID(specRel string) string {
	trimmed := strings.TrimSuffix(specRel, "/"+SpecFile)
	slug := strings.NewReplacer("/", "-", ".", "-").Replace(trimmed)
	return slug + "-" + Sum([]byte(specRel))[:8]
}

// ReviewRel is the root-relative path of a spec node's review checkpoint.
func ReviewRel(specRel string) string {
	return ReviewsDir + "/" + ReviewID(specRel) + ".json"
}

// ReadReview parses and validates a checkpoint file at path.
func ReadReview(path string) (*Review, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var r Review
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&r); err != nil {
		return nil, fmt.Errorf("invalid JSON: %v", err)
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		return nil, errors.New("invalid JSON: trailing content")
	}
	if err := r.Validate(); err != nil {
		return nil, err
	}
	r.RawSum = Sum(raw)
	return &r, nil
}

// Validate checks the structural invariants of a review checkpoint.
func (r *Review) Validate() error {
	if r.Version != ReviewVersion {
		return fmt.Errorf("version %d is not supported; expected %d", r.Version, ReviewVersion)
	}
	if strings.TrimSpace(r.Source) == "" {
		return errors.New("missing source")
	}
	if strings.TrimSpace(r.Hash) == "" {
		return errors.New("missing hash")
	}
	switch r.Scope {
	case "preserved", "expanded", "unknown":
	default:
		return fmt.Errorf("invalid scope %q", r.Scope)
	}
	seen := map[string]bool{}
	for _, c := range r.Candidates {
		cp := path.Clean(strings.TrimSpace(c.Path))
		if cp == "" || cp == "." || cp == ".." || strings.HasPrefix(cp, "../") {
			return errors.New("candidate with empty or invalid path")
		}
		if seen[cp] {
			return fmt.Errorf("duplicate candidate %q", cp)
		}
		seen[cp] = true
		if c.Kind != "technical" && c.Kind != "code" {
			return fmt.Errorf("candidate %q: invalid kind %q", cp, c.Kind)
		}
		switch c.Verdict {
		case "", "keep":
		case "add", "reduce":
			if len(c.Uses) == 0 {
				return fmt.Errorf("candidate %q: %s verdict requires uses", cp, c.Verdict)
			}
		case "exclude", "block":
			if strings.TrimSpace(c.Reason) == "" {
				return fmt.Errorf("candidate %q: %s verdict requires a reason", cp, c.Verdict)
			}
		default:
			return fmt.Errorf("candidate %q: invalid verdict %q", cp, c.Verdict)
		}
		seenUses := map[string]bool{}
		for _, u := range c.Uses {
			if strings.TrimSpace(u) == "" {
				return fmt.Errorf("candidate %q: empty rule in uses", cp)
			}
			key := strings.ToLower(u)
			if seenUses[key] {
				return fmt.Errorf("candidate %q: duplicate rule %q in uses", cp, u)
			}
			seenUses[key] = true
		}
	}
	return nil
}

// EncodeReview renders r byte-stably with sorted candidates.
func EncodeReview(r *Review) ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	c := *r
	if len(r.Candidates) > 0 {
		c.Candidates = make([]ReviewCandidate, len(r.Candidates))
		copy(c.Candidates, r.Candidates)
		sort.Slice(c.Candidates, func(i, j int) bool { return c.Candidates[i].Path < c.Candidates[j].Path })
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(&c); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// WriteReviewAtomic writes r atomically to .sdd/reviews/<id>.json after
// verifying that the file on disk has not changed since r.RawSum was read.
func WriteReviewAtomic(root string, r *Review) error {
	b, err := EncodeReview(r)
	if err != nil {
		return err
	}
	return WithReviewLock(root, r.Source, r.RawSum, func() error {
		if err := writeReview(root, r, b); err != nil {
			return err
		}
		r.RawSum = Sum(b)
		return nil
	})
}

// WithReviewLock serializes checkpoint compare-and-change across processes.
// expectedSum empty requires absence. The callback must not reacquire the lock.
func WithReviewLock(root, source, expectedSum string, change func() error) error {
	rel := ReviewRel(source)
	abs := filepath.Join(root, filepath.FromSlash(rel))
	dir := filepath.Dir(abs)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	// Keep the lock outside the removable reviews directory: a closer may remove
	// that directory while another process is about to acquire its own lock.
	lock := filepath.Join(root, ".sdd", ".review-"+ReviewID(source)+".lock")
	if err := os.Mkdir(lock, 0o700); err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("`%s` is being changed concurrently; reload and retry", rel)
		}
		return err
	}
	defer func() {
		_ = os.Remove(lock)
		_ = os.Remove(dir)
	}()
	if err := checkReview(abs, rel, expectedSum); err != nil {
		return err
	}
	return change()
}

func checkReview(abs, rel, expectedSum string) error {
	cur, readErr := os.ReadFile(abs)
	switch {
	case readErr == nil && expectedSum == "":
		return fmt.Errorf("`%s` was created concurrently; reload and retry", rel)
	case readErr == nil && Sum(cur) != expectedSum:
		return fmt.Errorf("`%s` changed concurrently; reload and retry", rel)
	case os.IsNotExist(readErr) && expectedSum != "":
		return fmt.Errorf("`%s` was removed concurrently; reload and retry", rel)
	case readErr != nil && !os.IsNotExist(readErr):
		return readErr
	}
	return nil
}

func writeReview(root string, r *Review, b []byte) error {
	rel := ReviewRel(r.Source)
	abs := filepath.Join(root, filepath.FromSlash(rel))
	dir := filepath.Dir(abs)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".review-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, abs); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}

// RemoveReview deletes .sdd/reviews/<id>.json after verifying expectedSum, and
// removes .sdd/reviews when empty.
func RemoveReview(root, specRel, expectedSum string) error {
	return CompleteReview(root, specRel, expectedSum, func() error { return nil })
}

// CompleteReview checks the checkpoint before saving reviewed and removing it,
// under the same lock used by checkpoint writers.
func CompleteReview(root, source, expectedSum string, save func() error) error {
	return WithReviewLock(root, source, expectedSum, func() error {
		if err := save(); err != nil {
			return err
		}
		if expectedSum == "" {
			return nil
		}
		return os.Remove(filepath.Join(root, filepath.FromSlash(ReviewRel(source))))
	})
}
