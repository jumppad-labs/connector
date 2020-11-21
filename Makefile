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

build_docker: snapshot
	docker tag gcr.io/shipyard-287511/connector:dev registry.shipyard.run/connector:dev

build_and_test: build_docker
	cd test/simple && shipyard test

install_local:
	go build -o ${GOPATH}/bin/connector .

snapshot:
	goreleaser release --rm-dist --snapshot

setup_multiarch:
	docker run --rm --privileged multiarch/qemu-user-static --reset -p yes
	#docker run --rm --privileged docker/binfmt:820fdd95a9972a5308930a2bdfb8573dd4447ad3
	docker buildx create --name multi
	docker buildx use multi
	docker buildx inspect --bootstrap

clean_multiarch:
	docker buildx rm multi

build_multi_docker:
	docker buildx build --platform linux/arm/v7,linux/amd64 \
		-t gcr.io/shipyard-287511/connector:multi \
		-f ./Dockerfile \
		./dist \
		--push

