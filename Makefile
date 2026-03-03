.PHONY: build test lint clean

build:
	go build -o k6delta ./cmd/k6delta

test:
	go test ./... -v

lint:
	go vet ./...

clean:
	rm -f k6delta
