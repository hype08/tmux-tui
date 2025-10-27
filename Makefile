all: build/tmux-tui

build:
	@mkdir -p build

build/tmux-tui: build $(shell find . -type f -name "*.go")
	go build -o build/tmux-tui -ldflags="-s -w" .

clean:
	@rm -rf build

.PHONY: all build clean
