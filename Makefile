SHELL := /bin/sh

GO ?= go
CGO_ENABLED ?= 0
GOCACHE := /tmp/go-build

BINARY ?= gavia
BUILD_DIR ?= build
OUTPUT ?= $(BUILD_DIR)/$(BINARY)

INSTALL_NAME ?= gv
INSTALL_ROOT_DIR ?= /bin

IMAGE ?= ghcr.io/urutau-ltd/gavia:latest
IMAGE_PLATFORM ?= linux/amd64

LDFLAGS ?= -s -w -buildid=

GIT_COMMIT ?= $(shell git rev-parse --short=12 HEAD 2>/dev/null || echo unknown)
GIT_TAG ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo dev)
BUILD_VERSION ?= $(GIT_TAG)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
UPSTREAM_REPO ?= urutau-ltd/gavia
UPSTREAM_VENDOR ?= Urutau Limited
BUILD_LDFLAGS ?= \
	-X 'main.buildVersion=$(BUILD_VERSION)' \
	-X 'main.buildTag=$(GIT_TAG)' \
	-X 'main.buildCommit=$(GIT_COMMIT)' \
	-X 'main.buildDate=$(BUILD_DATE)' \
	-X 'main.upstreamRepo=$(UPSTREAM_REPO)' \
	-X 'main.upstreamVendor=$(UPSTREAM_VENDOR)'
BUILD_FLAGS ?= $(BUILD_LDFLAGS)

FUZZTIME ?= 5s
GO_SOURCES := $(shell find . -type f -name '*.go' -not -path './vendor/*')

.PHONY: fmt fmt-check vet test test-race ci build install run run-local clean image compose-up compose-dev-up compose-down compose-logs env pkg

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
	GOCACHE=$(GOCACHE) $(GO) vet ./...

test:
	CGO_ENABLED=$(CGO_ENABLED) GOCACHE=$(GOCACHE) $(GO) test -v ./...

test-race:
	CGO_ENABLED=$(CGO_ENABLED) GOCACHE=$(GOCACHE) $(GO) test -race ./...

ci: fmt-check vet test test-race

build:
	mkdir -p $(dir $(OUTPUT))
	CGO_ENABLED=$(CGO_ENABLED) GOCACHE=$(GOCACHE) $(GO) build -trimpath -ldflags "$(LDFLAGS) $(BUILD_FLAGS)" -o $(OUTPUT) .

run:
	@podman compose --profile dev up --build gavia-dev

run-local:
	CGO_ENABLED=$(CGO_ENABLED) GOCACHE=$(GOCACHE) $(GO) run -ldflags "$(LDFLAGS) $(BUILD_FLAGS)" .

install:
	install -d "$(INSTALL_ROOT_DIR)"
	install -m 0755 "$(OUTPUT)" "$(INSTALL_ROOT_DIR)/$(INSTALL_NAME)"

clean:
	rm -rf $(BUILD_DIR)

image:
	podman build --target runtime --platform $(IMAGE_PLATFORM) \
		--build-arg TARGETOS=linux \
		--build-arg TARGETARCH=$$(echo "$(IMAGE_PLATFORM)" | awk -F/ '{if ($$2 == "") print $$1; else print $$2}') \
		--build-arg BUILD_VERSION=$(BUILD_VERSION) \
		--build-arg BUILD_TAG=$(GIT_TAG) \
		--build-arg BUILD_COMMIT=$(GIT_COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		--build-arg UPSTREAM_REPO=$(UPSTREAM_REPO) \
		--build-arg UPSTREAM_VENDOR=$(UPSTREAM_VENDOR) \
		-t $(IMAGE) .

compose-up:
	@podman compose up -d --build gavia

compose-dev-up:
	@podman compose --profile dev up --build gavia-dev

compose-down:
	@podman compose down

compose-logs:
	@podman compose logs -f --tail=100

env:
	guix shell --network -m ./manifest.scm

pkg:
	@test -f ./guix.scm || { \
		echo "guix.scm is not in this repository yet. Use 'make env' for the shell workflow."; \
		exit 1; \
	}
	guix build -f ./guix.scm
