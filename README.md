[![Go](https://github.com/project-illium/ilxd/actions/workflows/go.yml/badge.svg)](https://github.com/project-illium/ilxd/actions/workflows/go.yml)
[![golangci-lint](https://github.com/project-illium/ilxd/actions/workflows/golangci-lint.yml/badge.svg)](https://github.com/project-illium/ilxd/actions/workflows/golangci-lint.yml)

# ilxd
illium full node implementation written in Go

This is an alpha version of the illium full node software. This software does *not* have the proving system built in and 
is using mock proofs. 

The purpose is to validate all the rest of the code, networking, blockchain maintenance, transaction processing, etc, 
before we turn our attention to the proofs. 

If you want to test this alpha version you can download the binaries from the github releases page and run the node with
the `--alpha` flag.

```go
ilxd --alpha
```