Feature: Local Connector Kubernetes
  In order to test the Remote Connector running in Kubernetes
  I should setup a remmote connector in Kubernetes
  and a local connector in Docker
  and try to access a service

Scenario: Expose multiple local servics to Remote connector
  Given I have a running blueprint
  Then the following resources should be running
    | name                      | type          |
    | connector                 | k8s_cluster   |
    | local_connector           | container     |
  When I run the script
    ```
    #!/bin/bash
    sleep 10

    echo "Expose local service to remote server"
    curl -k https://localhost:9091/expose -d \
      '{ "name":"localconnector",
         "source_port": 9997,
         "remote_connector_addr": "connector.ingress.shipyard.run:19090",
         "destination_addr": "localhost:9091",
         "type": "local"
       }'
    ```
  Then I expect the exit code to be 0
  When I run the script
  ```
  #!/bin/bash
  sleep 10
  curl -k https://localhost:9997/health
  ```
  Then I expect the exit code to be 0
