# Build Stage for Rust
FROM rust:latest as rust-builder
WORKDIR /app
COPY ./crypto/rust /app
RUN cargo build --release

# Build Stage for Go
FROM golang:1.21 AS go-builder
WORKDIR /app
COPY . /app
# Copy the compiled Rust library from the previous stage
COPY --from=rust-builder /app/target/release/libillium_crypto.so /app/lib/
# Set necessary environment variables for Go
ENV GOPATH=/go
ENV PATH=$GOPATH/bin:$PATH
# Build the Go application
RUN go build -ldflags="-r /app/lib" -o /app/ilxd *.go

# Final Stage
FROM debian:latest
WORKDIR /app
# Copy the compiled Go binary from the build stage
COPY --from=go-builder /app/ilxd /app
# Copy the Rust library
COPY --from=rust-builder /app/target/release/libillium_crypto.so /app/lib/
# Expose necessary ports
EXPOSE 9002
EXPOSE 5001
ENV PATH="/app:${PATH}"
ENTRYPOINT ["/app/ilxd"]
CMD ["--alpha", "--loglevel=debug"]
