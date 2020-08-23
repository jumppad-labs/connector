container "remote_connector" {
  image {
    name = "shipyardrun/connector:dev"
  }

  env_var = {
    "BIND_ADDR_GRPC": "0.0.0.0:9092"
    "BIND_ADDR_HTTP": "0.0.0.0:9093"
    "LOG_LEVEL": "debug"
  }

  port_range {
    range = "9092-9093"
    enable_host = true
  }

  port_range {
    range = "13000-13100"
    enable_host = true
  }
  
  network {
    name = "network.local"
  }
}

container "remote_service" {
  image {
    name = "nicholasjackson/fake-service:v0.14.1"
  }

  env_var = {
    "NAME": "Remote Service"
    "LISTEN_ADDR": "0.0.0.0:9095"
  }

  port {
    local = 9095
    remote = 9095
    host = 9095
  }
  
  network {
    name = "network.local"
  }
}