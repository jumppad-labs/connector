resource "network" "test" {
  subnet = "10.6.0.0/16"
}

# Named dev to match the server address in install/nomad/generate_certs.sh
resource "nomad_cluster" "dev" {
  network {
    id = resource.network.test.meta.id
  }

  # the connector job runs the locally built image
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

resource "nomad_job" "fake_service" {
  cluster = resource.nomad_cluster.dev

  paths = ["./example.nomad"]

  health_check {
    timeout = "60s"
    jobs    = ["example_1"]
  }
}
