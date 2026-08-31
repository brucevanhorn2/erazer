package ui

import "time"

const (
	dissolveFrameCount = 6
	dissolveInterval   = 120 * time.Millisecond
)

// glitchChars reuses the block-shading characters from the About screen's
// logo, so the erasing animation feels visually consistent with the rest
// of erazer's chrome.
var glitchChars = []rune("░▒▓█#%&@$")

// dissolveText renders name with an increasing fraction of its characters
// replaced by glitch characters as frame advances toward frames,
// simulating a "derez" effect — a cheap, deterministic, campy stand-in for
// a real dissolve animation. At frame >= frames the whole string is
// glitched.
func dissolveText(name string, frame, frames int) string {
	runes := []rune(name)
	if frames <= 0 {
		frames = 1
	}
	cutoff := len(runes) * frame / frames
	if cutoff > len(runes) {
		cutoff = len(runes)
	}
	out := make([]rune, len(runes))
	for i, r := range runes {
		if i < cutoff {
			out[i] = glitchChars[(i+frame)%len(glitchChars)]
		} else {
			out[i] = r
		}
	}
	return string(out)
}
