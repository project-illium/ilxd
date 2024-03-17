# Build Stage for Rust
FROM rust:latest as rust-builder
WORKDIR /app
RUN --mount=type=cache,target=/tmp/docker-cache \
  --mount=type=bind,target=. \
  CARGO_TARGET_DIR="/build" \
  make VERBOSE=1 rust-bindings
RUN ls /tmp

# Build Stage for Go
FROM golang:1.21 AS go-builder
# Copy the source tree with compiled Rust from the previous stage
COPY --from=rust-builder /build/*.[ad] /build
RUN ls /build
# Set necessary environment variables for Go
ENV GOPATH=/go
ENV PATH=$GOPATH/bin:$PATH
# Build the Go application
RUN --mount=type=bind,target=. \
  make build
#RUN go build -ldflags="-r /app/lib" -o /app/ilxd *.go

# Final Stage
FROM debian:latest
WORKDIR /app
# Copy the compiled Go binary from the build stage
COPY --from=go-builder /app/ilxd /app
# Expose necessary ports
EXPOSE 9002
EXPOSE 5001
ENV PATH="/app:${PATH}"
ENTRYPOINT ["/app/ilxd"]
CMD ["--alpha", "--loglevel=debug"]
