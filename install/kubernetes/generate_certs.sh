#!/bin/bash

export KUBECONFIG="${HOME}/.shipyard/config/connector/kubeconfig.yaml"

rm -rf ./certs

mkdir ./certs
connector generate-certs --ca ./certs

mkdir ./certs/k8s
mkdir ./certs/local

# Define the DNS name and Ports for the public entrypoint for the K8s server
K8S_PUBLIC_ADDRESS="localhost"
K8S_PUBLIC_ADDRESS_API="${K8S_PUBLIC_ADDRESS}:19090"
K8S_PUBLIC_ADDRESS_GRPC="${K8S_PUBLIC_ADDRESS}:19091"

# Generate the leaf certs for the k8s connector
connector generate-certs \
          --leaf \
          --ip-address 127.0.0.1 \
          --dns-name "${K8S_PUBLIC_ADDRESS}" \
          --dns-name "${K8S_PUBLIC_ADDRESS_API}" \
          --dns-name "${K8S_PUBLIC_ADDRESS_GRPC}" \
          --dns-name ":9090" \
          --dns-name ":9091" \
          --dns-name "connector" \
          --dns-name "connector:9090" \
          --dns-name "connector:9091" \
          --root-ca ./certs/root.cert \
          --root-key ./certs/root.key \
          ./certs/k8s

# Generate certs for a local component
connector generate-certs \
          --leaf \
          --ip-address 127.0.0.1 \
          --dns-name "localhost" \
          --dns-name "localhost:9090" \
          --dns-name "localhost:9091" \
          --root-ca ./certs/root.cert \
          --root-key ./certs/root.key \
          ./certs/local

# Create the secret in K8s
kubectl create secret -n shipyard tls connector-tls-ca --cert="./certs/root.cert" --key="./certs/root.key"
kubectl create secret -n shipyard tls connector-tls-leaf --cert="./certs/k8s/leaf.cert" --key="./certs/k8s/leaf.key"
