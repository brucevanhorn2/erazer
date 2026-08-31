package ui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/brucevanhorn2/erazer/internal/browse"
	"github.com/brucevanhorn2/erazer/internal/shred"
)

type screen int

const (
	screenBrowsing screen = iota
	screenAbout
	screenConfirm
	screenDone
)

const defaultPasses = 3

// Model is erazer's whole application state: which screen is active, the
// shared selection tree, and per-screen data.
type Model struct {
	screen screen
	theme  Theme
	width  int
	height int

	selection *browse.Set
	browser   *BrowserPane
	about     *AboutPane

	passesInput  textinput.Model
	seedInput    textinput.Model
	confirmFocus int // 0 = passes field, 1 = seed field, 2 = ERAZE trigger
	confirmErr   string
	rotational   bool
	rotationalOK bool

	targets []string
	result  shred.Result
	doneErr string

	quitting bool
}

// NewModel builds the initial model, starting the browser at start (the
// entrypoint passes the user's home directory).
func NewModel(start string) Model {
	sel := browse.NewSet()
	browser := NewBrowserPane(start, sel)
	_ = browser.Refresh()

	passesInput := textinput.New()
	passesInput.Placeholder = strconv.Itoa(defaultPasses)
	passesInput.CharLimit = 3
	passesInput.SetValue(strconv.Itoa(defaultPasses))

	seedInput := textinput.New()
	seedInput.Placeholder = "blank = crypto/rand"
	seedInput.CharLimit = 20

	return Model{
		screen:      screenBrowsing,
		theme:       NewTheme(),
		selection:   sel,
		browser:     browser,
		about:       NewAboutPane(),
		passesInput: passesInput,
		seedInput:   seedInput,
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.browser.Width = msg.Width
		m.browser.Height = msg.Height - 2
		m.about.Width = msg.Width
		m.about.Height = msg.Height - 2
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}
		switch m.screen {
		case screenBrowsing:
			return m.handleBrowsingKey(msg)
		case screenAbout:
			return m.handleAboutKey(msg)
		case screenConfirm:
			return m.handleConfirmKey(msg)
		case screenDone:
			return m.handleDoneKey(msg)
		}
	}
	return m, nil
}

func (m Model) handleBrowsingKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		m.quitting = true
		return m, tea.Quit
	case "up", "k":
		m.browser.Up()
	case "down", "j":
		m.browser.Down()
	case "enter", "l", "right":
		if err := m.browser.Enter(); err != nil {
			m.doneErr = err.Error()
			m.screen = screenDone
		}
	case "backspace", "h", "left":
		if err := m.browser.Back(); err != nil {
			m.doneErr = err.Error()
			m.screen = screenDone
		}
	case " ":
		m.browser.ToggleSelect()
	case "?":
		m.screen = screenAbout
	case "e":
		m.targets = m.selection.SelectedRoots()
		if len(m.targets) == 0 {
			return m, nil
		}
		m.rotational, m.rotationalOK = false, false
		for _, target := range m.targets {
			r, ok := shred.IsRotational(target)
			if !ok {
				continue
			}
			m.rotationalOK = true
			if !r {
				m.rotational = false
				break
			}
			m.rotational = true
		}
		m.confirmFocus = 0
		m.confirmErr = ""
		m.passesInput.Focus()
		m.seedInput.Blur()
		m.screen = screenConfirm
	}
	return m, nil
}

func (m Model) handleAboutKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.screen = screenBrowsing
	return m, nil
}

func (m Model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = screenBrowsing
		return m, nil
	case "tab", "down":
		m.confirmFocus = (m.confirmFocus + 1) % 3
		m.syncConfirmFocus()
		return m, nil
	case "shift+tab", "up":
		m.confirmFocus = (m.confirmFocus + 2) % 3
		m.syncConfirmFocus()
		return m, nil
	case "enter":
		if m.confirmFocus == 2 {
			return m.startErase()
		}
		m.confirmFocus = (m.confirmFocus + 1) % 3
		m.syncConfirmFocus()
		return m, nil
	}
	var cmd tea.Cmd
	switch m.confirmFocus {
	case 0:
		m.passesInput, cmd = m.passesInput.Update(msg)
	case 1:
		m.seedInput, cmd = m.seedInput.Update(msg)
	}
	return m, cmd
}

func (m *Model) syncConfirmFocus() {
	if m.confirmFocus == 0 {
		m.passesInput.Focus()
	} else {
		m.passesInput.Blur()
	}
	if m.confirmFocus == 1 {
		m.seedInput.Focus()
	} else {
		m.seedInput.Blur()
	}
}

// parseConfirmSettings validates the passes/seed fields, returning the
// parsed shred.Options or an error message fit to show on the Confirm
// screen.
func (m Model) parseConfirmSettings() (shred.Options, string) {
	passesStr := strings.TrimSpace(m.passesInput.Value())
	if passesStr == "" {
		passesStr = strconv.Itoa(defaultPasses)
	}
	passes, err := strconv.Atoi(passesStr)
	if err != nil || passes <= 0 {
		return shred.Options{}, "passes must be a positive whole number"
	}

	seedStr := strings.TrimSpace(m.seedInput.Value())
	var seed *int64
	if seedStr != "" {
		v, err := strconv.ParseInt(seedStr, 10, 64)
		if err != nil {
			return shred.Options{}, "seed must be a whole number"
		}
		seed = &v
	}
	return shred.Options{Passes: passes, Seed: seed}, ""
}

// startErase runs the shred synchronously and moves straight to Done. Task
// 10 replaces this body with an async, animated version — the Confirm ->
// Done contract (m.result, m.screen) stays the same either way.
func (m Model) startErase() (tea.Model, tea.Cmd) {
	opts, errMsg := m.parseConfirmSettings()
	if errMsg != "" {
		m.confirmErr = errMsg
		return m, nil
	}
	m.result = shred.Result{}
	for _, target := range m.targets {
		res := shred.Shred(target, opts)
		m.result.FilesShredded += res.FilesShredded
		m.result.BytesOverwritten += res.BytesOverwritten
		m.result.Errors = append(m.result.Errors, res.Errors...)
	}
	m.screen = screenDone
	return m, nil
}

func (m Model) handleDoneKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.quitting = true
	return m, tea.Quit
}
