# connector
Client and Server packages for Shipyard connector which allow local applications to be integrated into a remote application stack

```
remote_connector "consul" {
  address = "http://remote.mydomain.com:1244"
}
```

```
local_service "fake" {
  target = "remote_connector.consul"

  name = "fake-service"
  port = 9090

  match {
    http {
      header = [
        {
          name  = "x-developer"
          exact = "1"
        },
      ]
    }
  }
}
```