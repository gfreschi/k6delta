VERSION ?= dev

.PHONY: build test lint clean

build:
	go build -ldflags="-s -w -X main.version=$(VERSION)" -o k6delta ./cmd/k6delta

test:
	go test ./... -v

lint:
	go vet ./...

clean:
	rm -f k6delta
