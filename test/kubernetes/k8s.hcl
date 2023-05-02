network "local" {
  subnet = "10.5.0.0/16"
}

k8s_cluster "connector" {

  image {
    name = var.connector_image
  }

  driver = "k3s"

  network {
    name = "network.local"
  }

  port {
    local  = 30090
    remote = 30090
    host   = 30090
  }

  port {
    local  = 30091
    remote = 30091
    host   = 30091
  }
}

output "KUBECONFIG" {
  value = k8s_config("connector")
}