package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/brucevanhorn2/erazer/internal/browse"
)

// Entry is one file or directory listed in the current directory.
type Entry struct {
	Name  string
	IsDir bool
}

// BrowserPane is a single-pane, navigable file listing. Selection lives in
// a shared *browse.Set rather than a per-directory map, so Refresh never
// clears it: a selection made here survives navigating to a different
// directory and back.
type BrowserPane struct {
	Cwd       string
	Entries   []Entry
	Cursor    int
	Selection *browse.Set
	Width     int
	Height    int
	scrollTop int
}

func NewBrowserPane(start string, sel *browse.Set) *BrowserPane {
	return &BrowserPane{Cwd: start, Selection: sel}
}

// Refresh reloads the current directory's listing (directories first, then
// alphabetical), resetting the cursor and scroll position but leaving
// Selection untouched.
func (b *BrowserPane) Refresh() error {
	dirEntries, err := os.ReadDir(b.Cwd)
	if err != nil {
		return err
	}
	entries := make([]Entry, 0, len(dirEntries))
	for _, e := range dirEntries {
		entries = append(entries, Entry{Name: e.Name(), IsDir: e.IsDir()})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return entries[i].Name < entries[j].Name
	})
	b.Entries = entries
	b.Cursor = 0
	b.scrollTop = 0
	return nil
}

func (b *BrowserPane) Up() {
	if b.Cursor > 0 {
		b.Cursor--
	}
	b.ensureVisible()
}

func (b *BrowserPane) Down() {
	if b.Cursor < len(b.Entries)-1 {
		b.Cursor++
	}
	b.ensureVisible()
}

func (b *BrowserPane) ensureVisible() {
	visibleRows := b.Height - 3
	if visibleRows < 1 {
		visibleRows = 1
	}
	if b.Cursor < b.scrollTop {
		b.scrollTop = b.Cursor
	}
	if b.Cursor >= b.scrollTop+visibleRows {
		b.scrollTop = b.Cursor - visibleRows + 1
	}
}

// CurrentPath returns the absolute path of the entry under the cursor, or
// "" if there is none.
func (b *BrowserPane) CurrentPath() string {
	if b.Cursor < 0 || b.Cursor >= len(b.Entries) {
		return ""
	}
	return filepath.Join(b.Cwd, b.Entries[b.Cursor].Name)
}

func (b *BrowserPane) Enter() error {
	if b.Cursor < 0 || b.Cursor >= len(b.Entries) {
		return nil
	}
	e := b.Entries[b.Cursor]
	if !e.IsDir {
		return nil
	}
	oldCwd := b.Cwd
	b.Cwd = filepath.Join(b.Cwd, e.Name)
	if err := b.Refresh(); err != nil {
		b.Cwd = oldCwd
		return err
	}
	return nil
}

func (b *BrowserPane) Back() error {
	parent := filepath.Dir(b.Cwd)
	if parent == b.Cwd {
		return nil
	}
	oldCwd := b.Cwd
	b.Cwd = parent
	if err := b.Refresh(); err != nil {
		b.Cwd = oldCwd
		return err
	}
	return nil
}

func (b *BrowserPane) ToggleSelect() {
	path := b.CurrentPath()
	if path == "" {
		return
	}
	b.Selection.Toggle(path)
}

func (b *BrowserPane) View(theme Theme) string {
	titleLine := gradientText(fmt.Sprintf(" %s ", b.Cwd), theme.PrimaryColor, theme.SecondaryColor)
	lines := []string{titleLine}

	contentHeight := b.Height - 3
	if contentHeight < 0 {
		contentHeight = 0
	}

	rowsRendered := 0
	for i := b.scrollTop; i < len(b.Entries) && i < b.scrollTop+contentHeight; i++ {
		e := b.Entries[i]
		path := filepath.Join(b.Cwd, e.Name)
		effective := b.Selection.Effective(path)
		explicit := b.Selection.IsExplicit(path)

		cursorMark := " "
		if i == b.Cursor {
			cursorMark = "►"
		}
		selectMark := " "
		switch {
		case effective && explicit:
			selectMark = "☑"
		case effective:
			selectMark = "·"
		}
		marker := cursorMark + selectMark + " "

		style := theme.BrowserFile
		if e.IsDir {
			style = theme.BrowserDir
		}
		if effective {
			style = theme.BrowserSelected
		}

		line := marker + style.Render(e.Name)
		if e.IsDir {
			line += "/"
		}
		lines = append(lines, line)
		rowsRendered++
	}
	for ; rowsRendered < contentHeight; rowsRendered++ {
		lines = append(lines, "")
	}

	content := strings.Join(lines, "\n")
	return gradientBox(content, b.Width, b.Height-2, theme.PrimaryColor, theme.SecondaryColor)
}
