exec_remote "certs" {
  image {
    name = var.connector_image
  }
  
  network {
    name = "network.local"
  }

  cmd = "/generate_certs.sh"

  # Mount a volume containing the config
  volume {
    source = "./generate_certs.sh"
    destination = "/generate_certs.sh"
  }
  
  volume {
    source = "./certs"
    destination = "/certs"
  }
}