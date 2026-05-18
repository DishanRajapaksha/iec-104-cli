.PHONY: fmt test build clean

APP_NAME := iec-104-cli

fmt:
	gofmt -w .

test:
	go test ./...

build:
	mkdir -p bin
	go build -o bin/$(APP_NAME) .

clean:
	rm -rf bin dist coverage.out
