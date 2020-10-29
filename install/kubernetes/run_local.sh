#!/bin/bash

nohup connector run \
  --grpc-bind "localhost:9090" \
  --http-bind "localhost:9091" \
  --root-cert-path "./certs/root.cert" \
  --server-cert-path "./certs/local/leaf.cert" \
  --server-key-path "./certs/local/leaf.key" \
  > /tmp/connector.log 2>&1 & 
