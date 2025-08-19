.PHONY: generate
generate:
	cd jeebie/disasm && go generate
	go fmt ./jeebie/disasm/

.PHONY: build
build: generate fmt
	go build -o bin/jeebie ./cmd/jeebie

# Build variants
.PHONY: build-sdl2
build-sdl2: generate fmt
	go build -tags sdl2 -o bin/jeebie-sdl2 ./cmd/jeebie

.PHONY: build-all
build-all: build build-sdl2

.PHONY: test
test:
	go test -short -v ./...

.PHONY: test-blargg
test-blargg:
	@echo "Running Blargg test suite (comparing screen data hashes)..."
	go test -v ./test/blargg/...

.PHONY: test-blargg-golden
test-blargg-golden:
	@echo "Generating reference screen data and snapshots for Blargg tests..."
	BLARGG_GENERATE_GOLDEN=true go test -v ./test/blargg/...

.PHONY: test-all
test-all: test test-blargg

.PHONY: fmt
fmt:
	go fmt ./...

# Quick test targets
.PHONY: test-headless
test-headless:
	./bin/jeebie --headless --test-pattern

.PHONY: test-sdl2
test-sdl2: build-sdl2
	./bin/jeebie-sdl2 --backend=sdl2 --test-pattern

# Extract ROM filename from command line goals
ROM_FILE = $(filter %.gb %.gbc,$(MAKECMDGOALS))

.PHONY: run-sdl2
run-sdl2: build-sdl2
	@if [ -z "$(ROM_FILE)" ]; then echo "Usage: make run-sdl2 <rom-file>"; exit 1; fi
	./bin/jeebie-sdl2 --backend=sdl2 $(ROM_FILE)

.PHONY: run-headless
run-headless:
	@if [ -z "$(ROM_FILE)" ]; then echo "Usage: make run-headless <rom-file> [FRAMES=60]"; exit 1; fi
	./bin/jeebie --headless --frames=$(or $(FRAMES),60) $(ROM_FILE)

.PHONY: run
run:
	@if [ -z "$(ROM_FILE)" ]; then echo "Usage: make run <rom-file>"; exit 1; fi
	./bin/jeebie $(ROM_FILE)

# Make ROM files valid targets that do nothing
%.gb %.gbc:
	@:
