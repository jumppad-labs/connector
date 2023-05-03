#!/bin/sh

server_cert=$(cat ./certs/nomad/leaf.cert)
server_key=$(cat ./certs/nomad/leaf.key)
server_ca=$(cat ./certs/root.cert)

cat << EOF > install.nomad
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
${server_cert}
        EOH

        destination = "local/certs/server.cert"
      }
      
      template {
        data = <<-EOH
${server_key}
        EOH

        destination = "local/certs/server.key"
      }
      
      template {
        data = <<-EOH
${server_ca}
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
        NOMAD_ADDR = "http://\${NOMAD_IP_http}:4646"
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