ARG GO_IMAGE=docker.io/library/golang:1.26-alpine
ARG RUNTIME_IMAGE=gcr.io/distroless/static-debian12:nonroot

FROM ${GO_IMAGE} AS builder
WORKDIR src/

RUN apk add --no-cache ca-certificates make

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG BUILD_VERSION=dev
ARG BUILD_TAG=dev
ARG BUILD_COMMIT=unknown
ARG BUILD_DATE=unknown
ARG UPSTREAM_REPO=urutau-ltd/gavia

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS="${TARGETOS}" GOARCH="${TARGETARCH}" \
    make build \
      GOCACHE=/root/.cache/go-build \
      GIT_TAG="${BUILD_TAG}" \
      GIT_COMMIT="${BUILD_COMMIT}" \
      BUILD_DATE="${BUILD_DATE}" \
      UPSTREAM_REPO="${UPSTREAM_REPO}"

RUN install -d /build && \
    install -m 0755 /bin/gv /out/gv

FROM ${RUNTIME_IMAGE}
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

COPY --from=builder /out/gv /bin/gv

USER nonroot:nonroot
CMD ["/bin/gv"]
