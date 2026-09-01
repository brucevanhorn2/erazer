package browse

import "testing"

func TestEffective_DefaultUnselected(t *testing.T) {
	s := NewSet()
	if s.Effective("/home/bruce/Documents") {
		t.Fatal("expected unselected path to be false by default")
	}
}

func TestToggle_SelectsRecursively(t *testing.T) {
	s := NewSet()
	s.Toggle("/home/bruce/Projects")

	if !s.Effective("/home/bruce/Projects") {
		t.Fatal("expected explicitly toggled path to be selected")
	}
	if !s.Effective("/home/bruce/Projects/erazer") {
		t.Fatal("expected child of selected folder to inherit selection")
	}
	if s.Effective("/home/bruce/Documents") {
		t.Fatal("expected unrelated path to remain unselected")
	}
}

func TestToggle_ChildOverridesParent(t *testing.T) {
	s := NewSet()
	s.Toggle("/home/bruce/Projects")
	s.Toggle("/home/bruce/Projects/erazer") // carve out an exception

	if !s.Effective("/home/bruce/Projects") {
		t.Fatal("expected parent to remain selected")
	}
	if s.Effective("/home/bruce/Projects/erazer") {
		t.Fatal("expected explicitly deselected child to be excluded")
	}
	if !s.Effective("/home/bruce/Projects/exfil") {
		t.Fatal("expected sibling to still inherit parent's selection")
	}
}

func TestToggle_Twice_ReselectsChild(t *testing.T) {
	s := NewSet()
	s.Toggle("/home/bruce/Projects")
	s.Toggle("/home/bruce/Projects/erazer")
	s.Toggle("/home/bruce/Projects/erazer") // toggle back on

	if !s.Effective("/home/bruce/Projects/erazer") {
		t.Fatal("expected re-toggled child to be selected again")
	}
}

func TestSelectedRoots_FindsTopmostSelectedPaths(t *testing.T) {
	s := NewSet()
	s.Toggle("/home/bruce/Projects")
	s.Toggle("/home/bruce/Projects/erazer") // excluded exception
	s.Toggle("/home/bruce/Documents/taxes") // a standalone leaf selection

	roots := s.SelectedRoots()
	want := []string{"/home/bruce/Documents/taxes", "/home/bruce/Projects"}
	if len(roots) != len(want) {
		t.Fatalf("got roots %v, want %v", roots, want)
	}
	for i := range want {
		if roots[i] != want[i] {
			t.Fatalf("got roots %v, want %v", roots, want)
		}
	}
}

func TestIsExplicit(t *testing.T) {
	s := NewSet()
	s.Toggle("/home/bruce/Projects")

	if !s.IsExplicit("/home/bruce/Projects") {
		t.Fatal("expected toggled path to be explicit")
	}
	if s.IsExplicit("/home/bruce/Projects/erazer") {
		t.Fatal("expected inherited path to not be explicit")
	}
}
