sample-config:
	cd repo && go-bindata -pkg=repo sample-ilxd.conf

protos:
	protoc -I=types/transactions -I=types/blocks --go_out=types/blocks --go_opt=paths=source_relative,Mtransactions.proto=github.com/project-illium/ilxd/types/transactions types/blocks/blocks.proto
	protoc -I=types/transactions -I=types/blocks --go_out=types/transactions --go_opt=paths=source_relative types/transactions/transactions.proto
	protoc -I=types/transactions -I=types/blocks -I=types/wire --go_out=types/wire --go_opt=paths=source_relative,Mtransactions.proto=github.com/project-illium/ilxd/types/transactions,Mblocks.proto=github.com/project-illium/ilxd/types/blocks types/wire/message.proto
	protoc -I=rpc -I=types/transactions -I=types/blocks --go_out=rpc/pb --go-grpc_out=rpc/pb --go_opt=paths=source_relative,Mtransactions.proto=github.com/project-illium/ilxd/types/transactions,Mblocks.proto=github.com/project-illium/ilxd/types/blocks --go-grpc_opt=paths=source_relative rpc/ilxrpc.proto


