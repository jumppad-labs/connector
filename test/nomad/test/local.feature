Feature: Local Connector Kubernetes
  In order to test the Remote Connector running in Kubernetes
  I should setup a remmote connector in Kubernetes
  and a local connector in Docker
  and try to access a service

Scenario: Expose multiple local and remote servics to a Kubernetes connector
  Given I have a running blueprint
  Then the following resources should be running
    | name                      | type          |
    | test                       | nomad_cluster   |
    | local_connector           | container     |
  When I run the script
    ```
    #!/bin/bash
    sleep 10
    
    echo "Expose kubernetes service to local machine"
    ## Create a service that exposes local port 9998 to the kubernetes service
    ## localservice.shipyard-test.svc:9997 
    curl -v -k https://localhost:9091/expose -d \
      '{ "name":"remoteservice",
         "source_port": 9998,
         "remote_connector_addr": "server.dev.nomad-cluster.shipyard.run:30090",
         "destination_addr": "example_1.fake_service.fake_service:http",
         "type": "remote"
       }'
    ```
  Then I expect the exit code to be 0
  When I run the script
  ```
  #!/bin/bash
  sleep 10

  ## Curl the local connector listener, this will make a connection to the 
  ## remote connector service localservice.shipyard-test.svc:9997 which in 
  ## turn is pointed at the local service localhost:9091
  curl -v http://localhost:9998
  ```
  Then I expect the exit code to be 0
