.PHONY: build test lint run-daemon

build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o pakai ./cmd/pakai

test:
	go test ./...

lint:
	go vet ./...

run-daemon:
	go run ./cmd/pakai daemon start --foreground
