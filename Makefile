VERSION ?= dev

.PHONY: build test test-all test-update test-tui lint clean

build:
	go build -ldflags="-s -w -X main.version=$(VERSION)" -o k6delta ./cmd/k6delta

test:
	go test ./... -v

test-all:
	go test -tags integration -count=1 ./... -v

test-update:
	UPDATE_GOLDEN=1 go test ./internal/tui/... -v

test-tui:
	go test ./internal/tui/... -v

lint:
	go vet ./...

clean:
	rm -f k6delta
