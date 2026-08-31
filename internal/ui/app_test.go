package ui

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/brucevanhorn2/erazer/internal/shred"
)

func TestModel_StartsOnBrowsingScreen(t *testing.T) {
	dir := t.TempDir()
	m := NewModel(dir)
	if m.screen != screenBrowsing {
		t.Fatalf("got screen %v, want screenBrowsing", m.screen)
	}
}

func TestHandleBrowsingKey_ENoSelectionStaysOnBrowsing(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	m := NewModel(dir)

	updated, _ := m.handleBrowsingKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	got := updated.(Model)
	if got.screen != screenBrowsing {
		t.Fatalf("got screen %v, want screenBrowsing", got.screen)
	}
}

func TestHandleBrowsingKey_ESelectionMovesToConfirm(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	m := NewModel(dir)
	m.browser.Cursor = 0
	m.browser.ToggleSelect()

	updated, _ := m.handleBrowsingKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	got := updated.(Model)
	if got.screen != screenConfirm {
		t.Fatalf("got screen %v, want screenConfirm", got.screen)
	}
	if len(got.targets) != 1 {
		t.Fatalf("got %d targets, want 1", len(got.targets))
	}
}

func TestHandleBrowsingKey_QuestionMarkOpensAbout(t *testing.T) {
	dir := t.TempDir()
	m := NewModel(dir)
	updated, _ := m.handleBrowsingKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	got := updated.(Model)
	if got.screen != screenAbout {
		t.Fatalf("got screen %v, want screenAbout", got.screen)
	}
}

func TestConfirmScreen_EscReturnsToBrowsing(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	m := NewModel(dir)
	m.browser.ToggleSelect()
	updated, _ := m.handleBrowsingKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	m = updated.(Model)

	updated, _ = m.handleConfirmKey(tea.KeyMsg{Type: tea.KeyEsc})
	got := updated.(Model)
	if got.screen != screenBrowsing {
		t.Fatalf("got screen %v, want screenBrowsing", got.screen)
	}
}

func TestConfirmScreen_TabCyclesFocus(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	m := NewModel(dir)
	m.browser.ToggleSelect()
	updated, _ := m.handleBrowsingKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	m = updated.(Model)

	if m.confirmFocus != 0 {
		t.Fatalf("got initial confirmFocus %d, want 0", m.confirmFocus)
	}
	updated, _ = m.handleConfirmKey(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)
	if m.confirmFocus != 1 {
		t.Fatalf("got confirmFocus %d after one tab, want 1", m.confirmFocus)
	}
	updated, _ = m.handleConfirmKey(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)
	if m.confirmFocus != 2 {
		t.Fatalf("got confirmFocus %d after two tabs, want 2", m.confirmFocus)
	}
}

func TestConfirmScreen_InvalidPassesShowsErrorAndDoesNotErase(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(path, []byte("x"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	m := NewModel(dir)
	m.browser.ToggleSelect()
	updated, _ := m.handleBrowsingKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	m = updated.(Model)

	m.passesInput.SetValue("not-a-number")
	m.confirmFocus = 2

	updated, _ = m.handleConfirmKey(tea.KeyMsg{Type: tea.KeyEnter})
	got := updated.(Model)
	if got.screen != screenConfirm {
		t.Fatalf("got screen %v, want to stay on screenConfirm", got.screen)
	}
	if got.confirmErr == "" {
		t.Fatal("expected a validation error message")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file to survive a failed validation, got err=%v", err)
	}
}

func TestConfirmScreen_TriggerStartsErasingScreen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(path, []byte("secret"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	m := NewModel(dir)
	m.browser.ToggleSelect()
	updated, _ := m.handleBrowsingKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	m = updated.(Model)
	m.confirmFocus = 2

	updated, _ = m.handleConfirmKey(tea.KeyMsg{Type: tea.KeyEnter})
	got := updated.(Model)

	if got.screen != screenErasing {
		t.Fatalf("got screen %v, want screenErasing", got.screen)
	}
	if len(got.targets) != 1 || got.targets[0] != path {
		t.Fatalf("got targets %v, want [%s]", got.targets, path)
	}
	if got.eventsCh == nil {
		t.Fatal("expected eventsCh to be set")
	}

	// Drain the real shred so the temp dir cleans up without a race.
	for range got.eventsCh {
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected file to eventually be gone, got err=%v", err)
	}
}

func TestErasingScreen_TicksWaitForEventBeforeAdvancing(t *testing.T) {
	m := Model{
		screen:  screenErasing,
		targets: []string{"/tmp/a", "/tmp/b"},
	}

	for i := 0; i < dissolveFrameCount+3; i++ {
		updated, _ := m.Update(dissolveTickMsg{})
		m = updated.(Model)
	}
	if m.dissolveFrame != dissolveFrameCount {
		t.Fatalf("got dissolveFrame %d, want it capped at %d", m.dissolveFrame, dissolveFrameCount)
	}
	if m.targetIdx != 0 {
		t.Fatal("expected targetIdx to hold at 0 until the real event for target 0 arrives")
	}

	updated, _ := m.Update(eraseEventMsg{Path: "/tmp/a", Result: shred.Result{FilesShredded: 1}})
	m = updated.(Model)
	if !m.targetDone {
		t.Fatal("expected targetDone once the event for the current target arrives")
	}

	updated, _ = m.Update(dissolveTickMsg{})
	m = updated.(Model)
	if m.targetIdx != 1 {
		t.Fatalf("got targetIdx %d, want 1", m.targetIdx)
	}
	if m.dissolveFrame != 0 {
		t.Fatalf("got dissolveFrame %d, want reset to 0 for the new target", m.dissolveFrame)
	}
	if m.targetDone {
		t.Fatal("expected targetDone to reset for the new target")
	}

	updated, _ = m.Update(eraseEventMsg{Path: "/tmp/b", Result: shred.Result{FilesShredded: 1}})
	m = updated.(Model)
	for i := 0; i < dissolveFrameCount+1; i++ {
		updated, _ = m.Update(dissolveTickMsg{})
		m = updated.(Model)
	}
	if m.screen != screenDone {
		t.Fatalf("got screen %v, want screenDone after the last target finishes", m.screen)
	}

	updated, _ = m.Update(eraseEventMsg{Done: true, Result: shred.Result{FilesShredded: 2}})
	m = updated.(Model)
	if m.result.FilesShredded != 2 {
		t.Fatalf("got FilesShredded=%d, want 2", m.result.FilesShredded)
	}
}
