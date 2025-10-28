package tui

import (
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/list"
	"github.com/muesli/reflow/truncate"
)

type Frame struct {
	title      string
	contents   string
	width      int
	height     int
	focused    bool
	scrollY    int
	scrollable bool
}

func NewFrame(m AppModel) Frame {
	return Frame{
		title:      "",
		contents:   "",
		width:      m.terminal.width,
		height:     m.terminal.height,
		focused:    true,
		scrollY:    0,
		scrollable: false,
	}
}

// ScrollUp scrolls the frame content up by the given number of lines
func (f *Frame) ScrollUp(lines int) {
	f.scrollY -= lines
	if f.scrollY < 0 {
		f.scrollY = 0
	}
}

// ScrollDown scrolls the frame content down by the given number of lines
func (f *Frame) ScrollDown(lines int) {
	contentLines := strings.Count(f.contents, "\n") + 1
	maxScroll := contentLines - f.height + 3 // Account for borders and padding
	if maxScroll < 0 {
		maxScroll = 0
	}

	f.scrollY += lines
	if f.scrollY > maxScroll {
		f.scrollY = maxScroll
	}
}

func (frame Frame) View(theme Theme) string {
	roundedBorder := lipgloss.RoundedBorder()

	// Width accounts for left and right borders
	width := frame.width - 2

	// Use base style without explicit colors to inherit terminal theme
	style := lipgloss.NewStyle()

	// Labeled top border (custom header line)
	truncated := truncate.String(fmt.Sprintf(" %s ", frame.title), uint(width-2))
	title := lipgloss.PlaceHorizontal(width-1, lipgloss.Left, truncated, lipgloss.WithWhitespaceChars(roundedBorder.Top))
	borderTop := fmt.Sprintf("%s%s%s%s", roundedBorder.TopLeft, roundedBorder.Top, title, roundedBorder.TopRight)

	// Calculate remaining height after header line
	// Total: frame.height
	// Used by header: 1 line
	// Available for pane: frame.height - 1
	paneHeight := frame.height - 1

	// Pane structure: no top border, has bottom border (1 line)
	// Content height: paneHeight - 1 (for border)
	// Only add bottom margin if there's room (height > 5)
	contentMaxHeight := paneHeight - 1
	contentStyle := style.Align(lipgloss.Left, lipgloss.Top).
		MaxWidth(width - 4).
		MaxHeight(contentMaxHeight)

	// Add bottom margin only for taller frames to provide visual spacing
	if frame.height > 5 {
		contentMaxHeight = paneHeight - 2
		contentStyle = contentStyle.MaxHeight(contentMaxHeight).MarginBottom(1)
	}

	// Apply scrolling if enabled
	displayContents := frame.contents
	if frame.scrollable && frame.scrollY > 0 {
		lines := strings.Split(frame.contents, "\n")
		if frame.scrollY < len(lines) {
			displayContents = strings.Join(lines[frame.scrollY:], "\n")
		} else {
			displayContents = ""
		}
	}

	contents := contentStyle.Render(displayContents)

	header := style.SetString(borderTop)
	// Set Height to account for borders being OUTSIDE the height value
	// paneHeight already accounts for header (frame.height - 1)
	// Border adds 1 line, so inner height = paneHeight - 1
	pane := style.Border(lipgloss.RoundedBorder(), false, true, true, true).
		PaddingLeft(1).
		PaddingRight(1).
		Width(width).
		Height(paneHeight - 1).
		SetString(contents)

	if frame.focused {
		header = header.BorderForeground(theme.Accent)
		pane = pane.BorderForeground(theme.Accent)
	}

	// Join header and pane
	joined := lipgloss.JoinVertical(lipgloss.Top, header.String(), pane.String())

	// Container enforces width but allows natural height up to max
	// Don't set Height() as it can truncate the bottom border
	container := lipgloss.NewStyle().
		Width(frame.width).
		MaxWidth(frame.width).
		MaxHeight(frame.height)

	return container.Render(joined)
}

