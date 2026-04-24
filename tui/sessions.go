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

		// Clean up stale temp session from a previous failed refresh
		tempName := sessionName + "-refreshing"
		exec.Command("tmux", "kill-session", "-t", "=" + tempName).Run()

		// Create replacement session with a temp name to avoid collision.
		// Use -P -F to capture the new session's ID for unambiguous targeting.
		// The after-new-session hook fires here, rebuilding default windows.
		args := []string{"new-session", "-d", "-s", tempName, "-P", "-F", "#{session_id}"}
		if sessionPath != "" {
			args = append(args, "-c", sessionPath)
		}
		newIdBytes, err := exec.Command("tmux", args...).Output()
		if err != nil {
			return tickMsg{}
		}
		newSessionId := strings.TrimSpace(string(newIdBytes))

		if isSelf {
			exec.Command("tmux", "switch-client", "-t", newSessionId).Run()
		}

		// Kill original session by ID to avoid name prefix-matching ambiguity
		exec.Command("tmux", "kill-session", "-t", sessionTarget).Run()
		exec.Command("tmux", "rename-session", "-t", newSessionId, sessionName).Run()

		if isSelf {
			exec.Command("tmux", "switch-client", "-t", newSessionId).Run()
		}

		return tickMsg{}
	}
}
