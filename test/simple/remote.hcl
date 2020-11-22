container "remote_connector" {
  depends_on = ["exec_local.certs"]

  image {
    name = var.connector_image
  }
  
  command = [
    "run",
    "--grpc-bind=:9092",
    "--http-bind=:9093",
    "--log-level=debug",
    "--root-cert-path=/certs/root.cert",
    "--server-cert-path=/certs/remote/leaf.cert",
    "--server-key-path=/certs/remote/leaf.key",
  ]

  port_range {
    range = "9092-9093"
    enable_host = true
  }

  port_range {
    range = "13000-13010"
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
