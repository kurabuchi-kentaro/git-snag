.PHONY: build test lint fmt vet clean

build:
	go build ./...

test:
	go test ./...

lint:
	golangci-lint run

fmt:
	golangci-lint fmt

vet:
	go vet ./...

clean:
	rm -rf dist/ git-snag
