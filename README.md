# tmux-tui

Terminal User Interface for managing tmux sessions, windows, and panes. Inspired by lazygit.

## Features

- Interactive TUI with vertical sidebar layout
- Preview pane content in real-time
- Create, rename, delete, and swap sessions/windows/panes
- Split panes vertically and horizontally
- Filter and search functionality
- Help menu with `?` key
- Toggle maximize preview with `m` key
- Inherits terminal theme (Ghostty/tmux colors)

## Installation

### Download pre-built binary (recommended)

1. Go to the [Releases page](https://github.com/hype08/tmux-tui/releases)
2. Download the archive for your platform:
   - macOS Apple Silicon: `tmux-tui_*_Darwin_arm64.tar.gz`
   - macOS Intel: `tmux-tui_*_Darwin_x86_64.tar.gz`
   - Linux ARM: `tmux-tui_*_Linux_arm64.tar.gz`
   - Linux x64: `tmux-tui_*_Linux_x86_64.tar.gz`
   - Windows ARM: `tmux-tui_*_Windows_arm64.zip`
   - Windows x64: `tmux-tui_*_Windows_x86_64.zip`

3. Extract the archive:
   ```bash
   # For .tar.gz files
   tar -xzf tmux-tui_*.tar.gz

   # For .zip files (Windows)
   unzip tmux-tui_*.zip
   ```

4. Move the binary to a location in your PATH:
   ```bash
   mkdir -p ~/.local/bin
   mv tmux-tui ~/.local/bin/
   # Add ~/.local/bin to your PATH if not already there
   ```

5. Verify installation:
   ```bash
   tmux-tui --version
   ```

### Build from source

If you prefer to build from source:

```bash
git clone https://github.com/hype08/tmux-tui.git
cd tmux-tui
make
```

The binary will be at `build/tmux-tui`.

## Quick Start Tutorial

### Step 1: Run tmux-tui

First, make sure you're inside a tmux session:

```bash
# If not in tmux, start a new session
tmux new -s demo

# Run tmux-tui
tmux-tui
```

### Step 2: Navigate the interface

- Use `j`/`k` or arrow keys to move up and down
- Press `tab` to move between columns (Sessions → Windows → Panes)
- Press `?` to see all keybindings

### Step 3: Create a new window

1. Press `tab` to focus the Windows column
2. Press `n` to create a new window
3. Type a name (or press `enter` for default)

### Step 4: Bind to a tmux key (recommended)

Exit tmux-tui by pressing `q`, then add this to your `~/.tmux.conf`:

```tmux
bind-key w display-popup -E -w 80% -h 80% "tmux-tui"
```

Reload your tmux configuration:

```bash
tmux source-file ~/.tmux.conf
```

Now press `prefix w` to open tmux-tui in a popup window.

### Step 5: Explore features

- Press `m` to maximize the preview pane
- Press `/` to filter items by name
- Press `v` or `h` to split panes (when in Panes column)

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

## How-To

### How to maximize preview for detailed inspection

Press `m` to toggle preview window to full screen. This is useful for:
- Reading long output or logs
- Inspecting detailed command output
- Viewing large file contents

Press `m` again to return to split view.

### How to filter sessions/windows/panes

1. Press `/` to enter filter mode
2. Type your search term
3. Press `enter` to apply filter
4. Press `a` to toggle showing all items vs filtered items
5. Press `/` and clear the text, then `enter` to remove filter

## Development

### Building

```bash
make
```

### Cleaning

```bash
make clean
```

### Project Structure

```
.
├── cmd/
│   └── root.go          # CLI entry point
├── tui/
│   ├── application.go   # Main TUI logic and state management
│   ├── frame.go         # UI rendering and grid layout
│   ├── sessions.go      # Session management operations
│   ├── windows.go       # Window management operations
│   ├── panes.go         # Pane management operations
│   ├── theme.go         # Minimal theme system
│   └── version.go       # Version information
├── main.go              # Program entry
├── Makefile             # Build system
└── go.mod               # Go dependencies
```

## Credits

Forked from [tmux-tui](https://github.com/acristoffers/tmux-tui) by acristoffers.

Changes in tmux-tui:
- Simplified theme system (inherits from terminal)
- Vertical sidebar layout instead of horizontal
- Help menu with `?` key
- Toggle maximize preview with `m` key
- Fixed cascading preview bug
- Removed Nix support and documentation generation

## License

See LICENSE file.
