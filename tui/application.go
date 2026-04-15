package tui

import (
	"bufio"
	"fmt"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type (
	errorMsg          string
	previewMsg        string
	tickMsg           time.Time
	clearInputTextMsg struct{}
	paneClosedMsg     struct{}

	listEntitiesMsg struct {
		sessions       []TmuxEntity
		windows        []TmuxEntity
		panes          []TmuxEntity
		currentSession int
		currentWindow  int
		currentPane    int
	}
)

const (
	None InputAction = iota
	Filter
	NewSession
	NewWindow
	RenameSession
	RenameWindow
	LiveInput
	Help
	DirPicker
)

type ScreenMode int

const (
	ScreenNormal ScreenMode = iota
	ScreenHalf
	ScreenFull
)

type (
	terminal struct {
		width  int
		height int
	}

	TmuxEntity struct {
		id     int
		name   string
		parent int
	}

	InputAction int

	AppModel struct {
		Error string

		terminal terminal
		theme    Theme

		preview  Frame
		sessions ListFrame
		windows  ListFrame
		panes    ListFrame

		focusedFrame int

		showAll         bool
		showHelp        bool
		maximizePreview bool
		swapSrc         int
		screenMode      ScreenMode
		scrollHeight    int

		textInput   textinput.Model
		inputAction InputAction
		filter          string
	liveInputTarget int

		dirPicker DirPickerModel
	}
)

func NewApplication(theme Theme) *tea.Program {
	model := AppModel{
		Error:        "",
		terminal:     terminal{80, 80},
		theme:        theme,
		preview:      Frame{title: "Preview", scrollable: true},
		sessions:     ListFrame{frame: Frame{title: "Sessions", focused: true}, parentId: -1, showNumbers: true},
		windows:      ListFrame{frame: Frame{title: "Windows"}, parentId: -1, showNumbers: true},
		panes:        ListFrame{frame: Frame{title: "Panes"}, parentId: -1, showNumbers: true},
		focusedFrame: 1,
		showAll:      false,
		swapSrc:      -1,
		inputAction:  None,
		screenMode:   ScreenHalf,
		scrollHeight: 5,
	}

	model.textInput = textinput.New()
	model.textInput.Focus()
	model.textInput.TextStyle = theme.NewStyle()

	return tea.NewProgram(model, tea.WithAltScreen())
}

func (m AppModel) Init() tea.Cmd {
	return tea.Batch(tickCmd(), listEntitiesCmd)
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd = nil

	if m.swapSrc != -1 {
		goto swap_mode
	}

	if m.inputAction != None {
		goto input_mode
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case tea.KeyEsc.String():
			cmd = tea.Quit
		case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9":
			idx := int(msg.String()[0] - '0')
			var listFrame *ListFrame
			switch m.focusedFrame {
			case 1:
				listFrame = &m.sessions
			case 2:
				listFrame = &m.windows
			case 3:
				listFrame = &m.panes
			}
			visibleItems := listFrame.VisibleItems()
			if idx > 0 && idx <= len(visibleItems) {
				listFrame.currentId = visibleItems[idx-1].id
				cmd = previewCmd(m)
			}
		case tea.KeyTab.String():
			// Cycle forward: 1->2->3->1
			m.focusedFrame = m.focusedFrame%3 + 1
			m.sessions.frame.focused = (m.focusedFrame == 1)
			m.windows.frame.focused = (m.focusedFrame == 2)
			m.panes.frame.focused = (m.focusedFrame == 3)
			cmd = previewCmd(m)
		case tea.KeyShiftTab.String():
			// Cycle backward: 3->2->1->3
			m.focusedFrame = m.focusedFrame - 1
			if m.focusedFrame < 1 {
				m.focusedFrame = 3
			}
			m.sessions.frame.focused = (m.focusedFrame == 1)
			m.windows.frame.focused = (m.focusedFrame == 2)
			m.panes.frame.focused = (m.focusedFrame == 3)
			cmd = previewCmd(m)
		case tea.KeyEnter.String():
			switch m.focusedFrame {
			case 1:
				cmd = goToSessionCmd(m)
			case 2:
				cmd = goToWindowCmd(m)
			case 3:
				cmd = goToPaneCmd(m)
			}
		case "d":
			switch m.focusedFrame {
			case 1:
				cmd = deleteSessionCmd(m)
			case 2:
				cmd = deleteWindowCmd(m)
			case 3:
				cmd = deletePaneCmd(m)
			}
		case "h":
			if m.focusedFrame == 3 {
				cmd = splitPane(m, true)
			}
		case "v":
			if m.focusedFrame == 3 {
				cmd = splitPane(m, false)
			}
		case "i":
			if m.focusedFrame >= 2 && m.panes.currentId != -1 {
				selfInfoCmd := exec.Command("tmux", "display-message", "-p", "#{pane_id}")
				selfInfoBytes, _ := selfInfoCmd.Output()
				selfPaneId := strings.TrimSpace(string(selfInfoBytes))
				targetPaneId := fmt.Sprintf("%%%d", m.panes.currentId)

				if selfPaneId != targetPaneId {
					m.inputAction = LiveInput
					m.liveInputTarget = m.panes.currentId
				}
			}
		case "r":
			switch m.focusedFrame {
			case 1:
				m.inputAction = RenameSession
				m.textInput.SetValue(m.sessions.ItemWithId(m.sessions.currentId).name)
				m.textInput.SetCursor(100)
			case 2:
				m.inputAction = RenameWindow
				m.textInput.SetValue(m.windows.ItemWithId(m.windows.currentId).name)
				m.textInput.SetCursor(100)
			}
		case "n":
			switch m.focusedFrame {
			case 1:
				m.inputAction = DirPicker
				m.dirPicker = NewDirPicker(m.theme)
				cmd = initDirPickerCmd(m.sessions.currentId)
			case 2:
				m.textInput.SetValue("")
				m.inputAction = NewWindow
			}
		case "N":
			switch m.focusedFrame {
			case 1:
				cmd = newSessionCmd(m)
			case 2:
				cmd = newWindowCmd(m)
			}
		case "R":
			if m.focusedFrame == 1 {
				cmd = refreshSessionCmd(m)
			}
		case "/":
			m.inputAction = Filter
			m.textInput.SetValue(m.filter)
			m.textInput.SetCursor(100)
		case "s":
			switch m.focusedFrame {
			case 2:
				m.swapSrc = m.windows.currentId
				m.windows.MarkSelection()
			case 3:
				m.swapSrc = m.panes.currentId
				m.panes.MarkSelection()
			}
		}
	case clearInputTextMsg:
		m.textInput.SetValue("")
		cmd = listEntitiesCmd
	}
	goto common_bindings

swap_mode:
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case tea.KeyEsc.String():
			m.swapSrc = -1
			m.windows.ClearMarks()
			m.panes.ClearMarks()
		case "s", tea.KeySpace.String(), tea.KeyEnter.String():
			m.windows.ClearMarks()
			m.panes.ClearMarks()
			switch m.focusedFrame {
			case 2:
				cmd = swapWindowsCmd(m, m.swapSrc)
			case 3:
				cmd = swapPanesCmd(m, m.swapSrc)
			}
			m.swapSrc = -1
		}
	}
	goto common_bindings

