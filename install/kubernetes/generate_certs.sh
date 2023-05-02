#!/bin/sh

COMMAND="/connector"

# We run this in the connector contaianer, if not running there
# fall back to default path
if ! command -v $COMMAND &> /dev/null; then
  COMMAND = "connector"
fi

rm -rf ./certs/local
rm -rf ./certs/k8s
rm -rf ./certs/root.cert
rm -rf ./certs/root.key

mkdir ./certs/local
mkdir ./certs/k8s

$COMMAND generate-certs --ca ./certs

# Generate the leaf certs for the k8s connector
$COMMAND generate-certs \
          --leaf \
          --ip-address 127.0.0.1 \
          --dns-name "localhost" \
          --dns-name "localhost:30090" \
          --dns-name "localhost:30091" \
          --dns-name ":9090" \
          --dns-name ":9091" \
          --dns-name "connector" \
					--dns-name "server.connector.k8s-cluster.shipyard.run:30090" \
					--dns-name "server.connector.k8s-cluster.shipyard.run:30091" \
          --dns-name "connector:9090" \
          --dns-name "connector:9091" \
          --root-ca ./certs/root.cert \
          --root-key ./certs/root.key \
          ./certs/k8s

# Generate certs for a local component
$COMMAND generate-certs \
          --leaf \
          --ip-address 127.0.0.1 \
          --dns-name ":9090" \
          --dns-name ":9091" \
          --dns-name "localhost" \
          --dns-name "localhost:9090" \
          --dns-name "localhost:9091" \
          --root-ca ./certs/root.cert \
          --root-key ./certs/root.key \
          ./certs/local

