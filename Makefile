setup_proto:
	go get -u google.golang.org/grpc
	go get -u github.com/golang/protobuf/protoc-gen-go

proto:
	protoc -I ./protos protos/server.proto --go_out=plugins=grpc:protos/shipyard

build_docker:
	goreleaser release --rm-dist --snapshot