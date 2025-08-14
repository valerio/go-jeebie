generate: fmt
	cd jeebie/disasm && go generate

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
