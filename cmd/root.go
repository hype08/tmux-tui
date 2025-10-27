package cmd

import (
	"fmt"
	"os"

	"github.com/henryzhang/tmux-tui/tui"
	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "tmux-tui",
	Short: "Terminal User Interface for managing tmux sessions, windows, and panes",
	Long:  "Interactive TUI for creating, renaming, moving, and deleting tmux sessions, windows, and panes. Inspired by lazygit.",
	Run: func(cmd *cobra.Command, args []string) {
		version, err := cmd.Flags().GetBool("version")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not parse options: %s\n", err)
			os.Exit(1)
		}

		if version {
			fmt.Printf("Version %s", tui.Version)
			return
		}

		theme := tui.DefaultTheme()
		p := tui.NewApplication(theme)
		m, err := p.Run()
		if err != nil {
			os.Stderr.WriteString(fmt.Sprintf("There's been an error: %v\n", err))
			os.Exit(1)
		}

		switch m := m.(type) {
		case tui.AppModel:
			if len(m.Error) != 0 {
				os.Stderr.WriteString(m.Error + "\n")
				os.Exit(1)
			}
		}
	},
}

func Execute() {
	err := RootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	RootCmd.Flags().BoolP("version", "v", false, "Prints the version.")
}
