container "local_connector" {
	depends_on = ["exec_local.certs"]

  image {
		name = var.connector_image
  }

  command = [
    "run",
    "--grpc-bind=:9090",
    "--http-bind=:9091",
    "--log-level=debug",
    "--root-cert-path=/certs/root.cert",
    "--server-cert-path=/certs/local/leaf.cert",
    "--server-key-path=/certs/local/leaf.key",
  ]
  
  port_range {
    range = "9090-9091"
    enable_host = true
  }

  network {
    name = "network.local"
  }

  volume {
    source = "../../install/kubernetes/certs"
    destination = "/certs"
  }
}

