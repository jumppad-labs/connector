#!/bin/bash

rm -rf ./ca
rm -rf ./public

mkdir ./ca
mkdir ./public

go run ../main.go generate-certs --ca ./ca

go run ../main.go generate-certs \
          --leaf \
          --ip-address 127.0.0.1 \
          --dns-name ":9090" \
          --dns-name ":9091" \
          --dns-name "localhost" \
          --dns-name "localhost:9090" \
          --dns-name "localhost:9091" \
          --root-ca ./ca/root.cert \
          --root-key ./ca/root.key \
          ./public

go run ../main.go run --grpc-bind ":9090" --http-bind ":9091" \
 --log-level=debug \
 --root-cert-path ./ca/root.cert \
 --root-cert-key ./ca/root.key \
 --server-cert-path ./public/leaf.cert \
 --server-key-path ./public/leaf.key