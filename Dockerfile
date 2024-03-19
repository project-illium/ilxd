##
# Build Stage for Rust:
#   - Copy the source files from the host, filtered by .dockerignore to exclude unecessary 
#     things like the .git repo
#   - Build the Rust libraries using make, with the Makefile in the source root.
#   - Leave the Rust build products (libraries) in the /build directory, so they don't have
#     to be picked out of the source tree. They'll be in /build/release.
#
FROM rust:latest as rust-builder
WORKDIR /src
COPY . /src
RUN CARGO_TARGET_DIR=/build make rust-bindings

##
# Build Stage for Go:
#   - Install required system libaries (libssl-dev)
#   - Copy the source tree, and the Rust build products, from the Rust stage
#   - Build the go apps, leaving the build products (executables) in the /build directory.
#
FROM golang:1.21 AS go-builder
RUN apt update && apt install -y libssl-dev
WORKDIR /src
COPY --from=rust-builder /src /src
COPY --from=rust-builder /build /build

# Set necessary environment variables for Go, and build the apps
ENV GOPATH=/go
ENV PATH=$GOPATH/bin:$PATH
RUN CGO_LDFLAGS="-L/build/release" ARGS="-o /build" make go
 

##
# Final build stage: 
#   - copy the ilxd and ilxcli binaries from the Go stage.
#
FROM debian:latest
WORKDIR /app
# Copy the compiled Go binary from the build stage
COPY --from=go-builder /build/ilxd /app
COPY --from=go-builder /build/cli /app/ilxcli
# Expose necessary ports
EXPOSE 9002
EXPOSE 5001
ENV PATH="/app:${PATH}"
ENTRYPOINT ["/app/ilxd"]
CMD ["--alpha", "--loglevel=debug"]
