package tui

import (
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func goToWindowCmd(m AppModel) tea.Cmd {
	return func() tea.Msg {
		window := m.windows.ItemWithId(m.windows.currentId)
		c := exec.Command("tmux",
			"switch-client", "-t", fmt.Sprintf("$%d", window.parent), ";",
			"select-window", "-t", fmt.Sprintf("@%d", window.id))
		c.Run()
		return tea.QuitMsg{}
	}
}

func renameWindowCmd(m AppModel) tea.Cmd {
	return func() tea.Msg {
		c := exec.Command("tmux", "rename-window", "-t", fmt.Sprintf("@%d", m.windows.currentId), m.textInput.Value())
		c.Run()
		return clearInputTextMsg{}
	}
}

func newWindowCmd(m AppModel) tea.Cmd {
	return func() tea.Msg {
		sessionTarget := fmt.Sprintf("$%d:", m.sessions.currentId)

		// Inherit the session's start directory so new windows open there too
		pathBytes, _ := exec.Command("tmux", "display-message", "-t",
			fmt.Sprintf("$%d", m.sessions.currentId), "-p", "#{session_path}").Output()
		sessionPath := strings.TrimSpace(string(pathBytes))

		args := []string{"new-window", "-t", sessionTarget}
		if len(m.textInput.Value()) > 0 {
			args = append(args, "-n", m.textInput.Value())
		}
		if sessionPath != "" {
			args = append(args, "-c", sessionPath)
		}
		exec.Command("tmux", args...).Run()
		return clearInputTextMsg{}
	}
}

func deleteWindowCmd(m AppModel) tea.Cmd {
	return func() tea.Msg {
		c := exec.Command("tmux", "kill-window", "-t", fmt.Sprintf("@%d", m.windows.currentId))
		c.Run()
		return tickMsg{}
	}
}

func swapWindowsCmd(m AppModel, src int) tea.Cmd {
	return func() tea.Msg {
		src := fmt.Sprintf("@%d", src)
		dst := fmt.Sprintf("@%d", m.windows.currentId)
		c := exec.Command("tmux", "swap-window", "-s", src, "-t", dst)
		c.Run()
		return tickMsg{}
	}
}
