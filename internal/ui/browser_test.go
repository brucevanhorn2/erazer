package ui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/brucevanhorn2/erazer/internal/browse"
)

func TestBrowserPane_SelectionPersistsAcrossNavigation(t *testing.T) {
	home := t.TempDir()
	sub := filepath.Join(home, "sub")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, "file.txt"), []byte("x"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	sel := browse.NewSet()
	b := NewBrowserPane(home, sel)
	if err := b.Refresh(); err != nil {
		t.Fatalf("Refresh: %v", err)
	}

	if len(b.Entries) == 0 || b.Entries[0].Name != "sub" {
		t.Fatalf("expected first entry to be 'sub', got %+v", b.Entries)
	}
	b.Cursor = 0
	b.ToggleSelect()
	if !sel.Effective(sub) {
		t.Fatal("expected sub to be selected after ToggleSelect")
	}

	if err := b.Enter(); err != nil {
		t.Fatalf("Enter: %v", err)
	}
	if b.Cwd != sub {
		t.Fatalf("expected Cwd to be %q, got %q", sub, b.Cwd)
	}
	if err := b.Back(); err != nil {
		t.Fatalf("Back: %v", err)
	}
	if b.Cwd != home {
		t.Fatalf("expected Cwd to be back at %q, got %q", home, b.Cwd)
	}

	if !sel.Effective(sub) {
		t.Fatal("expected selection to survive navigating into and back out of the folder")
	}
}

func TestBrowserPane_CurrentPath(t *testing.T) {
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, "a.txt"), []byte("x"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	sel := browse.NewSet()
	b := NewBrowserPane(home, sel)
	if err := b.Refresh(); err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if got := b.CurrentPath(); got != filepath.Join(home, "a.txt") {
		t.Fatalf("got %q, want %q", got, filepath.Join(home, "a.txt"))
	}
}

func TestBrowserPane_UpDown_StaysWithinBounds(t *testing.T) {
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, "a.txt"), []byte("x"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	sel := browse.NewSet()
	b := NewBrowserPane(home, sel)
	if err := b.Refresh(); err != nil {
		t.Fatalf("Refresh: %v", err)
	}

	b.Up()
	if b.Cursor != 0 {
		t.Fatalf("got Cursor %d, want 0", b.Cursor)
	}
	b.Down()
	if b.Cursor != 0 {
		t.Fatalf("got Cursor %d, want 0", b.Cursor)
	}
}
