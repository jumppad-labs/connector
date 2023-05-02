# Connector
![Build](https://github.com/jumppad-labs/connector/workflows/Build/badge.svg)
[![codecov](https://codecov.io/gh/jumppad-labs/connector/branch/master/graph/badge.svg)](https://codecov.io/gh/jumppad-labs/connector)

Connector allows you to expose local TCP sockets to remote machines, and access TCP sockets running on remote machines locally. It works by tunneling the TCP connection over gRPC between two servers. Connector was build to be used with Shipyard but will work standalone to allow remote services access local applications. One use for this is when you are developing a local service and would like to connect it to a larger microservice environment which may be running in a remote Kubernetes cluster.

## Running Connector
Connector is a single binary and can be run with the following command.

```shell
./connector run [flags]
```

To set the bind address, configure TLS, or set the log level the following flags can be set.

```
Flags:
  -h, --help                      help for run
      --grpc-bind string          Bind address for the gRPC API (default ":9090")
      --http-bind string          Bind address for the HTTP API (default ":9091")
      --log-level string          Log output level [debug, trace, info] (default "info")
      --root-cert-path string     Path for the PEM encoded TLS root certificate
      --server-cert-path string   Path for the servers PEM encoded TLS certificate
      --server-key-path string    Path for the servers PEM encoded Private Key 
 ```

## Exposing local services to remote hosts
In the following example a remote machine running on the public internet can access a local TCP socket on a machine inside a private network. 

1. The local machine makes a call to the remote machine requesting to expose a local socket
1. A gRPC bi-directional stream is opened from the local machine to a publicly accessible remote server. Due to NAT issues, the local machine is not directly accessible from the remote machine. Opening an outward connection from the private to the public server bypasses this problem similar to a reverse SSH tunnel. 
1. The remote machine opens a local TCP Socket which can accept traffic.
1. When a connection is received by the Remote socket, the traffic is transparently proxied over the gRPC stream to the local machine.
1. The local machine sends the traffic to the final destination.

![](./images/arch.png)

### Example
For example given you have the following setup:
* Connector running locally (127.0.0.1)
* Dev service running locally (127.0.0.1:9090)
* Remote Connector running on a publicly accessible server (82.42.12.21)

To expose the local dev service a POST request would be made to the local connectors JSON endpoint:
```
curl localhost:9091/expose -d \
  '{
    "name":"devservice", 
    "source_port": 9090, 
    "remote_connector_addr": "82.42.12.21:9092", 
    "destination_addr": "localhost:9090",
    "type": "local"
  }'
```

## Exposing remote service locally 
It is also possible for Connector to expose TCP sockets running on a remote machine to the local host. This works in the same way as exposing remote services, the connection is always opened outward from the local machine to avoid NAT problems.

To expose a service which is accessible from the remote connector a POST request can be made to the local connectors JSON endpoint:
```
curl localhost:9091/expose -d \
  '{
    "name":"remoteservice", 
    "source_port": 9091, 
    "remote_connector_addr": "82.42.12.21:9092", 
    "destination_addr": "localhost:8080",
    "type": "remote"
  }'
```

This would make the server which is accessible from the remote connector at `localhost:8080` available to the local connector at `localhost:9091`.

## Securing Connector

The Communication between Connector services can be secured using mTLS using the following startup flags.

```
      --root-cert-path string     Path for the PEM encoded TLS root certificate
      --server-cert-path string   Path for the servers PEM encoded TLS certificate
      --server-key-path string    Path for the servers PEM encoded Private Key 
```

Each server has its own x509 certificate which is used to enable TLS for the server but also used by the outbound HTTP client when connecting to another Connector. When a connection is received by an upstream Connector it validates that the connecting clients certificate is a descendant of the configured Root cert. If no certificate is presented, or if the signature is not valid then the upstream will not permit the connection. [Mutual authentication](https://en.wikipedia.org/wiki/Mutual_authentication), or mTLS is a simple and effective way to control access between two servers.

### Configuring mTLS certificates

To simplify the process of generating mTLS certificates the Connector binary has the `generate-certs` command which enables the generation of self signed x509 certificates.

```shell
➜ connector generate-certs --help
Allows you to generate a TLS root and leaf certificates for securing connector communication

Usage:
  connector generate-certs [output location] [flags]

Flags:
  -h, --help                 help for generate-certs
      --ca                   Generate a CA x509 certificate and private key
      --dns-name strings     DNS name to add to leaf certificate
      --ip-address strings   IP address to add to the leaf certificate
      --leaf                 Generate a leaf c509 certificate and private key
      --root-ca string       CA cert to use for generating the leaf certificate
      --root-key string      Root key to use for generating the leaf certificate
```

To secure two servers you need three certificates:
* Root certificate which has the CA role that can be used to sign child certificates
* Leaf certificate and private key for the first server signed by the CA 
* Leaf certificate and private key for the second server signed by the same CA used to sign the first servers certificate

#### Generating the root cert

To generate a root certificate you can use the following command:

```
mkdir ./certs
connector generate-certs --ca ./certs
```

This will output two files into the `./certs` folder, `root.cert`, and `root.key`. The `root.cert` file will be installed on both servers, the `root.key` file is only required to generate leaf certificates and should be kept in a safe location.

#### Generating a leaf certificate for server 1

Once the root cert has been generated you can now go ahead and generate a leaf certificate for server 1.

```shell
mkdir ./certs/server1

connector generate-certs \
          --leaf \
          --ip-address 127.0.0.1 \
          --dns-name "localhost" \
          --dns-name "localhost:9090" \
          --dns-name "localhost:9091" \
          --dns-name "server1" \
          --dns-name "server1:9090" \
          --dns-name "server1:9091" \
          --root-ca ./certs/root.cert \
          --root-key ./certs/root.key \
          ./certs/server1
```

This command will generate two files in the folder `./certs/server1`, `leaf.cert`, and `leaf.key`. The flags `ip-address`, and `dns-name` allow the configuration for valid  names for the certificate. For example, if you used this certificate for `server2` then when a connection is made to the server the client validate of the x509 certificate would fail. This is due to the certificate not containing the dnsname `server2`.  


#### Generating a leaf certificate for server 2

Using the same command you can generate certificates for `server2`:

```shell
mkdir ./certs/server2

connector generate-certs \
          --leaf \
          --ip-address 127.0.0.1 \
          --dns-name "localhost" \
          --dns-name "localhost:9092" \
          --dns-name "localhost:9093" \
          --dns-name "server2" \
          --dns-name "server2:9092" \
          --dns-name "server2:9093" \
          --root-ca ./certs/root.cert \
          --root-key ./certs/root.key \
          ./certs/server2
```

Because both of these certificates share a common root `./certs/root.cert`, they will be valid for securing the connection with mTLS, if a different root CA and private key was use to generate the second certificate then when `server2` attempted to connect to `server1`, `server1` would reject the connection. 

#### Running servers with mTLS

To run a Connector using mTLS you need to set all three of the certificate related flags:

```shell
      --root-cert-path string     Path for the PEM encoded TLS root certificate
      --server-cert-path string   Path for the servers PEM encoded TLS certificate
      --server-key-path string    Path for the servers PEM encoded Private Key 
```

The following example shows how you could start two Connectors using the newly created certificates


##### Server 1
```shell
connector run \
          --grpc-bind ":9090" \
          --http-bind ":9091" \
          --root-cert-path "./certs/root.cert" \
          --server-cert-path "./certs/server1/leaf.cert" \
          --server-key-path "./certs/server1/leaf.key"
```

##### Server 2
```shell
connector run \
          --grpc-bind ":9092" \
          --http-bind ":9093" \
          --root-cert-path "./certs/root.cert" \
          --server-cert-path "./certs/server2/leaf.cert" \
          --server-key-path "./certs/server2/leaf.key"
```

With both servers running you can could create a connection between them with the following command, this is just a simple command which exposed the HTTP API of server 2 at port `9099`.

```
curl -k https://localhost:9091/expose -d \
  '{
    "name":"devservice", 
    "source_port": 9099, 
    "remote_connector_addr": "localhost:9092", 
    "destination_addr": "localhost:9093",
    "type": "local"
  }'
```

You can then test the connector using the following command:

```
curl -k https://localhost:9099/health
```

You will see in the logs the connection is received by `server1` and it is proxied to `server2`, `server2` then sends the connection to the final destination. The upstream server in this example could have been any service which was accessible from the remote connector.

## Restful API
Connector uses a gRPC API however for convenience there is also a partial RESTful API.

### POST /expose
The expose endpoint allows you establish new connections between local and remote servers.

#### Parameters

**name**  
**type**: string

The name parameter is a human readable name for the exposed service.

**source_port**  
**type**: int

The port where the service will be accessible. If the service type is "local", this port will be a listener on the remote connector as it is exposing a local service. If the service type is "remote", this port will be a listener on the local connector as it is exposing a remote service.

**remote_connector_addr**  
**type**: string

The address of the remote connectors gRPC API

**destination_addr**  
**type**: string

FQDN of the exposed service, this address is used by the terminating Connector to send the traffic to the destination. E.g. localhost or Kubernetes service name.

**type**
**type** string [local, remote]

#### Returns
String GUID for the created connection

Type specifies the direction of the traffic. A value of `local`, exposes a service on the local machine to the remote connector. A value of `remote` exposes a service on the remote machine to the local connector.

### DELETE /expose/{id}

Delete the exposed service with the given id

### GET /health
Return the health of the Connector.

### GET /list
Return a list of configured services

```
[
  {
    "id": "",
    "name": "test",
    "source_port": 12000,
    "remote_connector_addr": "remote-connector.container.shipyard.run:9092",
    "destination_addr": "remote-service.container.shipyard.run:9095",
    "type": "REMOTE",
    "status": "COMPLETE"
  },
  {
    "id": "",
    "name": "test1",
    "source_port": 13000,
    "remote_connector_addr": "remote-connector.container.shipyard.run:9092",
    "destination_addr": "local-service.container.shipyard.run:9094",
    "type": "LOCAL",
    "status": "COMPLETE"
  }
]
```

## Testing
A simple test suite can be found in the folder `./test/simple`. These tests set up a pair of servers and test a local service exposed to a remote connector and a remote service exposed to a local connector. You can execute the tests using [Shipyard](https://shipyard.run):

```
shipyard test .
Feature: Remote Connector Simple
  In order to test the Remote Connector
  I should setup a remote and a local
  and try to access a service

  Scenario: Expose Local Service to Remote Server                           # /home/nicj/go/src/github.com/jumppad-labs/connector/test/simple/test/connector.feature:6
    Given I have a running blueprint                                        # test.go:181 -> *CucumberRunner
    Then the following resources should be running                          # test.go:262 -> *CucumberRunner
      | name             | type      |
      | local_connector  | container |
      | remote_connector | container |
      | local_service    | container |
    When I run the script                                                   # test.go:479 -> *CucumberRunner
      ```
      #!/bin/bash
      echo "Expose local service to remote server"
      curl localhost:9091/expose -d \
        '{
          "name":"test", 
          "local_port": 9094, 
          "remote_port": 13000, 
          "remote_server_addr": "remote-connector.container.shipyard.run:9092", 
          "service_addr": "local-service.container.shipyard.run:9094",
          "type": "local"
        }'
      ```
    Then I expect the exit code to be 0                                     # test.go:517 -> *CucumberRunner
    And a HTTP call to "http://localhost:13000" should result in status 200 # test.go:350 -> *CucumberRunner


  Scenario: Expose Remote Service to Localhost                              # /home/nicj/go/src/github.com/jumppad-labs/connector/test/simple/test/connector.feature:30
    Given I have a running blueprint                                        # test.go:181 -> *CucumberRunner
    Then the following resources should be running                          # test.go:262 -> *CucumberRunner
      | name             | type      |
      | local_connector  | container |
      | remote_connector | container |
      | remote_service   | container |
    When I run the script                                                   # test.go:479 -> *CucumberRunner
      ```
      #!/bin/bash
      echo "Expose local service to remote server"
      curl localhost:9091/expose -d \
        '{
          "name":"test", 
          "local_port": 12000, 
          "remote_port": 9095, 
          "remote_server_addr": "remote-connector.container.shipyard.run:9092", 
          "service_addr": "remote-service.container.shipyard.run:9095",
          "type": "remote"
        }'
      ```
    Then I expect the exit code to be 0                                     # test.go:517 -> *CucumberRunner
    And a HTTP call to "http://localhost:12000" should result in status 200 # test.go:350 -> *CucumberRunner


2 scenarios (2 passed)
10 steps (10 passed)
39.4207831s
```
