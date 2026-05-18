.PHONY: fmt test build examples docker-build deb clean

APP_NAME := iec-104-cli
DOCKER_IMAGE ?= iec-104-cli:latest
VERSION ?= 0.1.0
ARCH ?= amd64

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

deb: build
	rm -rf dist/deb
	mkdir -p dist/deb/DEBIAN dist/deb/usr/bin dist/deb/usr/share/doc/$(APP_NAME) dist/deb/usr/share/$(APP_NAME)/examples
	cp bin/$(APP_NAME) dist/deb/usr/bin/$(APP_NAME)
	cp README.md dist/deb/usr/share/doc/$(APP_NAME)/README.md
	cp -R examples/. dist/deb/usr/share/$(APP_NAME)/examples/
	printf 'Package: $(APP_NAME)\nVersion: $(VERSION)\nSection: net\nPriority: optional\nArchitecture: $(ARCH)\nMaintainer: Dishan Rajapaksha <noreply@example.com>\nDescription: Script-friendly IEC 60870-5-104 command-line client\n' > dist/deb/DEBIAN/control
	dpkg-deb --build dist/deb dist/$(APP_NAME)_$(VERSION)_$(ARCH).deb

clean:
	rm -rf bin dist coverage.out
