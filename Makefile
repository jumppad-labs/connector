setup:
	go get -u google.golang.org/grpc
	go get -u github.com/golang/protobuf/protoc-gen-go

unit_test:
	#rm -rf /tmp/certs	
	#mkdir -p /tmp/certs
	#go run main.go generate-certs --ca /tmp/certs
	#
	## Generate the leaf certs for the k8s connector
	#go run main.go generate-certs \
  #        --leaf \
  #        --ip-address 127.0.0.1 \
  #        --dns-name "localhost" \
	#				--dns-name "localhost:1234" \
	#				--dns-name "localhost:1235" \
	#				--dns-name "localhost:1236" \
  #        --root-ca /tmp/certs/root.cert \
  #        --root-key /tmp/certs/root.key \
  #        /tmp/certs
	
	go test -coverprofile=coverage.txt -covermode=atomic -v ./...

proto:
	protoc -I ./protos protos/server.proto --go_out=plugins=grpc:protos/shipyard

build_docker:
	goreleaser release --rm-dist --snapshot
	docker tag gcr.io/shipyard-287511/connector:dev registry.shipyard.run/connector:dev

build_and_test: build_docker
	cd test/simple && shipyard test

install_local:
	go build -o ${GOPATH}/bin/connector .
