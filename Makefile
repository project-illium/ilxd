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

install:
	go install
	cd cli && go build -o $(GOPATH)/bin/ilxcli

ROOT_DIR := $(dir $(realpath $(lastword $(MAKEFILE_LIST))))

.PHONY: build-dynamic
build-dynamic:
	mkdir -p lib
	@cd crypto/rust && cargo build --release
	@cp crypto/rust/target/release/libillium_crypto.so lib/
	go build -ldflags="-r $(ROOT_DIR)lib" *.go

test-crypto:
	export LD_LIBRARY_PATH=$(pwd)/lib:$LD_LIBRARY_PATH
	CGO_ENABLED=1 go test -c -ldflags="-r $(ROOT_DIR)lib" -o mytest ./crypto
	LD_LIBRARY_PATH=$(pwd)/lib ./mytest
	#LDFLAGS="-r $(ROOT_DIR)lib" CGO_ENABLED=1 go test -v ./crypto