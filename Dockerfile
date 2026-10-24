
# syntax=docker/dockerfile:1

# Comments are provided throughout this file to help you get started.
# If you need more help, visit the Dockerfile reference guide at
# https://docs.docker.com/engine/reference/builder/

################################################################################
# Create a stage for building the application.
ARG GO_VERSION=1.26
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION} AS build
ARG TARGETPLATFORM
ARG BUILDPLATFORM
WORKDIR /src

# Download dependencies as a separate step to take advantage of Docker's caching.
# Leverage a cache mount to /go/pkg/mod/ to speed up subsequent builds.
# Leverage bind mounts to go.sum and go.mod to avoid having to copy them into
# the container.
RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=bind,source=go.mod,target=go.mod \
    go mod download -x

# Build the application.
# Leverage a cache mount to /go/pkg/mod/ to speed up subsequent builds.
# Leverage a bind mount to the current directory to avoid having to copy the
# source code into the container.
ARG TARGETOS 
ARG TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,target=. \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /bin/cdk-notifier .

################################################################################
# Create a new stage for running the application that contains the minimal
# runtime dependencies for the application.
#
# cgr.dev/chainguard/static is a hardened, minimal base for statically linked
# binaries (built here with CGO_ENABLED=0). It ships a CA-certificates bundle
# (SSL_CERT_FILE is preset) and runs as the non-root user "nonroot" (uid/gid
# 65532) by default. It contains no shell and no package manager, which
# minimizes the attack surface and removes OS-package CVEs such as the Alpine
# musl advisories.
FROM cgr.dev/chainguard/static:latest AS final

# Copy the statically linked executable from the "build" stage.
COPY --from=build /bin/cdk-notifier /bin/cdk-notifier

# What the container should run when it is started.
ENTRYPOINT [ "/bin/cdk-notifier" ]
