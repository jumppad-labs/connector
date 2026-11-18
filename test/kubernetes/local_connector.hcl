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
    "--log-level=debug",
    "--root-cert-path=/certs/root.cert",
    "--server-cert-path=/certs/local/leaf.cert",
    "--server-key-path=/certs/local/leaf.key",
  ]

  port_range {
    range       = "9090-9100"
    enable_host = true
  }

  port_range {
    range       = "9990-9999"
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