input_mode:
	if m.inputAction == LiveInput {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.String() == "ctrl+q" {
				m.inputAction = None
				m.liveInputTarget = -1
			} else {
				cmd = sendKeyToPaneCmd(m, msg)
			}
		}
		goto basic_handlers
	}

	if m.inputAction == DirPicker {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case tea.KeyEsc.String(), "ctrl+c":
				m.inputAction = None
				goto basic_handlers
			case tea.KeyUp.String(), "ctrl+p":
				m.dirPicker.moveCursor(-1)
				goto basic_handlers
			case tea.KeyDown.String(), "ctrl+n":
				m.dirPicker.moveCursor(1)
				goto basic_handlers
			case tea.KeyEnter.String():
				var done bool
				cmd, done = m.dirPicker.handleEnter(m.sessions.items)
				if done {
					m.inputAction = None
				}
				goto basic_handlers
			case tea.KeyTab.String():
				m.dirPicker.showHidden = !m.dirPicker.showHidden
				m.dirPicker.clampCursor()
				goto basic_handlers
			}
		}
		oldFilter := m.dirPicker.filterInput.Value()
		m.dirPicker.filterInput, cmd = m.dirPicker.filterInput.Update(msg)
		if m.dirPicker.filterInput.Value() != oldFilter {
			m.dirPicker.cursor = 0
		}
		goto basic_handlers
	}

	m.textInput, cmd = m.textInput.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case tea.KeyEsc.String():
			if m.inputAction == Filter {
				m.filter = ""
				m.sessions.filterText = ""
				m.windows.filterText = ""
				m.panes.filterText = ""
			}
			m.inputAction = None
			m.textInput.SetValue("")
		case tea.KeyEnter.String():
			switch m.inputAction {
			case NewSession:
				cmd = newSessionCmd(m)
			case RenameSession:
				cmd = renameSessionCmd(m)
			case NewWindow:
				cmd = newWindowCmd(m)
			case RenameWindow:
				cmd = renameWindowCmd(m)
			case Filter:
				m.filter = m.textInput.Value()
			}
			m.inputAction = None
		}
	}

	if m.inputAction == Filter {
		m.sessions.filterText = m.textInput.Value()
		m.windows.filterText = m.textInput.Value()
		m.panes.filterText = m.textInput.Value()
	}

	goto basic_handlers

