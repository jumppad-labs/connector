#!/bin/bash

## Send a request to the connector to expose a local connection
curl -vv -k https://localhost:19091/expose \
  -d '{
  "name":"test1", "source_port": 13000, 
  "remote_connector_addr": "localhost:9090", 
  "destination_addr": "localhost:9091", 
  "type": "remote"
  }'