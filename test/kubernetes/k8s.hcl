resource "network" "local" {
  subnet = "10.5.0.0/16"
}

resource "k8s_cluster" "connector" {
  network {
    id = resource.network.local.meta.id
  }

  # the connector deployment runs the locally built image
  copy_image {
    name = variable.connector_image
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
  value = resource.k8s_cluster.connector.kube_config.path
}
