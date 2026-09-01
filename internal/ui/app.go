package ui

import (
	"strconv"
	"strings"
	"time"

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
	screenErasing
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

	// statusErr holds the error message from the most recent failed
	// browsing-screen navigation (Enter/Back), shown inline on the
	// Browsing screen. It does not end the session — BrowserPane rolls
	// Cwd back on failure, so there's no state corruption to escalate to.
	statusErr string

	targets   []string
	targetIdx int
	opts      shred.Options
	eventsCh  chan shred.Event

	dissolveFrame int
	targetDone    bool
	// failed[i] is true once the real shred.Event for targets[i] has
	// arrived and reported at least one error. Indexed by eventsReceived
	// at the time each event arrives (events arrive in target order — see
	// ShredAll), independent of how far the tick-driven animation has
	// gotten. Read by erasingView to distinguish a genuinely erased target
	// from one that failed and still exists.
	failed         []bool
	eventsReceived int
	// pendingDone counts real per-target completions (shred.Event with
	// Done=false) that have arrived but not yet been consumed by the
	// animation gate in handleDissolveTick. shred.ShredAll runs
	// unthrottled while the animation is paced to dissolveInterval, so
	// events for several targets — even all of them — can arrive before
	// the first target's animation finishes. Without this counter,
	// targetDone (a single bool) gets set true by whichever event arrives
	// next regardless of which target it's for, and a later burst event
	// can be silently dropped once already true — permanently losing the
	// completion signal for a subsequent target and hanging the UI short
	// of screenDone. Counting arrivals instead of latching a bool makes
	// each tick-driven advance consume exactly one already-arrived
	// completion, in order, however far ahead of the animation they came.
	pendingDone int

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

	case eraseEventMsg:
		return m.handleEraseEvent(msg)

	case dissolveTickMsg:
		return m.handleDissolveTick()
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
			m.statusErr = err.Error()
		} else {
			m.statusErr = ""
		}
	case "backspace", "h", "left":
		if err := m.browser.Back(); err != nil {
			m.statusErr = err.Error()
		} else {
			m.statusErr = ""
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

// startErase validates the Confirm screen's settings, then kicks off an
// async shred of every target: a goroutine runs shred.ShredAll and streams
// one shred.Event per target (plus a final aggregate event) back over
// eventsCh, consumed via the same channel-plus-re-arming-Cmd "subscription"
// pattern exfil/sneakernet use for transfer/backup progress. The Erasing
// screen's own dissolveTick paces the animation independently; each
// target only advances once both its animation has run its course AND its
// real shred.Event has arrived (handleEraseEvent/handleDissolveTick
// below), so the UI can never show a target as erazed before it actually
// is.
func (m Model) startErase() (tea.Model, tea.Cmd) {
	opts, errMsg := m.parseConfirmSettings()
	if errMsg != "" {
		m.confirmErr = errMsg
		return m, nil
	}
	m.opts = opts
	// Capture only the *browse.Set pointer, not the whole Model — the
	// closure below is retained by the shred goroutine (via m.opts.Skip)
	// for the life of the run, and closing over m directly would keep the
	// BrowserPane's listing and both textinputs alive for no reason.
	sel := m.selection
	m.opts.Skip = func(path string) bool { return !sel.Effective(path) }
	m.targetIdx = 0
	m.dissolveFrame = 0
	m.targetDone = false
	m.pendingDone = 0
	m.failed = make([]bool, len(m.targets))
	m.eventsReceived = 0
	m.result = shred.Result{}
	m.eventsCh = make(chan shred.Event, len(m.targets)+1)
	go shred.ShredAll(m.targets, m.opts, m.eventsCh)
	m.screen = screenErasing
	return m, tea.Batch(waitForShredEvent(m.eventsCh), dissolveTick())
}

// eraseEventMsg wraps shred.Event as a distinct type so Bubble Tea's
// type-based dispatch in Update routes it correctly.
type eraseEventMsg shred.Event

func waitForShredEvent(ch chan shred.Event) tea.Cmd {
	return func() tea.Msg {
		evt, ok := <-ch
		if !ok {
			return eraseEventMsg{Done: true}
		}
		return eraseEventMsg(evt)
	}
}

type dissolveTickMsg struct{}

func dissolveTick() tea.Cmd {
	return tea.Tick(dissolveInterval, func(time.Time) tea.Msg { return dissolveTickMsg{} })
}

func (m Model) handleEraseEvent(msg eraseEventMsg) (tea.Model, tea.Cmd) {
	evt := shred.Event(msg)
	if evt.Done {
		m.result = evt.Result
		return m, nil
	}
	// Count the arrival rather than latch a single bool: shred.ShredAll
	// can finish several (or all) targets before the animation catches up
	// to even the first one, and a second arrival must not be lost just
	// because targetDone was already true from the first.
	if m.eventsReceived < len(m.failed) {
		m.failed[m.eventsReceived] = len(evt.Result.Errors) > 0
	}
	m.eventsReceived++
	m.pendingDone++
	m.targetDone = true
	return m, waitForShredEvent(m.eventsCh)
}

func (m Model) handleDissolveTick() (tea.Model, tea.Cmd) {
	if m.screen != screenErasing {
		return m, nil
	}
	if m.dissolveFrame < dissolveFrameCount {
		m.dissolveFrame++
	}
	if m.dissolveFrame >= dissolveFrameCount && m.targetDone {
		m.targetIdx++
		m.dissolveFrame = 0
		m.pendingDone--
		m.targetDone = m.pendingDone > 0
		if m.targetIdx >= len(m.targets) {
			m.screen = screenDone
			return m, nil
		}
	}
	return m, dissolveTick()
}

func (m Model) handleDoneKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.quitting = true
	return m, tea.Quit
}