common_bindings:
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			cmd = tea.Quit
		case "ctrl+p", "k", tea.KeyUp.String():
			switch m.focusedFrame {
			case 1:
				m.sessions.SelectPrevious()
			case 2:
				m.windows.SelectPrevious()
			case 3:
				m.panes.SelectPrevious()
			}
			cmd = listEntitiesCmd
		case "ctrl+n", "j", tea.KeyDown.String():
			switch m.focusedFrame {
			case 1:
				m.sessions.SelectNext()
			case 2:
				m.windows.SelectNext()
			case 3:
				m.panes.SelectNext()
			}
			cmd = listEntitiesCmd
		case "ctrl+u", "K":
			// Scroll preview up
			m.preview.ScrollUp(m.scrollHeight)
		case "ctrl+d", "J":
			// Scroll preview down
			m.preview.ScrollDown(m.scrollHeight)
		case "+", "=":
			// Next screen mode (normal -> half -> full)
			m.nextScreenMode()
		case "_", "-":
			// Previous screen mode (full -> half -> normal)
			m.prevScreenMode()
		case "a":
			m.showAll = !m.showAll
			cmd = listEntitiesCmd
		case "?":
			m.showHelp = !m.showHelp
		case "m":
			m.maximizePreview = !m.maximizePreview
		case "t":
			m.toggleTransparent()
		}
	}

basic_handlers:
	switch msg := msg.(type) {
	case tickMsg:
		cmd = tea.Batch(tickCmd(), listEntitiesCmd)
	case tea.WindowSizeMsg:
		m.terminal.width = msg.Width
		m.terminal.height = msg.Height
	case listEntitiesMsg:
		m.sessions.items = msg.sessions
		m.windows.items = msg.windows
		m.panes.items = msg.panes
		if m.sessions.currentId == -1 && len(m.filter) == 0 {
			m.sessions.currentId = msg.currentSession
			m.windows.currentId = msg.currentWindow
			m.panes.currentId = msg.currentPane
		}
		cmd = previewCmd(m)
	case previewMsg:
		m.preview.contents = string(msg)
	case dirEntriesMsg:
		if m.inputAction == DirPicker {
			m.dirPicker.currentPath = msg.path
			m.dirPicker.entries = msg.entries
			m.dirPicker.err = msg.err
			m.dirPicker.cursor = 0
			m.dirPicker.filterInput.SetValue("")
		}
	case paneClosedMsg:
		if m.inputAction == LiveInput {
			m.inputAction = None
			m.liveInputTarget = -1
			m.Error = "Target pane was closed"
		}
	}

	if m.showAll {
		m.windows.parentId = -1
		m.panes.parentId = -1
	} else {
		m.windows.parentId = m.sessions.currentId
		m.panes.parentId = m.windows.currentId
	}

	m.sessions.Update()
	m.windows.Update()
	m.panes.Update()

	return m, cmd
}

