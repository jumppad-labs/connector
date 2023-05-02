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
}

output "KUBECONFIG" {
  value = k8s_config("connector")
}