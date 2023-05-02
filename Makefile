GIT_SHA_FETCH := $(shell git rev-parse HEAD | cut -c 1-8)
export GIT_SHA=$(GIT_SHA_FETCH)

setup:
	go get -u google.golang.org/grpc
	go get -u github.com/golang/protobuf/protoc-gen-go

unit_test:
	go test -coverprofile=coverage.txt -covermode=atomic -v ./...

proto:
	protoc -I ./protos protos/server.proto --go_out=plugins=grpc:protos/shipyard

install_local:
	go build -o ${GOPATH}/bin/connector .

snapshot:
	SHA=$(shell "git rev-parse HEAD")
	rm -rf ./dist
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.build=$(GIT_SHA)" -o ./dist/linux_amd64/connector main.go
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w -X main.build=$(GIT_SHA)" -o ./dist/darwin_amd64/connector main.go
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w -X main.build=$(GIT_SHA)" -o ./dist/linux_arm64/connector main.go
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w -X main.build=$(GIT_SHA)" -o ./dist/darwin_arm64/connector main.go

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

run_local:
	go run main.go \
		run \
		--grpc-bind=:9090 \
		--http-bind=:9091 \
		--log-level=debug \
		--root-cert-path=./install/nomad/certs/root.cert \
		--server-cert-path=./install/nomad/certs/local/leaf.cert \
		--server-key-path=./install/nomad/certs/local/leaf.key

run_nomad:
	go run main.go \
		run \
		--grpc-bind=:19090 \
		--http-bind=:19091 \
		--log-level=debug \
		--root-cert-path=./install/nomad/certs/root.cert \
		--server-cert-path=./install/nomad/certs/nomad/leaf.cert \
		--server-key-path=./install/nomad/certs/nomad/leaf.key \
		--integration=nomad