func (m AppModel) View() string {
	// If showing help, render help overlay
	if m.showHelp {
		return m.HelpView()
	}

	// If directory picker is open, render it as overlay
	if m.inputAction == DirPicker {
		return m.dirPicker.View(m.theme, m.terminal.width, m.terminal.height)
	}

	preview := m.preview

	sessions := m.sessions.RenderContents(m.theme)
	windows := m.windows.RenderContents(m.theme)
	panes := m.panes.RenderContents(m.theme)

	sessions.title = "Sessions"
	windows.title = "Windows"
	panes.title = "Panes"

	m.textInput.Width = m.terminal.width - 4
	var status = Frame{
		title:    "New name",
		contents: m.textInput.View(),
		width:    m.terminal.width,
		height:   m.terminal.height,
		focused:  true,
	}

	switch m.inputAction {
	case None:
		status = m.StatusBar()
	case Filter:
		status.title = "Filter"
	case LiveInput:
		status.title = "LIVE INPUT MODE - Ctrl+Q to exit"
		status.contents = ""
	}

	return m.DrawGrid(preview, sessions, windows, panes, status)
}

func (m AppModel) StatusBar() Frame {
	frame := Frame{title: "Status"}
	normalStyle := m.theme.NewStyle()
	accentStyle := normalStyle.Copy().Foreground(m.theme.Accent)
	left := []string{normalStyle.Render("Help: ?"), normalStyle.Render("Quit: q")}

	if m.swapSrc == -1 {
		left = append(left, normalStyle.Render("Go to: <enter>"))
		left = append(left, normalStyle.Render("Delete: d"))
		left = append(left, normalStyle.Render("Swap: s"))
		if m.focusedFrame == 1 {
			left = append(left, normalStyle.Render("New (dir): n"))
			left = append(left, normalStyle.Render("New (nameless): N"))
			left = append(left, normalStyle.Render("Rename: r"))
			left = append(left, normalStyle.Render("Refresh: R"))
		} else if m.focusedFrame == 2 {
			left = append(left, normalStyle.Render("New: n"))
			left = append(left, normalStyle.Render("New (nameless): N"))
			left = append(left, normalStyle.Render("Rename: r"))
			left = append(left, normalStyle.Render("Live input: i"))
		} else {
			left = append(left, normalStyle.Render("Vertical split: v"))
			left = append(left, normalStyle.Render("Horizontal split: h"))
			left = append(left, normalStyle.Render("Live input: i"))
		}
	} else {
		left = append(left, accentStyle.Render("Swap: s/<space>/<enter>"))
		left = append(left, normalStyle.Render("Cancel: <esc>"))
	}

	if m.showAll {
		left = append(left, accentStyle.Render("Show all: a"))
	} else {
		left = append(left, normalStyle.Render("Show all: a"))
	}

	if m.swapSrc == -1 {
		if len(m.textInput.Value()) > 0 {
			left = append(left, accentStyle.Render("Filter: /"))
		} else {
			left = append(left, normalStyle.Render("Filter: /"))
		}
	}

	if m.theme.Transparent {
		left = append(left, accentStyle.Render("Transparent: t"))
	} else {
		left = append(left, normalStyle.Render("Transparent: t"))
	}

	rightString := normalStyle.Foreground(m.theme.Secondary).Render(strings.TrimSpace(Version))

	separator := normalStyle.Render(" | ")
	maxWidth := uint(m.terminal.width - 7 - lipgloss.Width(rightString))
	leftString := left[0]
	for i, v := range left {
		if i == 0 {
			continue
		}
		newWidth := lipgloss.Width(leftString) + 3 + lipgloss.Width(v)
		if newWidth <= int(maxWidth) {
			leftString = fmt.Sprintf("%s%s%s", leftString, separator, v)
		}
	}
	leftString = normalStyle.Width(int(maxWidth)).Render(leftString)

	frame.contents = leftString + normalStyle.Render(" ") + rightString
	return frame
}