func (m AppModel) DrawGrid(preview, sessions, windows, frames, status Frame) string {
	w := m.terminal.width
	h := m.terminal.height
	contentHeight := h - 3 // Reserve 3 lines for status (top border, content, bottom border)

	// Status bar spans full width, needs 3 lines for bordered box
	status.width = w
	status.height = 3

	// If preview is maximized, show only preview and status bar
	if m.maximizePreview {
		preview.width = w
		preview.height = contentHeight

		previewRendered := preview.View(m.theme)
		statusRendered := status.View(m.theme)

		return lipgloss.JoinVertical(lipgloss.Top, previewRendered, statusRendered)
	}

	// Calculate layout based on screen mode
	var leftWidth, rightWidth int
	switch m.screenMode {
	case ScreenNormal:
		// Normal layout: Left sidebar: 40% width, right preview: 60% width
		leftWidth = w * 4 / 10
		rightWidth = w - leftWidth
	case ScreenHalf:
		// Half screen: Left sidebar: 30% width, right preview: 70% width
		leftWidth = w * 3 / 10
		rightWidth = w - leftWidth
	case ScreenFull:
		// Full screen: show only preview
		preview.width = w
		preview.height = contentHeight

		previewRendered := preview.View(m.theme)
		statusRendered := status.View(m.theme)

		return lipgloss.JoinVertical(lipgloss.Top, previewRendered, statusRendered)
	}

	// Each frame in left sidebar gets 1/3 of height minus status bar
	frameHeight := contentHeight / 3

	// Set dimensions for left sidebar frames (stacked vertically)
	sessions.width = leftWidth
	sessions.height = frameHeight
	windows.width = leftWidth
	windows.height = frameHeight
	frames.width = leftWidth
	frames.height = contentHeight - (2 * frameHeight) // Remaining height

	// Preview takes full right side
	preview.width = rightWidth
	preview.height = contentHeight

	// Render all frames
	previewRendered := preview.View(m.theme)
	sessionsRendered := sessions.View(m.theme)
	windowsRendered := windows.View(m.theme)
	framesRendered := frames.View(m.theme)
	statusRendered := status.View(m.theme)

	// Stack left sidebar vertically: sessions, windows, panes
	leftSidebar := lipgloss.JoinVertical(lipgloss.Top, sessionsRendered, windowsRendered, framesRendered)

	// Join left sidebar and preview horizontally
	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, leftSidebar, previewRendered)

	// Add status bar at bottom
	return lipgloss.JoinVertical(lipgloss.Top, mainContent, statusRendered)
}

type ListFrame struct {
	frame      Frame
	items      []TmuxEntity
	currentId  int
	markedIds  []int
	parentId   int
	filterText string
}

func (listFrame *ListFrame) Update() {
	visibleItems := listFrame.visibleItems()
	for _, item := range visibleItems {
		if item.id == listFrame.currentId {
			return
		}
	}
	if len(visibleItems) > 0 {
		listFrame.currentId = visibleItems[0].id
	} else {
		listFrame.currentId = -1
	}
}

func (listFrame *ListFrame) RenderContents(theme Theme) Frame {
	enumeratorStyle := theme.NewStyle().Foreground(theme.Accent)
	itemStyle := theme.NewStyle()

	currentIndex := -1

	l := list.New().EnumeratorStyle(enumeratorStyle).ItemStyle(itemStyle)
	for i, item := range listFrame.visibleItems() {
		if slices.Contains(listFrame.markedIds, item.id) {
			l.Item(itemStyle.Foreground(theme.Secondary).Render(fmt.Sprintf("[%d]: %s", item.id, item.name)))
		} else {
			l.Item(itemStyle.Render(fmt.Sprintf("[%d]: %s", item.id, item.name)))
		}
		if item.id == listFrame.currentId {
			currentIndex = i
		}
	}

	if currentIndex > -1 {
		enumerator := func(l list.Items, i int) string {
			if i == currentIndex {
				return "→ "
			}
			return ""
		}
		l = l.Enumerator(enumerator)
	}

	listFrame.frame.contents = l.String()
	return listFrame.frame
}

func (listFrame *ListFrame) SelectNext() {
	items := listFrame.visibleItems()
	if listFrame.currentId == -1 && len(items) > 0 {
		listFrame.currentId = 0
		return
	}
	for i, item := range items {
		if item.id == listFrame.currentId {
			if i+1 < len(items) {
				listFrame.currentId = items[i+1].id
			}
			return
		}
	}
	listFrame.currentId = -1
}

func (listFrame *ListFrame) SelectPrevious() {
	items := listFrame.visibleItems()
	if listFrame.currentId == -1 && len(items) > 0 {
		listFrame.currentId = items[len(items)-1].id
		return
	}
	for i, item := range items {
		if item.id == listFrame.currentId {
			if i-1 >= 0 {
				listFrame.currentId = items[i-1].id
			}
			return
		}
	}
	listFrame.currentId = -1
}

func (listFrame *ListFrame) MarkSelection() {
	listFrame.markedIds = append(listFrame.markedIds, listFrame.currentId)
}

func (listFrame *ListFrame) UnmarkSelection() {
	index := slices.Index(listFrame.markedIds, listFrame.currentId)
	if index != -1 {
		listFrame.markedIds = slices.Delete(listFrame.markedIds, index, 1)
	}
}

func (listFrame *ListFrame) IsMarked(id int) bool {
	return slices.Contains(listFrame.markedIds, id)
}

func (listFrame *ListFrame) ClearMarks() {
	listFrame.markedIds = nil
}

func (listFrame ListFrame) ItemWithId(id int) *TmuxEntity {
	for _, item := range listFrame.items {
		if item.id == id {
			return &item
		}
	}
	return nil
}

func (listFrame *ListFrame) visibleItems() []TmuxEntity {
	var items []TmuxEntity
	filter := strings.ToLower(listFrame.filterText)
	for _, item := range listFrame.items {
		matchesParent := listFrame.parentId == -1 || item.parent == listFrame.parentId
		matchesFilter := len(filter) == 0 || strings.Contains(strings.ToLower(item.name), filter)
		if matchesParent && matchesFilter {
			items = append(items, item)
		}
	}
	return items
}
