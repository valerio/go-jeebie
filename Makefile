generate:
	cd jeebie/disasm && go generate
	go fmt ./jeebie/disasm/

build: generate fmt
	go build -o bin/jeebie ./cmd/jeebie

test:
	go test -v ./...

test-blargg:
	@echo "Running Blargg test suite (comparing screen data hashes)..."
	go test -v ./test/blargg/...

test-blargg-golden:
	@echo "Generating reference screen data and snapshots for Blargg tests..."
	BLARGG_GENERATE_GOLDEN=true go test -v ./test/blargg/...

test-all: test test-blargg

fmt:
	go fmt ./...

# SDL2 development libraries installation
install-sdl2-ubuntu:
	sudo apt update && sudo apt install -y libsdl2-dev

install-sdl2-fedora:
	sudo dnf install -y SDL2-devel

install-sdl2-arch:
	sudo pacman -S sdl2

install-sdl2-macos:
	brew install sdl2

# Build variants
build-sdl2: generate fmt
	go build -tags sdl2 -o bin/jeebie ./cmd/jeebie

build-all: build build-sdl2

# Quick test targets
test-terminal:
	./bin/jeebie --backend=terminal --test-pattern

test-sdl2:
	./bin/jeebie --backend=sdl2 --test-pattern

test-headless:
	./bin/jeebie --headless --test-pattern

test-backends: test-headless test-terminal test-sdl2

# Development shortcuts
dev-terminal: build test-terminal

dev-sdl2: build-sdl2 test-sdl2

run:
	./bin/jeebie $(ROM)
