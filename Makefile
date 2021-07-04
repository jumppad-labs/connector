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

install_local:
	go build -o ${GOPATH}/bin/connector .

snapshot:
	goreleaser release --rm-dist --snapshot

setup_multiarch:
	docker run --rm --privileged multiarch/qemu-user-static --reset -p yes
	docker buildx create --name multi
	docker buildx use multi
	docker buildx inspect --bootstrap

clean_multiarch:
	docker buildx rm multi || true

build_docker: clean_multiarch snapshot setup_multiarch
	docker buildx build --platform linux/amd64 \
		-t connector:dev \
		-f ./Dockerfile \
		./dist \
		--load

push_multi_docker:
	docker buildx build --platform linux/arm/v6,linux/arm/v7,linux/arm64,linux/amd64 \
		-t shipyardrun/connector:dev \
		-f ./Dockerfile \
		./dist \
		--push

build_and_test: build_docker
	cd test/simple && shipyard test --var connector_image=connector:dev
	cd test/kubernetes && shipyard test --var connector_image=connector:dev

build_dev:
	docker build \
		-t connector:dev \
		-f ./Dockerfile.dev \
		.

run_dev:
	cd test/simple && shipyard run --var connector_image=connector:dev

setup_local_dev:
	curl -vv -k https://localhost:9091/expose -d '{"name":"test1", "source_port": 13000, "remote_connector_addr": "remote-connector.container.shipyard.run:9092", "destination_addr": "local-service.container.shipyard.run:9094", "type": "local"}'