// nextScreenMode cycles to the next screen mode: Normal -> Half -> Full -> Normal
func (m *AppModel) nextScreenMode() {
	modes := []ScreenMode{ScreenNormal, ScreenHalf, ScreenFull}
	for i, mode := range modes {
		if mode == m.screenMode {
			if i == len(modes)-1 {
				m.screenMode = modes[0]
			} else {
				m.screenMode = modes[i+1]
			}
			return
		}
	}
	m.screenMode = ScreenNormal
}

// prevScreenMode cycles to the previous screen mode: Full -> Half -> Normal -> Full
func (m *AppModel) prevScreenMode() {
	modes := []ScreenMode{ScreenNormal, ScreenHalf, ScreenFull}
	for i, mode := range modes {
		if mode == m.screenMode {
			if i == 0 {
				m.screenMode = modes[len(modes)-1]
			} else {
				m.screenMode = modes[i-1]
			}
			return
		}
	}
	m.screenMode = ScreenFull
}

func (m *AppModel) toggleTransparent() {
	m.theme.Transparent = !m.theme.Transparent
	if !m.theme.Transparent && m.theme.Background == "" {
		// Turning transparency OFF: set a default black background
		m.theme.Background = lipgloss.Color("0")
	} else if m.theme.Transparent {
		// Turning transparency ON: clear the background color
		m.theme.Background = lipgloss.Color("")
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second*1, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func listEntitiesCmd() tea.Msg {
	// Fetches info about all sessions, windows and panes at once
	c := exec.Command("tmux",
		"list-panes", "-aF", "#{session_id}\t#{window_id}\t#{pane_id}\t#{session_name}\t#{window_name}\t#{pane_current_command}", ";",
		"display-message", "-p", "#{session_id}\t#{window_id}\t#{pane_id}")
	bytes, err := c.Output()
	if err != nil {
		return nil
	}

	sessions := []TmuxEntity{}
	windows := []TmuxEntity{}
	panes := []TmuxEntity{}

	currentSession := 0
	currentWindow := 0
	currentPane := 0

	scanner := bufio.NewScanner(strings.NewReader(string(bytes[:])))
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), "\t")

		session_id, err := strconv.Atoi(strings.Replace(parts[0], "$", "", 1))
		if err != nil {
			continue
		}

		window_id, err := strconv.Atoi(strings.Replace(parts[1], "@", "", 1))
		if err != nil {
			continue
		}

		pane_id, err := strconv.Atoi(strings.Replace(parts[2], "%", "", 1))
		if err != nil {
			continue
		}

		if len(parts) == 3 {
			currentSession = session_id
			currentWindow = window_id
			currentPane = pane_id
			continue
		}

		session_name := parts[3]
		window_name := parts[4]
		pane_name := parts[5]

		sessions = append(sessions, TmuxEntity{session_id, session_name, -1})
		windows = append(windows, TmuxEntity{window_id, window_name, session_id})
		panes = append(panes, TmuxEntity{pane_id, pane_name, window_id})
	}

	if len(sessions) == 0 {
		return errorMsg("No sessions found. Is tmux running?")
	}

	eq := func(a, b TmuxEntity) bool {
		return a.id == b.id
	}

	sessions = slices.CompactFunc(sessions, eq)
	windows = slices.CompactFunc(windows, eq)
	panes = slices.CompactFunc(panes, eq)

	msg := listEntitiesMsg{}

	msg.sessions = sessions
	msg.windows = windows
	msg.panes = panes

	msg.currentSession = currentSession
	msg.currentWindow = currentWindow
	msg.currentPane = currentPane

	return msg
}

