# Build the model-registry binary
FROM --platform=$BUILDPLATFORM registry.access.redhat.com/ubi9/go-toolset:1.26 AS common
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace
# Copy the Go Modules manifests and workspace file
COPY ["go.mod", "go.sum", "./"]
COPY ["pkg/openapi/go.mod", "pkg/openapi/"]
COPY ["catalog/pkg/openapi/go.mod", "catalog/pkg/openapi/"]
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Download tools
COPY Makefile .
COPY ["main.go", ".openapi-generator-ignore", "openapitools.json", "./"]
COPY cmd/ cmd/
COPY api/ api/
COPY internal/ internal/
COPY scripts/ scripts/
COPY pkg/ pkg/
COPY templates/ templates/
COPY patches/ patches/
COPY catalog/ catalog/

FROM common AS builder

USER root

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} make build/compile

# Use distroless as minimal base image to package the model-registry binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM registry.access.redhat.com/ubi9/ubi-minimal:latest
WORKDIR /
# copy the registry binary
COPY --from=builder /workspace/model-registry .
USER 65532:65532

ENTRYPOINT ["/model-registry"]
