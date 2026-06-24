BINARY := cc-orchestra

.PHONY: build test lint install run tidy
build:
	go build -o bin/$(BINARY) ./cmd/cc-orchestra
test:
	go test ./...
lint:
	golangci-lint run
install:
	go install ./cmd/cc-orchestra
run: build
	./bin/$(BINARY)
tidy:
	go mod tidy
