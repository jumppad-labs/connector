network "local" {
  subnet = "10.5.0.0/16"
}

k8s_cluster "connector" {

  image {
    name = "registry.shipyard.run/connector:dev"
  }

  driver = "k3s"
  version = "v1.17.9-k3s1"

  network {
    name = "network.local"
  }
}

exec_local "certs" {
	depends_on = ["k8s_cluster.connector"]

	cmd = "../../install/kubernetes/generate_certs.sh"
	working_directory = "../../install/kubernetes/"
}

k8s_ingress "connector" {
  cluster     = "k8s_cluster.connector"
	namespace   = "shipyard" 
  service     = "connector"

  network {
    name = "network.local"
  }

  port {
    local  = 19090
    remote = 19090
    host   = 19090
  }

  port {
    local  = 19091
    remote = 19091
    host   = 19091
  }
}

k8s_config "connector" {
	cluster = "k8s_cluster.connector"
	paths = ["../../install/kubernetes/connector_rbac.yaml", "../../install/kubernetes/connector.yaml"]
	wait_until_ready = true
}

k8s_ingress "localconnector" {
  cluster     = "k8s_cluster.connector"
	namespace   = "shipyard" 
  service     = "localconnector"

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
