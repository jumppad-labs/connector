setup_proto:
	go get -u google.golang.org/grpc
	go get -u github.com/golang/protobuf/protoc-gen-go

proto:
	protoc -I ./protos protos/server.proto --go_out=plugins=grpc:protos/shipyard

build_docker:
	goreleaser release --rm-dist --snapshot
	docker tag gcr.io/shipyard-287511/connector:dev registry.shipyard.run/connector:dev

build_and_test: build_docker
	cd test/simple && shipyard test

install_local:
	go build -o ${GOPATH}/bin/connector .
