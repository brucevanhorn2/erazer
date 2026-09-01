package ui

import "testing"

func TestNewTheme_SetsPrimaryColor(t *testing.T) {
	theme := NewTheme()
	if theme.PrimaryColor != "#B341F5" {
		t.Fatalf("got %v, want #B341F5", theme.PrimaryColor)
	}
	if theme.MutedPrimaryColor == theme.PrimaryColor {
		t.Fatal("expected muted color to differ from the full-strength color")
	}
}
