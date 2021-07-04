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

# Define the DNS name and Ports for the public entrypoint for the K8s server
K8S_PUBLIC_ADDRESS="localhost"
K8S_PUBLIC_ADDRESS_API="${K8S_PUBLIC_ADDRESS}:19090"
K8S_PUBLIC_ADDRESS_GRPC="${K8S_PUBLIC_ADDRESS}:19091"

# Generate the leaf certs for the k8s connector
$COMMAND generate-certs \
          --leaf \
          --ip-address 127.0.0.1 \
          --dns-name "${K8S_PUBLIC_ADDRESS}" \
          --dns-name "${K8S_PUBLIC_ADDRESS_API}" \
          --dns-name "${K8S_PUBLIC_ADDRESS_GRPC}" \
          --dns-name ":9090" \
          --dns-name ":9091" \
          --dns-name "connector" \
					--dns-name "connector.ingress.shipyard.run:19090" \
					--dns-name "connector.ingress.shipyard.run:19091" \
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

