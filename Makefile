bootstrap:
	brew install sdl2
	go mod download

build:
	go build -o bin/jeebie main.go

run:
	go run main.go test-roms/01-special.gb

test:
	go test -v ./...