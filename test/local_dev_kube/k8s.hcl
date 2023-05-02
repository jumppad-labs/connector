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

k8s_ingress "localconnector" {
  cluster   = "k8s_cluster.connector"
  namespace = "shipyard-test"
  service   = "localconnector"

  network {
    name = "network.local"
  }

  port {
    local  = 9997
    remote = 9997
    host   = 9997
  }
}

output "KUBECONFIG" {
  value = k8s_config("connector")
}