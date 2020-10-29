container "local_connector" {
	depends_on = ["exec_local.certs"]

  image {
		name = "registry.shipyard.run/connector:dev"
  }

  command = [
    "run",
    "--grpc-bind=:9090",
    "--http-bind=:9091",
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

