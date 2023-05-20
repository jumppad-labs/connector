resource "network" "local" {
  subnet = "10.5.0.0/16"
}

resource "nomad_cluster" "dev" {
  network {
    id = resource.network.local.id
  }

  image {
    name = "connector:dev"
  }

  port {
    local  = 30090
    remote = 30090
    host   = 30090
  }
}

resource "nomad_job" "fake_service" {
  cluster = resource.nomad_cluster.dev.id

  paths = ["./example.nomad"]
  health_check {
    timeout    = "60s"
    nomad_jobs = ["example_1"]
  }
}