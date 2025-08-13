generate: fmt
	cd jeebie/disasm && go generate

build: generate fmt
	go build -o bin/jeebie ./cmd/jeebie

run:
	go run ./cmd/jeebie test-roms/01-special.gb

test:
	go test -v ./...

fmt:
	go fmt ./...