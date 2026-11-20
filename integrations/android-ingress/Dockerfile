# Isolated Android -> Russian-server Bridge Decapsulator.
# Build from the current checked-out source. This does not change running services.
FROM golang:1.23-bookworm AS builder
WORKDIR /src
COPY prototype/ ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o /out/tervon-bridge ./cmd/bridge

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/tervon-bridge /tervon-bridge
USER 65532:65532
EXPOSE 8443/tcp
ENTRYPOINT ["/tervon-bridge", "-config", "/config/server.json"]
