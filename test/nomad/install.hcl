resource "exec" "certs" {
  image {
    name = variable.connector_image
  }

  # the script writes to ./certs
  working_directory = "/"
  script            = file("../../install/nomad/generate_certs.sh")

  volume {
    source      = "./certs"
    destination = "/certs"
  }
}

# Writes ./job/install.nomad with the generated certs
resource "exec" "create_job" {
  depends_on = ["resource.exec.certs"]

  image {
    name = variable.connector_image
  }

  working_directory = "/job"
  script            = file("../../install/nomad/generate_job.sh")

  volume {
    source      = "./job"
    destination = "/job"
  }

  volume {
    source      = "./certs"
    destination = "/job/certs"
  }
}

resource "nomad_job" "connector" {
  depends_on = ["resource.exec.create_job"]

  cluster = resource.nomad_cluster.dev
  paths   = ["./job/install.nomad"]

  health_check {
    timeout = "60s"
    jobs    = ["connector"]
  }
}
