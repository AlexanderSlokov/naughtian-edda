.PHONY: build install test clean

# Build edda into bin/.
build:
	go build -o bin/edda ./cmd/edda

# Install edda into $(go env GOPATH)/bin and TLC into ~/.cache/edda.
install:
	go install ./cmd/edda
	scripts/fetch-tla2tools.sh

test:
	go test ./...

clean:
	rm -rf bin
