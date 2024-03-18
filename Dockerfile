# Build Stage for Rust
FROM rust:latest as rust-builder
WORKDIR /app
RUN --mount=type=bind,target=. \
  make rust-bindings

# Build Stage for Go
FROM golang:1.21 AS go-builder
# Set necessary environment variables for Go
ENV GOPATH=/go
ENV PATH=$GOPATH/bin:$PATH
# Build the Go application
RUN --mount=type=bind,target=.,rw \
  make go 
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
