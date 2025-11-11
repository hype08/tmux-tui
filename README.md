# tmux-tui

Terminal User Interface for managing tmux sessions, windows, and panes.

## Getting Started

### Download Binary

1. Go to the [Releases page](https://github.com/hype08/tmux-tui/releases)
2. Download the archive for your platform
3. Extract and move to your PATH:

```bash
tar -xzf tmux-tui_*.tar.gz
mkdir -p ~/.local/bin
mv tmux-tui ~/.local/bin/
```

### Usage

Run inside a tmux session:

```bash
tmux-tui
```

Or bind to a tmux key (add to `~/.tmux.conf`):

```tmux
bind-key w display-popup -E -w 80% -h 80% "tmux-tui"
```

## Keybindings

Press `?` in the application to see all keybindings, or see below:

### Navigation
- `j/k` or `↓/↑` - Navigate down/up in current list
- `ctrl+n/ctrl+p` - Navigate down/up (alternative)
- `1` - Focus sessions column
- `2` - Focus windows column
- `3` - Focus panes column
- `tab` - Cycle focus forward (1→2→3→1)
- `shift+tab` - Cycle focus backward (3→2→1→3)
- `enter` - Switch to selected session/window/pane

### Entity Management
- `n` - Create new (prompts for name)
- `N` - Create new (auto-generated name)
- `r` - Rename selected item
- `d` - Delete selected item
- `s` - Enter swap mode (then `s`/`space`/`enter` to confirm)

### Pane Operations
- `v` - Split pane vertically (when pane focused)
- `h` - Split pane horizontally (when pane focused)

### Filtering & Display
- `/` - Enter filter mode
- `a` - Toggle show all items
- `m` - Toggle maximize preview window

### General
- `?` - Toggle help menu
- `q` or `ctrl+c` - Quit application
- `esc` - Cancel current operation

## Development

Build from source:

```bash
git clone https://github.com/hype08/tmux-tui.git
cd tmux-tui
make
```

## License

See LICENSE file.