func previewCmd(m AppModel) tea.Cmd {
	return func() tea.Msg {
		// Get our own pane/window/session IDs to detect self-preview
		selfInfoCmd := exec.Command("tmux", "display-message", "-p", "#{session_id}\t#{window_id}\t#{pane_id}")
		selfInfoBytes, err := selfInfoCmd.Output()
		if err != nil {
			// If we can't get our own info, proceed normally
			selfInfoBytes = []byte("")
		}
		selfInfo := strings.TrimSpace(string(selfInfoBytes))
		parts := strings.Split(selfInfo, "\t")

		var selfSessionId, selfWindowId, selfPaneId string
		if len(parts) == 3 {
			selfSessionId = parts[0]
			selfWindowId = parts[1]
			selfPaneId = parts[2]
		}

		id := ""
		switch m.focusedFrame {
		case 1:
			id = fmt.Sprintf("$%d", m.sessions.currentId)
			// Check if previewing our own session
			if selfSessionId != "" && id == selfSessionId {
				return previewMsg("")
			}
		case 2:
			id = fmt.Sprintf("@%d", m.windows.currentId)
			// Check if previewing our own window
			if selfWindowId != "" && id == selfWindowId {
				return previewMsg("")
			}
		case 3:
			id = fmt.Sprintf("%%%d", m.panes.currentId)
			// Check if previewing our own pane
			if selfPaneId != "" && id == selfPaneId {
				return previewMsg("")
			}
		}

		c := exec.Command("tmux", "capture-pane", "-ep", "-t", id)
		bytes, err := c.Output()
		if err != nil {
			return nil
		}
		preview := string(bytes[:])

		return previewMsg(preview)
	}
}

func mapKeyToTmuxSendKeys(msg tea.KeyMsg) string {
	switch msg.Type {
	case tea.KeyEnter:
		return "Enter"
	case tea.KeyTab:
		return "Tab"
	case tea.KeyBackspace:
		return "BSpace"
	case tea.KeyUp:
		return "Up"
	case tea.KeyDown:
		return "Down"
	case tea.KeyLeft:
		return "Left"
	case tea.KeyRight:
		return "Right"
	case tea.KeySpace:
		return "Space"
	case tea.KeyEsc:
		return "Escape"
	case tea.KeyCtrlA:
		return "C-a"
	case tea.KeyCtrlB:
		return "C-b"
	case tea.KeyCtrlC:
		return "C-c"
	case tea.KeyCtrlD:
		return "C-d"
	case tea.KeyCtrlE:
		return "C-e"
	case tea.KeyCtrlF:
		return "C-f"
	case tea.KeyCtrlG:
		return "C-g"
	case tea.KeyCtrlH:
		return "C-h"
	case tea.KeyCtrlJ:
		return "C-j"
	case tea.KeyCtrlK:
		return "C-k"
	case tea.KeyCtrlL:
		return "C-l"
	case tea.KeyCtrlN:
		return "C-n"
	case tea.KeyCtrlO:
		return "C-o"
	case tea.KeyCtrlP:
		return "C-p"
	case tea.KeyCtrlR:
		return "C-r"
	case tea.KeyCtrlS:
		return "C-s"
	case tea.KeyCtrlT:
		return "C-t"
	case tea.KeyCtrlU:
		return "C-u"
	case tea.KeyCtrlV:
		return "C-v"
	case tea.KeyCtrlW:
		return "C-w"
	case tea.KeyCtrlX:
		return "C-x"
	case tea.KeyCtrlY:
		return "C-y"
	case tea.KeyCtrlZ:
		return "C-z"
	case tea.KeyRunes:
		return msg.String()
	default:
		return msg.String()
	}
}

