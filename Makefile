bootstrap:
	brew install sdl2
	go mod download

build:
	go build -o bin/jeebie ./cmd/jeebie

run:
	go run ./cmd/jeebie test-roms/01-special.gb

test:
	go test -v ./...