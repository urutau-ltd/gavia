SHELL := /bin/sh

BINARY ?= gavia
BUILD_DIR ?= build
OUTPUT ?= $(BUILD_DIR)/$(BINARY)

INSTALL_NAME ?= gv
INSTALL_ROOT_DIR ?= /bin

IMAGE ?= ghcr.io/urutau-ltd/gavia:latest
IMAGE_PLATFORM ?= linux/amd64

GO ?= go

LDFLAGS ?= -s -w -buildid=

GIT_COMMIT ?= $(shell git rev-parse --short=12 HEAD 2>/dev/null || echo unknown)
GIT_TAG ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo dev)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
UPSTREAM_REPO ?= urutau-ltd/gavia
UPSTREAM_VENDOR ?= Urutau Limited
BUILD_LDFLAGS ?= \
	-X codeberg.org/urutau-ltd/gavia/cmd.BuildVersion=$(GIT_TAG) \
	-X codeberg.org/urutau-ltd/gavia/cmd.BuildTag=$(GIT_TAG) \
	-X codeberg.org/urutau-ltd/gavia/cmd.BuildCommit=$(GIT_COMMIT) \
	-X codeberg.org/urutau-ltd/gavia/cmd.BuildDate=$(BUILD_DATE) \
	-X codeberg.org/urutau-ltd/gavia/cmd.UpstreamRepo=$(UPSTREAM_REPO) \
	-X codeberg.org/urutau-ltd/gavia/cmd.UpstreamVendor=$(UPSTREAM_VENDOR)
FUZZTIME ?= 5s
GO_SOURCES := $(shell find . -type f -name '*.go' -not -path './vendor/*')

.PHONY:

fmt:
	@gofmt -w $(GO_SOURCES)

fmt-check:
	@UNFORMATTED="$$(gofmt -l $(GO_SOURCES))"; \
	if [ -n "$$UNFORMATTED" ]; then \
		echo "Files not formatted with gofmt:"; \
		echo "$$UNFORMATTED"; \
		exit 1; \
	fi

vet:
	$(GO) vet ./...

test:
	CGO_ENABLED=0 go test -v ./...

test-race:
	CGO_ENABLED=0 go test -race ./...

ci: fmt-check vet test test-race

build:
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags "$(LDFLAGS) $(BUILD_FLAGS)" -o $(OUTPUT) main.go

install:
	install -d "$(INSTALL_ROOT_DIR)";
	install -m 0755 "$(OUTPUT)" "$(INSTALL_ROOT_DIR)/$(INSTALL_NAME)";

run:
	CGO_ENABLED=0 $(GO) run main.go

clean:
	rm -rf $(BUILD_DIR)

image:
	podman build --platform $(IMAGE_PLATFORM) \
		--build-arg TARGETOS=linux \
		--build-arg TARGETARCH=$$(echo "$(IMAGE_PLATFORM)" | awk -F/ '{if ($$2 == "") print $$1; else print $$2}') \
		--build-arg BUILD_VERSION=$(GIT_TAG) \
		--build-arg BUILD_TAG=$(GIT_TAG) \
		--build-arg BUILD_COMMIT=$(GIT_COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		--build-arg UPSTREAM_REPO=$(UPSTREAM_REPO) \
		--build-arg UPSTREAM_VENDOR=$(UPSTREAM_VENDOR) \
		-t $(IMAGE) .

compose-up:
	@podman compose up -d --force-recreate

compose-down:
	@podman compose down

env:
	guix shell --network -m ./manifest.scm

pkg:
	guix build -f ./guix.scm
