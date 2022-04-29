# Build the manager binary
FROM golang:1.17.9 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/
COPY pkg/ pkg/
COPY cmd/ cmd/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o arlon main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
# Note that distroless images do not contain shell
# Use busybox to debug with shell and tools like ls, cat
#FROM busybox:1.35.0-uclibc as busybox
WORKDIR /
COPY --from=builder /workspace/arlon .
COPY deploy/manifests deploy/manifests
COPY config config
USER 65532:65532

ENTRYPOINT ["/arlon"]

# Uncomment when debugging with busybox image : to debug with shell and tools like ls, cat
#ENTRYPOINT [ "/bin/sh" ]