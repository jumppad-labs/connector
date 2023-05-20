resource "container" "local_connector" {
  image {
    name = "connector:dev"
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
    range       = "9090-9091"
    enable_host = true
  }

  network {
    id = resource.network.local.id
  }

  volume {
    source      = resource.certificate_leaf.nomad.output
    destination = "/certs"
  }
}