# Installer for Kubernetes

```
connector run \
          --grpc-bind ":9090" \
          --http-bind ":9091" \
          --root-cert-path "./certs/root.cert" \
          --server-cert-path "./certs/local/leaf.cert" \
          --server-key-path "./certs/local/leaf.key"
```

# Expose an example app

```
curl -k https://localhost:9091/expose -d \
  '{
		"name":"devservice", 
    "source_port": 9099, 
		"remote_connector_addr": "localhost:19091", 
    "destination_addr": "localhost:9093",
    "type": "local"
  }'
```
