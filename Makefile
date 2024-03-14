protos:
	protoc -I=types/transactions -I=types/blocks --go_out=types/blocks --go_opt=paths=source_relative,Mtransactions.proto=github.com/project-illium/ilxd/types/transactions types/blocks/blocks.proto
	protoc -I=types/transactions -I=types/blocks --go_out=types/transactions --go_opt=paths=source_relative types/transactions/transactions.proto
	protoc -I=types/transactions -I=types/blocks -I=types/wire --go_out=types/wire --go_opt=paths=source_relative,Mtransactions.proto=github.com/project-illium/ilxd/types/transactions,Mblocks.proto=github.com/project-illium/ilxd/types/blocks types/wire/message.proto
	awk '/type Transaction struct {/,/}/{if ($$0 == "}") {print "	cachedTxid []byte\n\tcachedWid []byte"; print $$0; next} }1' types/transactions/transactions.pb.go > tmp && mv tmp types/transactions/transactions.pb.go
	protoc -I=blockchain/pb -I=types/transactions --go_out=blockchain/pb --go_opt=paths=source_relative,Mtransactions.proto=github.com/project-illium/ilxd/types/transactions blockchain/pb/db_models.proto
	protoc -I=net/pb --go_out=net/pb net/pb/db_net_models.proto
	protoc -I=rpc -I=types/transactions -I=types/blocks --go_out=rpc/pb --go-grpc_out=rpc/pb --go_opt=paths=source_relative,Mtransactions.proto=github.com/project-illium/ilxd/types/transactions,Mblocks.proto=github.com/project-illium/ilxd/types/blocks --go-grpc_opt=paths=source_relative rpc/ilxrpc.proto
	protoc -I=blockchain/indexers/pb --go_out=blockchain/indexers/pb blockchain/indexers/pb/db_indexer_models.proto

install:
ifdef CUDA
	@$(MAKE) build ARGS="-tags=cuda -o $(GOPATH)/bin/ilxd"
	cd cli && go build -tags=cuda -o $(GOPATH)/bin/ilxcli
else
	@$(MAKE) build ARGS="-o $(GOPATH)/bin/ilxd"
	cd cli && go build -o $(GOPATH)/bin/ilxcli
endif

.PHONY: build
build: rust-bindings
ifdef CUDA
	go build -tags=cuda $(ARGS) *.go
else
	go build $(ARGS) *.go
endif

.PHONY: rust-bindings
rust-bindings:
ifdef CUDA
	@cd crypto/rust && cargo build --release
	cd ../..
	@cd zk/rust && cargo build --release --features cuda
else
	export CUDA_PATH=
	export NVCC=off
	export EC_GPU_FRAMEWORK=none
	@cd crypto/rust && cargo build --release
	cd ../..
	@cd zk/rust && cargo build --release
endif

clean:
	cd zk/rust && cargo clean
	cd crypto/rust && cargo clean

update:
	cd zk/rust && cargo update
	cd crypto/rust && cargo update