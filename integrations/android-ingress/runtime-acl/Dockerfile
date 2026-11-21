# Build ONLY the isolated Android-facing M6 hot-ACL candidate.
# Build context MUST be a clean RELEASE v1.0.1 tree (commit 96c1367)
# already staged by apply-to-release-source.py.
# Never build from the old prototype on work/m6-android-ingress.
FROM golang:1.25.5-bookworm AS build
WORKDIR /src
COPY prototype/ ./
RUN grep -Fq 'M6_ACL_RELOAD=PASS digest=' cmd/bridge/main.go \
 && grep -Fq 'AuthorizationLookup' bridge.go \
 && test -f acl_snapshot.go \
 && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o /out/tervon-bridge ./cmd/bridge

FROM gcr.io/distroless/static-debian12:nonroot
LABEL org.tervon.bridge.hot_acl="m6-v1"
COPY --from=build /out/tervon-bridge /tervon-bridge
USER 65532:65532
EXPOSE 8443/tcp
ENTRYPOINT ["/tervon-bridge","-config","/config/server.json"]
