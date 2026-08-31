package ui

import (
	"fmt"
	"strings"
)

func (m Model) View() string {
	if m.quitting {
		return ""
	}
	switch m.screen {
	case screenAbout:
		return m.about.View(m.theme)
	case screenConfirm:
		return m.confirmView()
	case screenDone:
		return m.doneView()
	default:
		return m.browsingView()
	}
}

func (m Model) browsingView() string {
	hint := m.theme.StatusBar.Render(
		"↑/↓ move  enter open  space select  e erase selected  ? about  q quit",
	)
	return m.browser.View(m.theme) + "\n" + hint
}

func (m Model) confirmView() string {
	var b strings.Builder
	b.WriteString(m.theme.Header.Render("Confirm erasure"))
	b.WriteString("\n\n")
	b.WriteString("Targets:\n")
	for _, t := range m.targets {
		b.WriteString("  " + t + "\n")
	}
	b.WriteString("\n")
	if m.rotationalOK && !m.rotational {
		b.WriteString(m.theme.StatusError.Render(
			"warning: at least one target is on non-rotational (SSD/NVMe) storage;\n"+
				"overwrite shredding is not a guarantee against forensic recovery on flash media.") + "\n\n")
	}
	b.WriteString(m.theme.StatusKey.Render("Passes: ") + m.passesInput.View() + "\n")
	b.WriteString(m.theme.StatusKey.Render("Seed:   ") + m.seedInput.View() + "\n\n")

	trigger := "[ ERAZE ]"
	if m.confirmFocus == 2 {
		trigger = m.theme.EraseTriggerFocused.Render(trigger)
	} else {
		trigger = m.theme.EraseTrigger.Render(trigger)
	}
	b.WriteString(trigger + "\n")

	if m.confirmErr != "" {
		b.WriteString("\n" + m.theme.StatusError.Render(m.confirmErr) + "\n")
	}
	b.WriteString("\n" + m.theme.StatusBar.Render("tab/shift+tab move  enter confirm  esc back"))
	return b.String()
}

func (m Model) doneView() string {
	var b strings.Builder
	b.WriteString(m.theme.StatusKey.Render("Done.") + "\n\n")
	if m.doneErr != "" {
		b.WriteString(m.theme.StatusError.Render(m.doneErr) + "\n")
	} else {
		b.WriteString(fmt.Sprintf("%d file(s) shredded, %d bytes overwritten\n", m.result.FilesShredded, m.result.BytesOverwritten))
		for _, e := range m.result.Errors {
			b.WriteString(m.theme.StatusError.Render(fmt.Sprintf("error: %s: %v", e.Path, e.Err)) + "\n")
		}
	}
	b.WriteString("\n" + m.theme.StatusBar.Render("press any key to quit"))
	return b.String()
}
