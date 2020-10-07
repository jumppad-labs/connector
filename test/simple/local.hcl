container "local_connector" {
  image {
    name = "gcr.io/shipyard-287511/connector:latest"
  }

  command = [
    "run",
    "--grpc-bind=:9090",
    "--http-bind=:9091",
    "--root-cert-path=/certs/root.cert",
    "--server-cert-path=/certs/leaf.cert",
    "--server-key-path=/certs/leaf.key",
  ]
  
  port_range {
    range = "9090-9091"
    enable_host = true
  }

  port_range {
    range = "12000-12010"
    enable_host = true
  }

  network {
    name = "network.local"
  }

  volume {
    source = "./certs"
    destination = "/certs"
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