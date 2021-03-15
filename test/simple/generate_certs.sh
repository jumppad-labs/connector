#!/bin/sh
rm -rf ./certs/root.cert
rm -rf ./certs/root.key
rm -rf ./certs/local
rm -rf ./certs/remote

mkdir -p ./certs/local
mkdir -p ./certs/remote

/connector generate-certs --ca ./certs

# Generate the leaf certs for the k8s connector
/connector generate-certs \
          --leaf \
          --ip-address 127.0.0.1 \
          --dns-name ":9090" \
          --dns-name ":9091" \
          --dns-name ":9092" \
          --dns-name ":9093" \
          --dns-name "connector" \
          --dns-name "connector:9090" \
          --dns-name "connector:9091" \
          --dns-name "connector:9092" \
          --dns-name "connector:9093" \
					--dns-name "*.container.shipyard.run:9090" \
					--dns-name "*.container.shipyard.run:9091" \
					--dns-name "*.container.shipyard.run:9092" \
					--dns-name "*.container.shipyard.run:9093" \
					--dns-name "local-connector.container.shipyard.run:9090" \
					--dns-name "local-connector.container.shipyard.run:9091" \
          --root-ca ./certs/root.cert \
          --root-key ./certs/root.key \
          ./certs/local

/connector generate-certs \
          --leaf \
          --ip-address 127.0.0.1 \
          --dns-name ":9090" \
          --dns-name ":9091" \
          --dns-name ":9092" \
          --dns-name ":9093" \
          --dns-name "connector" \
          --dns-name "connector:9090" \
          --dns-name "connector:9091" \
          --dns-name "connector:9092" \
          --dns-name "connector:9093" \
					--dns-name "*.container.shipyard.run:9090" \
					--dns-name "*.container.shipyard.run:9091" \
					--dns-name "*.container.shipyard.run:9092" \
					--dns-name "*.container.shipyard.run:9093" \
					--dns-name "remote-connector.container.shipyard.run:9092" \
					--dns-name "remote-connector.container.shipyard.run:9093" \
          --root-ca ./certs/root.cert \
          --root-key ./certs/root.key \
          ./certs/remote
