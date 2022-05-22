sample-config:
	cd repo && go-bindata -pkg=repo sample-ilxd.conf

protos:
	cd types/transactions && PATH=$(PATH):$(GOPATH)/bin protoc --go_out=. *.proto
	cd types/blocks && PATH=$(PATH):$(GOPATH)/bin protoc --go_out=. --proto_path=../transactions --proto_path=../blocks *.proto
	cd types/blocks && sed -i '14d' blocks.pb.go
	cd types/blocks && sed -i '14i\"github.com/project-illium/ilxd/models/transactions"\' blocks.pb.go
	cd types/blocks && gofmt -s -w blocks.pb.go
	cd blockchain/pb && PATH=$(PATH):$(GOPATH)/bin protoc --go_out=. --proto_path=../../models/transactions --proto_path=../pb *.proto
	cd blockchain/pb && sed -i '10d' db_models.pb.go
	cd blockchain/pb && sed -i '10i\"github.com/project-illium/ilxd/models/transactions"\' db_models.pb.go
	cd blockchain/pb && gofmt -s -w db_models.pb.go
	cd types/wire && PATH=$(PATH):$(GOPATH)/bin protoc --go_out=. *.proto