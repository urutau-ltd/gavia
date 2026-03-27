# syntax=docker/dockerfile:1.7

ARG GO_IMAGE=docker.io/library/golang:1.26.1-alpine
ARG RUNTIME_IMAGE=gcr.io/distroless/static-debian12:nonroot

FROM ${GO_IMAGE} AS base
WORKDIR /workspace
ENV CGO_ENABLED=0
RUN apk add --no-cache ca-certificates make git

FROM base AS deps
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

FROM deps AS dev
COPY . .
ENV GOCACHE=/tmp/go-build
CMD ["make", "run-local"]

FROM deps AS builder
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG BUILD_VERSION=dev
ARG BUILD_TAG=dev
ARG BUILD_COMMIT=unknown
ARG BUILD_DATE=unknown
ARG UPSTREAM_REPO=urutau-ltd/gavia
ARG UPSTREAM_VENDOR=Urutau Limited

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/tmp/go-build \
    GOOS="${TARGETOS}" GOARCH="${TARGETARCH}" GOCACHE=/tmp/go-build \
    make build OUTPUT=/out/gv \
      BUILD_VERSION="${BUILD_VERSION}" \
      GIT_TAG="${BUILD_TAG}" \
      GIT_COMMIT="${BUILD_COMMIT}" \
      BUILD_DATE="${BUILD_DATE}" \
      UPSTREAM_REPO="${UPSTREAM_REPO}" \
      UPSTREAM_VENDOR="${UPSTREAM_VENDOR}"

FROM ${RUNTIME_IMAGE} AS runtime
WORKDIR /workspace

ARG BUILD_VERSION=dev
ARG BUILD_TAG=dev
ARG BUILD_COMMIT=unknown
ARG BUILD_DATE=unknown
ARG UPSTREAM_REPO=urutau-ltd/gavia

LABEL org.opencontainers.image.title="Gavia" \
      org.opencontainers.image.description="Infrastructure inventory control web service" \
      org.opencontainers.image.version="${BUILD_VERSION}" \
      org.opencontainers.image.revision="${BUILD_COMMIT}" \
      org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.source="https://github.com/${UPSTREAM_REPO}" \
      org.opencontainers.image.licenses="AGPL-3.0-or-later"

ENV GAVIA_ADDR=:9091 \
    GAVIA_DB_PATH=/workspace/db/app.sqlite \
    GAVIA_LOG_FORMAT=text \
    GAVIA_LOG_LEVEL=info

COPY --from=builder /out/gv /bin/gv
COPY --chown=nonroot:nonroot db/.gitkeep /workspace/db/.gitkeep

EXPOSE 9091

USER nonroot:nonroot
ENTRYPOINT ["/bin/gv"]
