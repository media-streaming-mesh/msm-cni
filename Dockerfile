# Build the manager binary
FROM --platform=$BUILDPLATFORM golang:1.17 as builder

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
    GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o msm-cni cmd/msm-cni/main.go

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o installer cmd/installer/main.go

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o msm-iptables util/msm-iptables/main.go util/msm-iptables/constants.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates

LABEL description="MSM CNI plugin installer."

COPY --from=builder /workspace/msm-cni /opt/cni/bin/
COPY --from=builder /workspace/msm-iptables /opt/cni/bin/
COPY --from=builder /workspace/installer /usr/local/bin/

ENV PATH=$PATH:/opt/cni/bin
WORKDIR /opt/cni/bin
CMD ["/usr/local/bin/installer"]
