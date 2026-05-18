.PHONY: fmt test build examples docker-build clean

APP_NAME := iec-104-cli
DOCKER_IMAGE ?= iec-104-cli:latest

fmt:
	gofmt -w .

test:
	go test ./...

build:
	mkdir -p bin
	go build -o bin/$(APP_NAME) .

examples:
	go run . generate-configs --dir examples

docker-build:
	docker build -t $(DOCKER_IMAGE) .

clean:
	rm -rf bin dist coverage.out
