#!/bin/sh

# Create the secret in K8s
kubectl create secret -n shipyard-test tls connector-tls-ca --cert="./certs/root.cert" --key="./certs/root.key"
kubectl create secret -n shipyard-test tls connector-tls-leaf --cert="./certs/k8s/leaf.cert" --key="./certs/k8s/leaf.key"