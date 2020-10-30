setup_proto:
	go get -u google.golang.org/grpc
	go get -u github.com/golang/protobuf/protoc-gen-go

unit_test:
	rm -rf /tmp/certs	
	mkdir -p /tmp/certs
	go run main.go generate-certs --ca /tmp/certs
	
	# Generate the leaf certs for the k8s connector
	go run main.go generate-certs \
          --leaf \
          --ip-address 127.0.0.1 \
          --dns-name ":9090" \
          --dns-name ":9091" \
          --dns-name ":9092" \
          --dns-name ":9093" \
          --dns-name "connector" \
          --dns-name "localhost" \
					--dns-name "localhost:1234" \
					--dns-name "localhost:1235" \
					--dns-name "localhost:1236" \
          --dns-name "connector:9090" \
          --dns-name "connector:9091" \
					--dns-name "*.container.shipyard.run:9090" \
					--dns-name "*.container.shipyard.run:9091" \
					--dns-name "*.container.shipyard.run:9092" \
					--dns-name "*.container.shipyard.run:9093" \
					--dns-name "local-connector.container.shipyard.run:9090" \
					--dns-name "local-connector.container.shipyard.run:9091" \
          --root-ca /tmp/certs/root.cert \
          --root-key /tmp/certs/root.key \
          /tmp/certs
	
	gotestsum -- -coverprofile=coverage.txt -covermode=atomic -v -p 1 ./...

proto:
	protoc -I ./protos protos/server.proto --go_out=plugins=grpc:protos/shipyard

build_docker:
	goreleaser release --rm-dist --snapshot
	docker tag gcr.io/shipyard-287511/connector:dev registry.shipyard.run/connector:dev

build_and_test: build_docker
	cd test/simple && shipyard test

install_local:
	go build -o ${GOPATH}/bin/connector .