func (m AppModel) HelpView() string {
	titleStyle := lipgloss.NewStyle().
		Foreground(m.theme.Accent).
		Bold(true).
		Padding(1, 0)

	sectionStyle := lipgloss.NewStyle().
		Foreground(m.theme.Accent).
		Bold(true).
		Padding(1, 0, 0, 0)

	keyStyle := lipgloss.NewStyle().
		Foreground(m.theme.Accent).
		Width(20)

	descStyle := lipgloss.NewStyle().
		Foreground(m.theme.Foreground)

	helpContent := titleStyle.Render("tmux-tui - Keybindings") + "\n\n"

	// Navigation section
	helpContent += sectionStyle.Render("Navigation") + "\n"
	helpContent += keyStyle.Render("  j/k, ↓/↑") + descStyle.Render("Navigate down/up in current list") + "\n"
	helpContent += keyStyle.Render("  ctrl+n/ctrl+p") + descStyle.Render("Navigate down/up (alternative)") + "\n"
	helpContent += keyStyle.Render("  1-9") + descStyle.Render("Jump to item by number in current list") + "\n"
	helpContent += keyStyle.Render("  tab") + descStyle.Render("Cycle focus forward (sessions→windows→panes)") + "\n"
	helpContent += keyStyle.Render("  shift+tab") + descStyle.Render("Cycle focus backward") + "\n"
	helpContent += keyStyle.Render("  enter") + descStyle.Render("Switch to selected session/window/pane") + "\n\n"

	// Entity management section
	helpContent += sectionStyle.Render("Entity Management") + "\n"
	helpContent += keyStyle.Render("  n") + descStyle.Render("New session (pick directory) / window (name prompt)") + "\n"
	helpContent += keyStyle.Render("  N") + descStyle.Render("Create new (auto-generated name)") + "\n"
	helpContent += keyStyle.Render("  r") + descStyle.Render("Rename selected item") + "\n"
	helpContent += keyStyle.Render("  d") + descStyle.Render("Delete selected item") + "\n"
	helpContent += keyStyle.Render("  R") + descStyle.Render("Refresh session (kill & recreate, triggers hooks)") + "\n"
	helpContent += keyStyle.Render("  s") + descStyle.Render("Enter swap mode (then s/space/enter to confirm)") + "\n\n"

	// Pane operations section
	helpContent += sectionStyle.Render("Pane Operations (when pane focused)") + "\n"
	helpContent += keyStyle.Render("  v") + descStyle.Render("Split pane vertically") + "\n"
	helpContent += keyStyle.Render("  h") + descStyle.Render("Split pane horizontally") + "\n\n"

	// Live input section
	helpContent += sectionStyle.Render("Live Input (windows/panes)") + "\n"
	helpContent += keyStyle.Render("  i") + descStyle.Render("Enter live input mode (send keys to selected pane)") + "\n"
	helpContent += keyStyle.Render("  ctrl+q") + descStyle.Render("Exit live input mode") + "\n\n"

	// Filtering section
	helpContent += sectionStyle.Render("Filtering & Display") + "\n"
	helpContent += keyStyle.Render("  /") + descStyle.Render("Enter filter mode") + "\n"
	helpContent += keyStyle.Render("  a") + descStyle.Render("Toggle show all items") + "\n\n"

	// Preview control section
	helpContent += sectionStyle.Render("Preview Control") + "\n"
	helpContent += keyStyle.Render("  ctrl+u, K") + descStyle.Render("Scroll preview up") + "\n"
	helpContent += keyStyle.Render("  ctrl+d, J") + descStyle.Render("Scroll preview down") + "\n"
	helpContent += keyStyle.Render("  +/=") + descStyle.Render("Next screen mode (normal→half→full)") + "\n"
	helpContent += keyStyle.Render("  _/-") + descStyle.Render("Previous screen mode (full→half→normal)") + "\n"
	helpContent += keyStyle.Render("  m") + descStyle.Render("Toggle maximize preview window") + "\n"
	helpContent += keyStyle.Render("  t") + descStyle.Render("Toggle transparent background") + "\n\n"

	// General section
	helpContent += sectionStyle.Render("General") + "\n"
	helpContent += keyStyle.Render("  ?") + descStyle.Render("Toggle this help menu") + "\n"
	helpContent += keyStyle.Render("  q, ctrl+c") + descStyle.Render("Quit application") + "\n"
	helpContent += keyStyle.Render("  esc") + descStyle.Render("Cancel current operation") + "\n\n"

	helpContent += descStyle.Foreground(m.theme.Secondary).Render("Press ? to close this help menu")

	// Calculate content height and ensure it fits within the terminal
	contentLines := strings.Count(helpContent, "\n")
	maxHeight := m.terminal.height - 6
	boxWidth := m.terminal.width*8/10 - 4

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Accent).
		Padding(1, 2).
		Width(boxWidth)

	if contentLines < maxHeight {
		boxStyle = boxStyle.Height(contentLines + 2)
	} else {
		boxStyle = boxStyle.Height(maxHeight)
	}

	helpBox := boxStyle.Render(helpContent)

	return lipgloss.Place(
		m.terminal.width,
		m.terminal.height,
		lipgloss.Center,
		lipgloss.Center,
		helpBox,
	)
}
