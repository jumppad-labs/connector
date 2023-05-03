exec_remote "certs" {
  image {
    name = "connector:dev"
  }

  network {
    name = "network.local"
  }

  cmd = "/generate_certs.sh"

  # Mount a volume containing the config
  volume {
    source      = "../../install/nomad/generate_certs.sh"
    destination = "/generate_certs.sh"
  }

  volume {
    source      = "./certs"
    destination = "/certs"
  }
}

exec_remote "create_job" {
  depends_on = ["exec_remote.certs"]

  image {
    name = "connector:dev"
  }

  network {
    name = "network.local"
  }

  cmd = "/generate_job.sh"

  # Mount a volume containing the config
  volume {
    source      = "../../install/nomad/generate_job.sh"
    destination = "/generate_job.sh"
  }

  working_directory = "/job"

  volume {
    source      = "./job"
    destination = "/job"
  }

  volume {
    source      = "./certs"
    destination = "/job/certs"
  }
}

nomad_job "connector" {
  depends_on = ["exec_remote.create_job"]
  cluster    = "nomad_cluster.dev"

  paths = ["./job/install.nomad"]
  health_check {
    timeout    = "60s"
    nomad_jobs = ["connector"]
  }
}