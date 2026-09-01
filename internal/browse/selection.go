package browse

import (
	"sort"
	"strings"
)

// Set tracks explicit include/exclude overrides on a path tree. A path's
// effective selection is inherited from the nearest ancestor override,
// defaulting to unselected when no ancestor has one — so selecting a
// folder cascades to everything under it, and a deeper override lets you
// carve out an exception. Unlike a flat per-directory selection map, this
// state is independent of which directory is currently being browsed, so
// it survives navigating away and back.
type Set struct {
	overrides map[string]bool
}

// NewSet returns an empty selection (nothing selected).
func NewSet() *Set {
	return &Set{overrides: make(map[string]bool)}
}

// Toggle flips the effective selection at path by recording an explicit
// override there (the opposite of whatever Effective(path) currently
// returns).
func (s *Set) Toggle(path string) {
	s.overrides[path] = !s.Effective(path)
}

// Effective reports whether path is selected: an explicit override at path
// wins; otherwise it's inherited from the nearest ancestor override; with
// no override anywhere in the chain it's false.
func (s *Set) Effective(path string) bool {
	p := path
	for {
		if v, ok := s.overrides[p]; ok {
			return v
		}
		parent := parentOf(p)
		if parent == p {
			return false
		}
		p = parent
	}
}

// IsExplicit reports whether path itself (not an ancestor) carries an
// override. Used by the UI to distinguish an explicitly-checked entry from
// one that's merely selected because a parent is.
func (s *Set) IsExplicit(path string) bool {
	_, ok := s.overrides[path]
	return ok
}

// SelectedRoots returns the minimal set of paths that are effective-true
// but whose parent is not — the top-level targets a shred run should start
// from. Callers must still consult Effective per-descendant when walking a
// root, since a root can contain deeper excluded exceptions.
func (s *Set) SelectedRoots() []string {
	var roots []string
	for p := range s.overrides {
		if s.Effective(p) && !s.Effective(parentOf(p)) {
			roots = append(roots, p)
		}
	}
	sort.Strings(roots)
	return roots
}

func parentOf(path string) string {
	trimmed := strings.TrimRight(path, "/")
	idx := strings.LastIndex(trimmed, "/")
	if idx <= 0 {
		return "/"
	}
	return trimmed[:idx]
}
