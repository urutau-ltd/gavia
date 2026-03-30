#!/bin/sh
set -eu

GO_BIN="${GO:-go}"
CGO_ENABLED_VALUE="${CGO_ENABLED:-0}"
GO_CACHE_DIR="${GOCACHE:-/tmp/go-build}"
LDFLAGS_VALUE="${LDFLAGS:--s -w -buildid=}"

if [ -n "${GIT_COMMIT:-}" ]; then
    BUILD_COMMIT_VALUE="${GIT_COMMIT}"
else
    BUILD_COMMIT_VALUE="$(git rev-parse --short=12 HEAD 2>/dev/null || echo unknown)"
fi

if [ -n "${GIT_TAG:-}" ]; then
    BUILD_TAG_VALUE="${GIT_TAG}"
else
    BUILD_TAG_VALUE="$(git describe --tags --abbrev=0 2>/dev/null || echo dev)"
fi

BUILD_VERSION_VALUE="${BUILD_VERSION:-${BUILD_TAG_VALUE}}"
BUILD_DATE_VALUE="${BUILD_DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"
UPSTREAM_REPO_VALUE="${UPSTREAM_REPO:-urutau-ltd/gavia}"
UPSTREAM_VENDOR_VALUE="${UPSTREAM_VENDOR:-Urutau Limited}"
OUTPUT_PATH="${OUTPUT_PATH:-/tmp/gavia-dev-bin}"

BUILD_LDFLAGS_VALUE="-X 'main.buildVersion=${BUILD_VERSION_VALUE}' -X 'main.buildTag=${BUILD_TAG_VALUE}' -X 'main.buildCommit=${BUILD_COMMIT_VALUE}' -X 'main.buildDate=${BUILD_DATE_VALUE}' -X 'main.upstreamRepo=${UPSTREAM_REPO_VALUE}' -X 'main.upstreamVendor=${UPSTREAM_VENDOR_VALUE}'"

env CGO_ENABLED="${CGO_ENABLED_VALUE}" GOCACHE="${GO_CACHE_DIR}" \
    "${GO_BIN}" build -trimpath -ldflags "${LDFLAGS_VALUE} ${BUILD_LDFLAGS_VALUE}" -o "${OUTPUT_PATH}" .

exec "${OUTPUT_PATH}"
