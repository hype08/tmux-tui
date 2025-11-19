package tui

import (
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func goToPaneCmd(m AppModel) tea.Cmd {
	return func() tea.Msg {
		pane := m.panes.ItemWithId(m.panes.currentId)
		window := m.windows.ItemWithId(pane.parent)
		c := exec.Command("tmux",
			"switch-client", "-t", fmt.Sprintf("$%d", window.parent), ";",
			"select-window", "-t", fmt.Sprintf("@%d", window.id), ";",
			"select-pane", "-t", fmt.Sprintf("%%%d", pane.id))
		c.Run()
		return tea.QuitMsg{}
	}
}

func deletePaneCmd(m AppModel) tea.Cmd {
	return func() tea.Msg {
		c := exec.Command("tmux", "kill-pane", "-t", fmt.Sprintf("%%%d", m.panes.currentId))
		c.Run()
		return tickMsg{}
	}
}

func swapPanesCmd(m AppModel, src int) tea.Cmd {
	return func() tea.Msg {
		src := fmt.Sprintf("%%%d", src)
		dst := fmt.Sprintf("%%%d", m.panes.currentId)
		c := exec.Command("tmux", "swap-pane", "-s", src, "-t", dst)
		c.Run()
		return tickMsg{}
	}
}

func splitPane(m AppModel, horizontal bool) tea.Cmd {
	return func() tea.Msg {
		target := fmt.Sprintf("%%%d", m.panes.currentId)
		orientation := "-v"
		if horizontal {
			orientation = "-h"
		}
		c := exec.Command("tmux", "split-pane", "-d", "-t", target, orientation)
		c.Run()
		return tickMsg{}
	}
}
func sendKeyToPaneCmd(m AppModel, key tea.KeyMsg) tea.Cmd {
	return func() tea.Msg {
		target := fmt.Sprintf("%%%d", m.liveInputTarget)

		checkCmd := exec.Command("tmux", "list-panes", "-a", "-F", "#{pane_id}")
		output, err := checkCmd.Output()
		if err != nil {
			return paneClosedMsg{}
		}

		paneExists := false
		for _, line := range strings.Split(string(output), "\n") {
			if strings.TrimSpace(line) == target {
				paneExists = true
				break
			}
		}

		if !paneExists {
			return paneClosedMsg{}
		}

		keyString := mapKeyToTmuxSendKeys(key)
		c := exec.Command("tmux", "send-keys", "-t", target, keyString)
		c.Run()
		return nil
	}
}
