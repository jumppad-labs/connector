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
    source      = "../../install/kubernetes/generate_certs.sh"
    destination = "/generate_certs.sh"
  }

  volume {
    source      = "./certs"
    destination = "/certs"
  }
}

k8s_config "rbac" {
  depends_on = ["k8s_cluster.connector"]

  cluster          = "k8s_cluster.connector"
  paths            = ["../../install/kubernetes/connector_rbac.yaml"]
  wait_until_ready = true
}

exec_remote "secret" {
  depends_on = ["exec_remote.certs", "k8s_config.rbac"]

  image {
    name = "shipyardrun/tools:v0.3.0"
  }

  network {
    name = "network.local"
  }

  cmd = "/create_secret.sh"

  # Mount a volume containing the config
  volume {
    source      = "../../install/kubernetes/create_secrets.sh"
    destination = "/create_secret.sh"
  }

  volume {
    source      = "./certs"
    destination = "/certs"
  }

  volume {
    source      = "${k8s_config_docker("connector")}"
    destination = "/.kube/config"
  }

  env_var = {
    "KUBECONFIG" = "/.kube/config"
  }
}