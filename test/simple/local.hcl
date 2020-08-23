container "local_connector" {
  image {
    name = "shipyardrun/connector:dev"
  }

  env_var = {
    "BIND_ADDR_GRPC": "0.0.0.0:9090"
    "BIND_ADDR_HTTP": "0.0.0.0:9091"
    "LOG_LEVEL": "debug"
  }
  
  port_range {
    range = "9090-9091"
    enable_host = true
  }

  port_range {
    range = "12000-12100"
    enable_host = true
  }

  network {
    name = "network.local"
  }
}

container "local_service" {
  image {
    name = "nicholasjackson/fake-service:v0.14.1"
  }

  env_var = {
    "NAME": "Local Service"
    "LISTEN_ADDR": "0.0.0.0:9094"
  }

  port {
    local = 9094
    remote = 9094
    host = 9094
  }
  
  network {
    name = "network.local"
  }
}