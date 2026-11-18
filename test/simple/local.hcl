resource "container" "local_connector" {
  depends_on = ["resource.exec.certs"]

  image {
    name = variable.connector_image
  }

  command = [
    "/connector",
    "run",
    "--grpc-bind=:9090",
    "--http-bind=:9091",
    "--log-level=trace",
    "--root-cert-path=/certs/root.cert",
    "--server-cert-path=/certs/local/leaf.cert",
    "--server-key-path=/certs/local/leaf.key",
  ]

  port_range {
    range       = "9090-9091"
    enable_host = true
  }

  port_range {
    range       = "12000-12010"
    enable_host = true
  }

  network {
    id = resource.network.local.meta.id
  }

  volume {
    source      = "./certs"
    destination = "/certs"
  }
}

resource "container" "local_service" {
  image {
    name = "nicholasjackson/fake-service:v0.14.1"
  }

  environment = {
    NAME        = "Local Service"
    LISTEN_ADDR = "0.0.0.0:9094"
  }

  port {
    local  = 9094
    remote = 9094
    host   = 9094
  }

  network {
    id = resource.network.local.meta.id
  }
}
