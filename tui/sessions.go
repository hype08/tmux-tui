package tui

import (
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// getSelfSessionId returns the tmux session ID (e.g. "$0") that the TUI is running in.
func getSelfSessionId() string {
	out, err := exec.Command("tmux", "display-message", "-p", "#{session_id}").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func goToSessionCmd(m AppModel) tea.Cmd {
	return func() tea.Msg {
		c := exec.Command("tmux", "switch-client", "-t", fmt.Sprintf("$%d", m.sessions.currentId))
		c.Run()
		return tea.QuitMsg{}
	}
}

func renameSessionCmd(m AppModel) tea.Cmd {
	return func() tea.Msg {
		_ = goToSessionCmd(m)()
		session := fmt.Sprintf("$%d", m.sessions.currentId)
		c := exec.Command("tmux", "rename-session", "-t", session, m.textInput.Value())
		c.Run()
		return clearInputTextMsg{}
	}
}

func newSessionCmd(m AppModel) tea.Cmd {
	return func() tea.Msg {
		if len(m.textInput.Value()) > 0 {
			c := exec.Command("tmux",
				"new-session", "-ds", m.textInput.Value(), ";",
				"switch-client", "-t", m.textInput.Value())
			c.Run()
		} else {
			c := exec.Command("tmux", "new-session", "-d")
			c.Run()
		}
		return clearInputTextMsg{}
	}
}

func deleteSessionCmd(m AppModel) tea.Cmd {
	return func() tea.Msg {
		sessionName := m.sessions.ItemWithId(m.sessions.currentId).name
		sessionTarget := fmt.Sprintf("$%d", m.sessions.currentId)

		// If deleting our own session, switch to the last-used session first.
		// If no other session exists, the kill simply detaches us.
		if getSelfSessionId() == sessionTarget {
			exec.Command("tmux", "switch-client", "-l").Run()
		}

		exec.Command("tmux", "kill-session", "-t", sessionName).Run()
		return tickMsg{}
	}
}

func refreshSessionCmd(m AppModel) tea.Cmd {
	return func() tea.Msg {
		session := m.sessions.ItemWithId(m.sessions.currentId)
		sessionName := session.name
		sessionTarget := fmt.Sprintf("$%d", m.sessions.currentId)

		// Grab the session's start directory so we can recreate in the same place
		pathBytes, _ := exec.Command("tmux", "display-message", "-t", sessionTarget, "-p", "#{session_path}").Output()
		sessionPath := strings.TrimSpace(string(pathBytes))

		isSelf := getSelfSessionId() == sessionTarget

		// Create replacement session with a temp name to avoid collision.
		// The after-new-session hook fires here, rebuilding default windows.
		tempName := sessionName + "-refreshing"
		args := []string{"new-session", "-d", "-s", tempName}
		if sessionPath != "" {
			args = append(args, "-c", sessionPath)
		}
		exec.Command("tmux", args...).Run()

		if isSelf {
			exec.Command("tmux", "switch-client", "-t", tempName).Run()
		}

		exec.Command("tmux", "kill-session", "-t", sessionName).Run()
		exec.Command("tmux", "rename-session", "-t", tempName, sessionName).Run()

		if isSelf {
			exec.Command("tmux", "switch-client", "-t", sessionName).Run()
		}

		return tickMsg{}
	}
}
