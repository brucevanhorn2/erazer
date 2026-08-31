package ui

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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

func TestConfirmScreen_TriggerErasesSelectedFileAndReachesDone(t *testing.T) {
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

	if got.screen != screenDone {
		t.Fatalf("got screen %v, want screenDone", got.screen)
	}
	if got.result.FilesShredded != 1 {
		t.Fatalf("got FilesShredded=%d, want 1", got.result.FilesShredded)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected file to be gone, got err=%v", err)
	}
}
