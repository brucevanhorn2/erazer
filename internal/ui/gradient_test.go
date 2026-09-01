package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestGradientBox_PadsToRequestedHeight(t *testing.T) {
	out := gradientBox("hi", 10, 5, lipgloss.Color("#000000"), lipgloss.Color("#ffffff"))
	lines := strings.Split(out, "\n")
	if len(lines) != 7 { // height + top/bottom border rows
		t.Fatalf("got %d lines, want 7", len(lines))
	}
}

func TestGradientText_ReturnsNonEmptyStringForSingleLine(t *testing.T) {
	out := gradientText("hello", lipgloss.Color("#000000"), lipgloss.Color("#ffffff"))
	if out == "" {
		t.Fatal("expected non-empty rendered text")
	}
}
