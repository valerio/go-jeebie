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

.PHONY: test-integration
test-integration:
	@echo "Running integration tests (comparing screen data hashes)..."
	go test -v ./test/blargg/...

.PHONY: test-integration-golden
test-integration-golden:
	@echo "Generating reference screen data and snapshots for integration tests..."
	BLARGG_GENERATE_GOLDEN=true go test -v ./test/blargg/...

.PHONY: test-all
test-all: test-roms-download test test-integration

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

TEST_ROMS_VERSION := v7.0

.PHONY: test-roms-download
test-roms-download:
	@echo "Downloading game-boy-test-roms-$(TEST_ROMS_VERSION)..."
	@mkdir -p test-roms
	@if [ ! -f "test-roms/.version" ] || [ "$$(cat test-roms/.version 2>/dev/null)" != "$(TEST_ROMS_VERSION)" ]; then \
		echo "Fetching test ROMs version $(TEST_ROMS_VERSION)..."; \
		wget -q https://github.com/c-sp/gameboy-test-roms/releases/download/$(TEST_ROMS_VERSION)/game-boy-test-roms-$(TEST_ROMS_VERSION).zip -O test-roms/test-roms.zip; \
		rm -rf test-roms/game-boy-test-roms; \
		mkdir -p test-roms/game-boy-test-roms; \
		cd test-roms/game-boy-test-roms && unzip -q -o ../test-roms.zip && cd ../.. && rm -f test-roms/test-roms.zip && echo "$(TEST_ROMS_VERSION)" > test-roms/.version; \
	else \
		echo "Test ROMs $(TEST_ROMS_VERSION) already present, skipping download"; \
	fi
	@echo "Test ROMs ready at test-roms/game-boy-test-roms"

.PHONY: test-roms-clean
test-roms-clean:
	rm -rf test-roms/game-boy-test-roms test-roms/.version

# Allow ROM files as make arguments (e.g., make run-sdl2 tetris.gb)
%.gb %.gbc:
	@:
