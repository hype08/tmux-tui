package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type dirEntry struct {
	name string
}

// DirPickerModel is a centered modal for browsing and selecting directories.
type DirPickerModel struct {
	currentPath string
	entries     []dirEntry
	cursor      int
	filterInput textinput.Model
	showHidden  bool
	err         string
}

type dirEntriesMsg struct {
	path    string
	entries []dirEntry
	err     string
}

// NewDirPicker creates a new directory picker model.
func NewDirPicker(theme Theme) DirPickerModel {
	ti := textinput.New()
	ti.Focus()
	ti.Placeholder = "Filter..."
	ti.TextStyle = theme.NewStyle()
	return DirPickerModel{
		filterInput: ti,
	}
}

// initDirPickerCmd fetches the current session's working directory and reads its contents.
func initDirPickerCmd(sessionId int) tea.Cmd {
	return func() tea.Msg {
		target := fmt.Sprintf("$%d", sessionId)
		pathBytes, err := exec.Command("tmux", "display-message", "-t", target, "-p", "#{session_path}").Output()
		if err != nil {
			home, _ := os.UserHomeDir()
			return readDirEntries(home)
		}
		path := strings.TrimSpace(string(pathBytes))
		if path == "" {
			home, _ := os.UserHomeDir()
			path = home
		}
		return readDirEntries(path)
	}
}

func readDirCmd(path string) tea.Cmd {
	return func() tea.Msg {
		return readDirEntries(path)
	}
}

func readDirEntries(path string) dirEntriesMsg {
	osEntries, err := os.ReadDir(path)
	if err != nil {
		return dirEntriesMsg{path: path, err: err.Error()}
	}
	var entries []dirEntry
	for _, e := range osEntries {
		if e.IsDir() {
			entries = append(entries, dirEntry{name: e.Name()})
		}
	}
	return dirEntriesMsg{path: path, entries: entries}
}

func (dp DirPickerModel) filteredEntries() []string {
	filter := strings.ToLower(dp.filterInput.Value())
	result := []string{".", ".."}
	for _, e := range dp.entries {
		if !dp.showHidden && strings.HasPrefix(e.name, ".") {
			continue
		}
		if filter != "" && !strings.Contains(strings.ToLower(e.name), filter) {
			continue
		}
		result = append(result, e.name)
	}
	return result
}

func (dp *DirPickerModel) moveCursor(delta int) {
	entries := dp.filteredEntries()
	if len(entries) == 0 {
		dp.cursor = 0
		return
	}
	dp.cursor += delta
	if dp.cursor < 0 {
		dp.cursor = 0
	}
	if dp.cursor >= len(entries) {
		dp.cursor = len(entries) - 1
	}
}

func (dp *DirPickerModel) clampCursor() {
	entries := dp.filteredEntries()
	if len(entries) == 0 {
		dp.cursor = 0
		return
	}
	if dp.cursor >= len(entries) {
		dp.cursor = len(entries) - 1
	}
	if dp.cursor < 0 {
		dp.cursor = 0
	}
}

// handleEnter processes Enter in the picker.
// Returns a command and whether the picker should close.
func (dp *DirPickerModel) handleEnter(sessions []TmuxEntity) (tea.Cmd, bool) {
	entries := dp.filteredEntries()
	if len(entries) == 0 || dp.cursor >= len(entries) {
		return nil, false
	}
	selected := entries[dp.cursor]
	switch selected {
	case ".":
		return newSessionInDirCmd(dp.currentPath, sessions), true
	case "..":
		parent := filepath.Dir(dp.currentPath)
		dp.filterInput.SetValue("")
		dp.cursor = 0
		return readDirCmd(parent), false
	default:
		newPath := filepath.Join(dp.currentPath, selected)
		dp.filterInput.SetValue("")
		dp.cursor = 0
		return readDirCmd(newPath), false
	}
}

var sanitizeRegexp = regexp.MustCompile(`[^a-zA-Z0-9_-]`)

// sanitizeSessionName makes a directory basename safe for use as a tmux session name.
func sanitizeSessionName(name string) string {
	name = strings.TrimLeft(name, ".")
	name = sanitizeRegexp.ReplaceAllString(name, "-")
	name = strings.Trim(name, "-")
	if name == "" {
		name = "unnamed"
	}
	return name
}

