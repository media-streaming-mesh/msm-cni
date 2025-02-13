# Build the manager binary
FROM --platform=$BUILDPLATFORM golang:1.24 AS builder

WORKDIR /workspace

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

RUN go mod download

# Copy the go source
COPY cmd/ cmd/
COPY internal/ internal/
COPY util/ util/

ARG TARGETOS TARGETARCH

# Build
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    GOOS=$TARGETOS GOARCH=$TARGETARCH CGO_ENABLED=0 go build -a -o msm-cni cmd/msm-cni/main.go

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    GOOS=$TARGETOS GOARCH=$TARGETARCH CGO_ENABLED=0 go build -a -o installer cmd/installer/main.go

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    GOOS=$TARGETOS GOARCH=$TARGETARCH CGO_ENABLED=0 go build -a -o msm-iptables util/msm-iptables/main.go util/msm-iptables/constants.go

FROM ubuntu:24.04

LABEL description="MSM CNI plugin installer."

COPY --from=builder /workspace/msm-cni /opt/cni/bin/msm-cni
COPY --from=builder /workspace/msm-iptables /opt/cni/bin/msm-iptables
COPY --from=builder /workspace/installer /usr/local/bin/installer

ENV PATH=$PATH:/opt/cni/bin
WORKDIR /opt/cni/bin
CMD ["/usr/local/bin/installer"]
