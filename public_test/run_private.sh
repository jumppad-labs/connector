#!/bin/bash

rm -rf ./private
mkdir ./private

payload=$(curl -k https://localhost:9091/certificate -s -d '{
  "name": "private",
  "ip_addresses":["127.0.0.1"],
  "dns_names": [
    ":19090",
    ":19091",
    "localhost",
    "localhost:19090",
    "localhost:19091"
  ]
}')

echo $payload | jq -r '.ca' > ./private/ca.cert 
echo $payload | jq -r '.certificate' > ./private/leaf.cert 
echo $payload | jq -r '.private_key' > ./private/leaf.key


go run ../main.go run --grpc-bind ":19090" --http-bind ":19091" \
 --log-level=debug \
 --root-cert-path ./private/ca.cert \
 --server-cert-path ./private/leaf.cert \
 --server-key-path ./private/leaf.key