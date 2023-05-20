resource "certificate_ca" "root" {
  output = data("certs")
}

resource "certificate_leaf" "nomad" {
  ca_key  = resource.certificate_ca.root.private_key.filename
  ca_cert = resource.certificate_ca.root.certificate.filename

  ip_addresses = ["127.0.0.1"]

  dns_names = [
    "localhost",
    "localhost:30090",
    "30090",
    "connector",
    "connector",
  ]

  output = data("certs")
}

resource "template" "connector_job" {
  source = <<-EOF
    job "connector" {
      datacenters = ["dc1"]
      type        = "service"

      update {
        max_parallel      = 1
        min_healthy_time  = "10s"
        healthy_deadline  = "3m"
        progress_deadline = "10m"
        auto_revert       = false
        canary            = 0
      }

      migrate {
        max_parallel     = 1
        health_check     = "checks"
        min_healthy_time = "10s"
        healthy_deadline = "5m"
      }

      group "connector" {
        count = 1

        network {
          port "grpc" {
            to     = 30090
            static = 30090
          }

          port "http" {
            to     = 30091
            static = 30091
          }
        }

        restart {
          # The number of attempts to run the job within the specified interval.
          attempts = 2
          interval = "30m"
          delay    = "15s"
          mode     = "fail"
        }

        ephemeral_disk {
          size = 30
        }

        task "connector" {
          template {
            data = <<-EOH
    ${resource.certificate_leaf.nomad.certificate.contents}
            EOH

            destination = "local/certs/server.cert"
          }

          template {
            data = <<-EOH
    ${resource.certificate_leaf.nomad.private_key.contents}
            EOH

            destination = "local/certs/server.key"
          }

          template {
            data = <<-EOH
    ${resource.certificate_ca.root.certificate.contents}
            EOH

            destination = "local/certs/ca.cert"
          }

          # The "driver" parameter specifies the task driver that should be used to
          # run the task.
          driver = "docker"

          logs {
            max_files     = 2
            max_file_size = 10
          }

          env {
            NOMAD_ADDR = "http://\$${NOMAD_IP_http}:4646"
          }

          config {
            image = "connector:dev"

            ports   = ["http", "grpc"]
            command = "/connector"
            args = [
              "run",
    		      "--grpc-bind=:30090",
    		      "--http-bind=:30091",
              "--log-level=trace",
              "--root-cert-path=local/certs/ca.cert",
              "--server-cert-path=local/certs/server.cert",
              "--server-key-path=local/certs/server.key",
              "--integration=nomad",
            ]
          }

          resources {
            cpu    = 500 # 500 MHz
            memory = 256 # 256MB

          }
        }
      }
    }
  EOF

  destination = "${data("jobs")}/connector.hcl"
}

//resource "nomad_job" "connector" {
//  cluster = resource.nomad_cluster.dev.id
//
//  paths = ["${resource.template.connector_job.destination}"]
//  health_check {
//    timeout    = "60s"
//    nomad_jobs = ["connector"]
//  }
//}