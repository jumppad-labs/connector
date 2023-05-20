#!/bin/bash -e

if [[ $1 == "run_local" ]];then
	go run ../main.go run --bind ":29090" \
 	--log-level=debug
fi

if [[ $1 == "run_remote" ]];then
	go run ../main.go run --bind ":19090" \
 	--log-level=debug
fi

if [[ $1 == "run_public" ]];then
	go run ../main.go run --bind ":9090" \
 	--log-level=debug
fi

if [[ $1 == "create_test" ]];then
	# expose the local service localhost:19090 to the public connector
	# at port 13000
	curl -vv -k http://localhost:29090/expose -d '{
	 "remote_connector_addr": "localhost:9090", 
	 "type": "local",
   "config": {
      "port": "13000",
	    "address": "localhost:29090"
    }
	}'

  # expose the public service localhost:13000 to the local machine
  # at port 13000
  curl -vv -k http://localhost:19090/expose -d '{
    "remote_connector_addr": "localhost:9090", 
    "type": "remote",
    "config": {
      "port": "13001",
      "address": "localhost:13000"
     }
  }'

fi

if [[ $1 == "create_nomad" ]];then
	# expose the local service localhost:19090 to the public connector
	# at port 13000
	curl -vv -k http://localhost:19090/expose -d '{
	 "remote_connector_addr": "localhost:9090", 
	 "type": "local",
   "config": {
	    "address": "localhost:19090",
      "port": "13000"
    }
	}'

  # expose the public service localhost:13000 to the local machine
  # at port 13000
  curl -vv -k http://localhost:29090/expose -d '{
    "remote_connector_addr": "localhost:9090", 
    "type": "remote",
    "config": {
      "job": "local",
      "task": "nomad",
      "group": "service",
      "job_port": "tcp"
      "port": "9090"
     }
  }'
fi

if [[ $1 == "create_kubernetes" ]];then
	# expose the local service localhost:19090 to the public connector
	# at port 13000
	curl -vv -k http://localhost:19090/expose -d '{
	 "remote_connector_addr": "localhost:9090", 
	 "type": "local",
   "config": {
      "name": "nics-local",
      "port": "13001",
	    "address": "localhost:19090", 
    }
	}'

  # expose the public service localhost:13000 to the local machine
  # at port 13000
  curl -vv -k http://localhost:29090/expose -d '{
    "remote_connector_addr": "localhost:9090", 
    "type": "remote",
    "config": {
	    "address": "nics-local.shipyard.svc:13001", 
     }
  }'
fi

if [[ $1 == "create_auth" ]];then
	# expose the local service localhost:19090 to the public connector
	# at port 13000
	curl -vv -k http://localhost:19090/expose -d '{
	 "remote_connector_addr": "localhost:9090", 
	 "type": "local",
   "config": {
	    "address": "localhost:19090", 
    }
	}'

  # expose the public service localhost:13000 to the local machine
  # at port 13000
  curl -vv -k http://localhost:29090/expose -d '{
    "remote_connector_addr": "localhost:9090", 
    "type": "remote",
    "config": {
      "service_id": "abc123"
      "auth_token": "123sdfs-sdfdf", 
     }
  }'
fi