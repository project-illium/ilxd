sample-config:
	cd repo && go-bindata -pkg=repo sample-ilxd.conf

protos:
	protoc -I=types/transactions -I=types/blocks --go_out=types/blocks --go_opt=paths=source_relative,Mtransactions.proto=github.com/project-illium/ilxd/types/transactions types/blocks/blocks.proto
	protoc -I=types/transactions -I=types/blocks --go_out=types/transactions --go_opt=paths=source_relative types/transactions/transactions.proto
	protoc -I=types/transactions -I=types/blocks -I=types/wire --go_out=types/wire --go_opt=paths=source_relative,Mtransactions.proto=github.com/project-illium/ilxd/types/transactions,Mblocks.proto=github.com/project-illium/ilxd/types/blocks types/wire/message.proto
	awk '/type Transaction struct {/,/}/{if ($$0 == "}") {print "	cachedTxid []byte"; print $$0; next} }1' types/transactions/transactions.pb.go > tmp && mv tmp types/transactions/transactions.pb.go
	protoc -I=blockchain/pb -I=types/transactions --go_out=blockchain/pb --go_opt=paths=source_relative,Mtransactions.proto=github.com/project-illium/ilxd/types/transactions blockchain/pb/db_models.proto
	protoc -I=net/pb --go_out=net/pb net/pb/db_net_models.proto
	protoc -I=rpc -I=types/transactions -I=types/blocks --go_out=rpc/pb --go-grpc_out=rpc/pb --go_opt=paths=source_relative,Mtransactions.proto=github.com/project-illium/ilxd/types/transactions,Mblocks.proto=github.com/project-illium/ilxd/types/blocks --go-grpc_opt=paths=source_relative rpc/ilxrpc.proto

ROOT_DIR := $(dir $(realpath $(lastword $(MAKEFILE_LIST))))

install:
	@$(MAKE) build ARGS="-o $(GOPATH)/bin/ilxd"
	cd cli && go build -o $(GOPATH)/bin/ilxcli

.PHONY: build
build: ensure-rust-installed rust-bindings
	go build $(ARGS) *.go

.PHONY: rust-bindings
rust-bindings:
	mkdir -p lib
	@cd crypto/rust && cargo build --release
	@cp crypto/rust/target/release/libillium_crypto.a lib/
	cd ../..
	@cd zk/rust && cargo build --release
	@cp zk/rust/target/release/libillium_zk.a lib/

.PHONY: ensure-rust-installed
ensure-rust-installed:
ifeq ($(OS),Windows_NT)
	if not exist $(CARGO_HOME) (
		echo "Cargo is not available, installing Rust..."
		PowerShell -Command "Invoke-WebRequest -OutFile rustup-init.exe https://win.rustup.rs/x86_64"
		./rustup-init.exe -y
		del rustup-init.exe
	) else (
		echo "Cargo is already installed"
	)
else
ifndef CARGO_HOME
	$(eval CARGO_HOME := $(HOME)/.cargo)
endif
ifeq (, $(shell which cargo))
	@echo "Cargo is not available, installing Rust..."
	@curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
	@export PATH=$(CARGO_HOME)/bin:$(PATH)
else
	@echo "Cargo is already installed"
endif
endif

test-crypto: rust-bindings
	export LD_LIBRARY_PATH=$(pwd)/lib:$LD_LIBRARY_PATH
	CGO_ENABLED=1 go test -v -c -ldflags="-r $(ROOT_DIR)lib" -o crypto_test ./crypto
	LD_LIBRARY_PATH=$(pwd)/lib ./crypto_test
	rm -f ./crypto_test