// newSessionInDirCmd creates a detached tmux session in the given directory.
// If a session with the same basename already exists, appends a numeric suffix.
func newSessionInDirCmd(path string, sessions []TmuxEntity) tea.Cmd {
	return func() tea.Msg {
		basename := filepath.Base(path)
		name := sanitizeSessionName(basename)

		// If name collides, find a unique suffix
		if sessionExists(name, sessions) {
			for i := 2; i < 100; i++ {
				candidate := fmt.Sprintf("%s-%d", name, i)
				if !sessionExists(candidate, sessions) {
					name = candidate
					break
				}
			}
		}

		exec.Command("tmux", "new-session", "-ds", name, "-c", path).Run()

		panesBytes, _ := exec.Command("tmux", "list-panes", "-s", "-t", name, "-F", "#{pane_id}").Output()
		for _, paneId := range strings.Split(strings.TrimSpace(string(panesBytes)), "\n") {
			if paneId != "" {
				exec.Command("tmux", "send-keys", "-t", paneId, fmt.Sprintf("cd %q && clear", path), "Enter").Run()
			}
		}

		return clearInputTextMsg{}
	}
}

func sessionExists(name string, sessions []TmuxEntity) bool {
	for _, s := range sessions {
		if s.name == name {
			return true
		}
	}
	return false
}

// View renders the directory picker as a centered modal overlay.
func (dp DirPickerModel) View(theme Theme, termWidth, termHeight int) string {
	modalWidth := termWidth * 6 / 10
	if modalWidth < 40 {
		modalWidth = min(40, termWidth-4)
	}
	modalHeight := termHeight * 6 / 10
	if modalHeight < 10 {
		modalHeight = min(10, termHeight-4)
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(theme.Accent).
		Bold(true)
	normalStyle := theme.NewStyle()
	selectedStyle := theme.NewStyle().Foreground(theme.Accent)
	secondaryStyle := theme.NewStyle().Foreground(theme.Secondary)

	// Path with ~ shorthand
	pathDisplay := dp.currentPath
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(pathDisplay, home) {
		pathDisplay = "~" + pathDisplay[len(home):]
	}
	title := titleStyle.Render(pathDisplay)

	// Filter input
	dp.filterInput.Width = modalWidth - 8
	filterLine := dp.filterInput.View()

	// Hidden toggle hint
	var hiddenHint string
	if dp.showHidden {
		hiddenHint = secondaryStyle.Render("  [tab: hide hidden]")
	} else {
		hiddenHint = secondaryStyle.Render("  [tab: show hidden]")
	}

	// Error state
	if dp.err != "" {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
		content := title + "\n" + filterLine + hiddenHint + "\n\n" + errStyle.Render("Error: "+dp.err)

		boxStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Accent).
			Padding(1, 2).
			Width(modalWidth - 4).
			Height(modalHeight - 4)

		errPlaceOpts := []lipgloss.WhitespaceOption{}
		if !theme.Transparent {
			errPlaceOpts = append(errPlaceOpts, lipgloss.WithWhitespaceBackground(theme.Background))
		}
		return lipgloss.Place(termWidth, termHeight, lipgloss.Center, lipgloss.Center, boxStyle.Render(content), errPlaceOpts...)
	}

	// Entry list with scroll
	entries := dp.filteredEntries()
	listHeight := modalHeight - 8
	if listHeight < 3 {
		listHeight = 3
	}

	startIdx := max(0, dp.cursor-listHeight+1)
	endIdx := startIdx + listHeight
	if endIdx > len(entries) {
		endIdx = len(entries)
		startIdx = max(0, endIdx-listHeight)
	}

	var lines []string
	for i := startIdx; i < endIdx; i++ {
		entry := entries[i]
		prefix := "  "
		if i == dp.cursor {
			prefix = "● "
		}

		var display string
		switch entry {
		case ".":
			display = ".  (select this directory)"
		case "..":
			display = ".."
		default:
			display = entry + "/"
		}

		if i == dp.cursor {
			lines = append(lines, selectedStyle.Render(prefix+display))
		} else if entry == "." || entry == ".." {
			lines = append(lines, secondaryStyle.Render(prefix+display))
		} else {
			lines = append(lines, normalStyle.Render(prefix+display))
		}
	}

	listContent := strings.Join(lines, "\n")
	content := title + "\n" + filterLine + hiddenHint + "\n\n" + listContent

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Accent).
		Padding(1, 2).
		Width(modalWidth - 4)

	contentLines := strings.Count(content, "\n") + 1
	maxContentHeight := modalHeight - 4
	if contentLines < maxContentHeight {
		boxStyle = boxStyle.Height(contentLines + 2)
	} else {
		boxStyle = boxStyle.Height(maxContentHeight)
	}

	placeOpts := []lipgloss.WhitespaceOption{}
	if !theme.Transparent {
		placeOpts = append(placeOpts, lipgloss.WithWhitespaceBackground(theme.Background))
	}
	return lipgloss.Place(termWidth, termHeight, lipgloss.Center, lipgloss.Center, boxStyle.Render(content), placeOpts...)
}
