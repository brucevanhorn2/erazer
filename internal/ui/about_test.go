package ui

import (
	"strings"
	"testing"
)

func TestAboutPane_ViewContainsTagline(t *testing.T) {
	a := NewAboutPane()
	a.Width, a.Height = 60, 20
	theme := NewTheme()
	out := a.View(theme)
	if !strings.Contains(out, "bad week") {
		t.Fatal("expected the tagline text to appear in the rendered output")
	}
}
