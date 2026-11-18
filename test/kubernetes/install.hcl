resource "exec" "certs" {
  image {
    name = variable.connector_image
  }

  # the script writes to ./certs
  working_directory = "/"
  script            = file("../../install/kubernetes/generate_certs.sh")

  volume {
    source      = "./certs"
    destination = "/certs"
  }
}

resource "k8s_config" "rbac" {
  cluster          = resource.k8s_cluster.connector
  paths            = ["../../install/kubernetes/connector_rbac.yaml"]
  wait_until_ready = true
}

# Creates the TLS secrets with kubectl from the k3s image, using a kubeconfig
# that reaches the API server over the Docker network
resource "exec" "secret" {
  depends_on = ["resource.exec.certs", "resource.k8s_config.rbac"]

  image {
    name = "ghcr.io/jumppad-labs/kubernetes:v1.36.2"
  }

  network {
    id = resource.network.local.meta.id
  }

  working_directory = "/"

  script = <<-EOF
    #!/bin/sh
    set -e

    cat > /tmp/kubeconfig <<EOK
    apiVersion: v1
    kind: Config
    clusters:
    - name: connector
      cluster:
        server: $K8S_SERVER
        certificate-authority-data: $K8S_CA
    users:
    - name: admin
      user:
        client-certificate-data: $K8S_CERT
        client-key-data: $K8S_KEY
    contexts:
    - name: connector
      context:
        cluster: connector
        user: admin
    current-context: connector
    EOK

    KUBECONFIG=/tmp/kubeconfig sh /create_secrets.sh
  EOF

  volume {
    source      = "../../install/kubernetes/create_secrets.sh"
    destination = "/create_secrets.sh"
  }

  volume {
    source      = "./certs"
    destination = "/certs"
  }

  environment = {
    K8S_SERVER = "https://${resource.k8s_cluster.connector.container_name}:${resource.k8s_cluster.connector.api_port}"
    K8S_CA     = resource.k8s_cluster.connector.kube_config.ca
    K8S_CERT   = resource.k8s_cluster.connector.kube_config.client_certificate
    K8S_KEY    = resource.k8s_cluster.connector.kube_config.client_key
  }
}

resource "k8s_config" "connector" {
  depends_on = ["resource.exec.secret"]

  cluster          = resource.k8s_cluster.connector
  paths            = ["../../install/kubernetes/connector.yaml"]
  wait_until_ready = true
}
