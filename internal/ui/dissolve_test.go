package ui

import "testing"

func TestDissolveText_FrameZeroIsUnchanged(t *testing.T) {
	got := dissolveText("secret.csv", 0, 6)
	if got != "secret.csv" {
		t.Fatalf("got %q, want unchanged string at frame 0", got)
	}
}

func TestDissolveText_FinalFrameFullyGlitched(t *testing.T) {
	original := "secret.csv"
	got := dissolveText(original, 6, 6)
	if got == original {
		t.Fatal("expected the final frame to differ from the original text")
	}
	if len([]rune(got)) != len([]rune(original)) {
		t.Fatalf("got length %d, want %d", len([]rune(got)), len([]rune(original)))
	}
}

func TestDissolveText_ProgressesPartway(t *testing.T) {
	original := "secret.csv"
	full := dissolveText(original, 6, 6)
	half := dissolveText(original, 3, 6)
	if half == original || half == full {
		t.Fatal("expected a partial frame to differ from both the original and the final frame")
	}
}
