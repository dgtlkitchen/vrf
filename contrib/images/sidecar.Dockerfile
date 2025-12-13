# syntax=docker/dockerfile:1

# --------------------------------------------------------
# Arguments
# --------------------------------------------------------

ARG GO_VERSION="1.25.2"
ARG ALPINE_VERSION="3.22"

# --------------------------------------------------------
# Builder
# --------------------------------------------------------

FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS builder
ENV GOTOOLCHAIN=go1.25.2

RUN apk add --no-cache \
    ca-certificates \
    build-base \
    linux-headers \
    git

WORKDIR /vrf

# Copy Go dependencies
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/root/go/pkg/mod \
    go mod download

# Copy source code
COPY . .

# Build sidecar binary
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/root/go/pkg/mod \
    LEDGER_ENABLED=false BUILD_TAGS=muslc LINK_STATICALLY=true make build \
    && file /vrf/bin/sidecar \
    && echo "Ensuring sidecar binary is statically linked ..." \
    && (file /vrf/bin/sidecar | grep "statically linked")

# --------------------------------------------------------
# Runner
# --------------------------------------------------------

FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION}
COPY --from=builder /vrf/bin/sidecar /bin/sidecar

ENV HOME=/.sidecar
WORKDIR $HOME

EXPOSE 8080 8002

ENTRYPOINT ["sidecar"]
