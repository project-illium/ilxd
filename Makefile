sample-config:
	cd repo && go-bindata -pkg=repo sample-ilxd.conf

protos:
	cd models/transactions && PATH=$(PATH):$(GOPATH)/bin protoc --go_out=. *.proto
	cd models/blocks && PATH=$(PATH):$(GOPATH)/bin protoc --go_out=. --proto_path=../transactions --proto_path=../blocks *.proto
	cd models/blocks && sed -i '14d' blocks.pb.go
	cd models/blocks && sed -i '14i\"github.com/project-illium/ilxd/models/transactions"\' blocks.pb.go
	cd models/blocks && gofmt -s -w blocks.pb.go
	cd blockchain/pb && PATH=$(PATH):$(GOPATH)/bin protoc --go_out=. --proto_path=../../models/transactions --proto_path=../pb *.proto
	cd blockchain/pb && sed -i '10d' data_models.pb.go
	cd blockchain/pb && sed -i '10i\"github.com/project-illium/ilxd/models/transactions"\' data_models.pb.go
	cd blockchain/pb && gofmt -s -w data_models.pb.go
	cd models/wire && PATH=$(PATH):$(GOPATH)/bin protoc --go_out=. *.proto