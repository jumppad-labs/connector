#!/bin/sh

COMMAND="/connector"

# We run this in the connector contaianer, if not running there
# fall back to default path
if ! command -v $COMMAND &> /dev/null; then
  COMMAND="connector"
fi

rm -rf ./certs/local
rm -rf ./certs/nomad
rm -rf ./certs/root.cert
rm -rf ./certs/root.key

mkdir -p ./certs/local
mkdir -p ./certs/nomad

$COMMAND generate-certs --ca ./certs

# Define the DNS name and Ports for the public entrypoint for the Nomad server
NOMAD_PUBLIC_ADDRESS="localhost"
NOMAD_PUBLIC_ADDRESS_API="${NOMAD_PUBLIC_ADDRESS}:30090"
NOMAD_PUBLIC_ADDRESS_GRPC="${NOMAD_PUBLIC_ADDRESS}:30091"

# Generate the leaf certs for the nomad connector
$COMMAND generate-certs \
          --leaf \
          --ip-address 127.0.0.1 \
          --dns-name "${NOMAD_PUBLIC_ADDRESS}" \
          --dns-name "${NOMAD_PUBLIC_ADDRESS_API}" \
          --dns-name "${NOMAD_PUBLIC_ADDRESS_GRPC}" \
          --dns-name ":9090" \
          --dns-name ":9091" \
          --dns-name "connector" \
					--dns-name "server.dev.nomad-cluster.shipyard.run:30090" \
					--dns-name "server.dev.nomad-cluster.shipyard.run:30091" \
          --dns-name "connector:9090" \
          --dns-name "connector:9091" \
          --root-ca ./certs/root.cert \
          --root-key ./certs/root.key \
          ./certs/nomad

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

