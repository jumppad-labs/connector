network "local" {
  subnet = "10.5.0.0/16"
}

nomad_cluster "dev" {
  network {
    name = "network.local"
  }

  image {
    name = "connector:dev"
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

nomad_job "fake_service" {
  cluster = "nomad_cluster.dev"

  paths = ["./example.nomad"]
  health_check {
    timeout    = "60s"
    nomad_jobs = ["example_1"]
  }
}