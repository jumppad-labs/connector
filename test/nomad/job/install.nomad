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
-----BEGIN CERTIFICATE-----
MIIGGTCCBAGgAwIBAgIRAKA7FYnmxfkNTwFWcvWaRGgwDQYJKoZIhvcNAQELBQAw
KjERMA8GA1UEChMIU2hpcHlhcmQxFTATBgNVBAMTDENvbm5lY3RvciBDQTAeFw0y
MzA1MDMwOTExNTBaFw0yNDAxMjMwOTExNTBaMCwxETAPBgNVBAoTCFNoaXB5YXJk
MRcwFQYDVQQDEw5Db25uZWN0b3IgTGVhZjCCAiIwDQYJKoZIhvcNAQEBBQADggIP
ADCCAgoCggIBAL3KobJIfa5fsHFs85ZfhgZjeomQPfZqLFpqj8ZXUf0DlWlmKxLH
xfI3Md8IUK7UhG3v8uCaeW7qe855T+7Bl8zceScoTHDnVNAFr9YInvQc4PQP/loZ
WcZLIW2GxBvF5dRWXbzpxTP7L3J0J+cB2kpX2kgzd2iKTcghTozBCPfzmfsyqxnX
apx6cmV8lpaxIzbuujvq62BYuwISE5T4vsWUfHtk5a4FZVY4HPrz200ri+jyWLDC
qq8VbubNYIc2nCjCNL45n+ITWjQxhSQU5k3RLj8WeV6k6f1hFYPoCNLc38GLQYRA
jH2uIbUPu9ux2qEwYdryaLh/4dWRkVOUcXJP9VrbV7XtR6IauNGOglNrBCAw+BL1
SwRuOis3Q0Aqm4CsG0pQSmCQBxfSDvrJmIPGuA2EVMpxK2Vx2tGa/mn7q8Wvzvl6
PoG8KZ9J3yolxDCiXml4fgSguQQi7iGkMExoANLrqVSU4o2PTeaaxipa7bptiwID
SxxYixEvJU9siZUMAav2iX2dHVBfhk+hy6rz7EDirM/ip22s9qPMZq0pvHMAo1o2
pqN3gntxig1m2nDSWX1Z/fKmykcsBiFONzzzukiKQMh0Qg6FrWhwRKhVQvsj0m/9
aBmEzs1fkHnuZn0BiI43Cgr+cIU+T1Hryz5Wg01TdmP1+7irLCNm75mTAgMBAAGj
ggE2MIIBMjAOBgNVHQ8BAf8EBAMCB4AwHQYDVR0lBBYwFAYIKwYBBQUHAwEGCCsG
AQUFBwMCMAwGA1UdEwEB/wQCMAAwHwYDVR0jBBgwFoAUIkZn8ITGFad7ps1sAmdN
h8Qdq5YwgdEGA1UdEQSByTCBxoIJbG9jYWxob3N0gg9sb2NhbGhvc3Q6MzAwOTCC
D2xvY2FsaG9zdDozMDA5MYIFOjkwOTCCBTo5MDkxggljb25uZWN0b3KCK3NlcnZl
ci5kZXYubm9tYWQtY2x1c3Rlci5zaGlweWFyZC5ydW46MzAwOTCCK3NlcnZlci5k
ZXYubm9tYWQtY2x1c3Rlci5zaGlweWFyZC5ydW46MzAwOTGCDmNvbm5lY3Rvcjo5
MDkwgg5jb25uZWN0b3I6OTA5MYcEfwAAATANBgkqhkiG9w0BAQsFAAOCAgEAV7Ki
jJ74QHmfCrwEZ7d5+NekcpEMLQJb5MNPHb223H2jXTS4LCNDNxY+srOBmtAB4OlM
YfimLemgJl3VT350cWFmqVNDXs4jNMdxb7xcfmxeA3mOzw3qRe0BBtFF1Lpf+duB
qD72QTzey/W50pGScmwpUHqVfbkd8302Rxd++L9SIjctx380Gl1uknW0DsgB5dcv
c6QahNYcqLqwG6mG5mYTyF7cTWTJLQ9RvOrDiDY5vBL4FIQfXSakRESn9TtN8HvJ
PBXU0Yn9VFg5Ywf3QxA2Z5hZUr0lwVtE+/NJvFE8Ae2XoQ/patlUaeZVPQAOPUpU
AmqC8dR1VW9j33sZSL39fM3+HNowfe+/H4bTL8zAPKpIeSRLwGNslJOsP75inytb
gAEszPPM9Ajs4GaRkwWIcAdfOWPd7Tbg+yJqOPJcFn9+3EPmz+87mG+CvCty79iT
4f4rdd5tRIOXuqYk5h4XhKNLyb9oTSX26IytBEZugNPV8HpO+QFaxuV5RwbOVK56
gNjZYUZRlttL/AIku459F8YFJ5egLw+HlMKAL/rP1vYPeQKnMHOXz3KXuk3iMVGT
im3S64jq8ZRUifEmK+NBBDkGwNFJAGiFZorMoTA+UvcMRkedsBlpxF0zhBtOftJz
dbKGwzdNM7xubALrg6ReOQUV+F7WPHsG+oi5sPc=
-----END CERTIFICATE-----
        EOH

        destination = "local/certs/server.cert"
      }
      
      template {
        data = <<-EOH
-----BEGIN RSA PRIVATE KEY-----
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
-----END RSA PRIVATE KEY-----
        EOH

        destination = "local/certs/server.key"
      }
      
      template {
        data = <<-EOH
-----BEGIN CERTIFICATE-----
MIIFITCCAwmgAwIBAgIRAKXlzPw/F9Vh3N4pcx2WwaowDQYJKoZIhvcNAQELBQAw
KjERMA8GA1UEChMIU2hpcHlhcmQxFTATBgNVBAMTDENvbm5lY3RvciBDQTAeFw0y
MzA1MDMwOTExNDlaFw0yNDAxMjMwOTExNDlaMCoxETAPBgNVBAoTCFNoaXB5YXJk
MRUwEwYDVQQDEwxDb25uZWN0b3IgQ0EwggIiMA0GCSqGSIb3DQEBAQUAA4ICDwAw
ggIKAoICAQDF3Q6NhSKfZbzJcRV//5UcZeJv2FZW0+PLP7wOvB/uC0la60F8gjxI
NDqRITjihcFhXvAjDY9LgpbGNyMvtEV41mj74WO9v+pqxuSa9XWVUjf1MYktLnQ6
iRrCLPfLDGucFz2oJecRurljK0xsCTefdFL+pSpxcWY3nV2KoKA6jnUP6f3dsbon
1XGFvX6W22OoP5fUTMcwWO7MHickxI8thGaAPv0aEV6m6qbJzVlCrndZUyzKnU4t
xllJ8boXB8XO7DUJwJpkh5TBv1AbdRu6DKtMIW9iobuMBCtSEYAf7AVjd0I4ZPU5
fQjHNqu9xfQGLGj5NnyijXD00/GGoIbPUtQUw3mjyZ4WEEletfBoJ2LdB6+4+Dmx
jSP2Ea7+5bKpCE4dF91o0gjByQUWFiORYrqW6jbmiSqHss3JMPuEjaqHWBFdVtE0
03kru1Pxzra8eelx4pBW3R4yyHjaOamz5WMS9rz23YZlvu3oItNmDiNpaqd4Fzv+
DXswJnc5DW/C9C8QSpTUXrZ36bmSnK2x6HgGtfL0TTBMPx/UrvpRK3eQ9Y4pXTDU
L8VMEUuXZqwNxom72r6wrcmf4QoCb+3swBcHzDeuN4bQ4XDlcgAv1DKuWq04BrKT
wLhqxW1f9NV2bscaHBp8hqtnYjRjrbNWgRqtZqnt142lGd9Gr09qaQIDAQABo0Iw
QDAOBgNVHQ8BAf8EBAMCAoQwDwYDVR0TAQH/BAUwAwEB/zAdBgNVHQ4EFgQUIkZn
8ITGFad7ps1sAmdNh8Qdq5YwDQYJKoZIhvcNAQELBQADggIBAD9uujXZxGNqzUnI
Gw0QRDnybNjhEFHkfNi9gFtU7uq4VPpsy7gKsf01Q4UGZni0Ia23GA4CTfZKI2BC
naJ09YFEGnrozdtFBb5MOI0/tKvYpGTeENLmzRJv6hsD58eiBwGrK3PONkMTTxOg
i3eWP0y1PGzgXw9TYs6wJCSPybjXwTr73N3wB7x/cSwlJuJ/HgqS7CLlciffIKYJ
qnexHcTyakv/2+yMtIGkNTEs/1eJZjey4c39Ex5MMoy1cj2BZfINI5aJ12kbAuxf
rII8VeJGrVB/268WpWQS+2Gu7HVS34XzWPhtFo4MXExIL/9Px5U0djhPfgb2DJtm
WDLr0LWJ7X5E8S3QNwwve8ob4fPAio8JfbosULJVEV7dsHyXGHHrBGDYmvX1z9gp
Apiq8B4l3p+gWPzDCQMq3hIapdhKrXc0+oF1YUVoKx5zczjOY0dSdukL9RJpPqRq
PCzbf4qSnYA64dfNw9gDGiuTi1G2tRrXif696QlTkc7uuXeR7PAr5NkMq3X6Qi2f
uN5ChQjo9F5vJkHxL9XZYShVr6XVLJ6Ha3weudbH+QFhUaL8RSC0R4VaBfS0QcNw
vv7ciNZjUmTPTzUSjNdLp8FRg3b4iRELplE2JDz8LVgM3RYg2CxG1tOdzN7ntSxS
z4SIQOhYnAdiwZ8y7f6055fpD9uJ
-----END CERTIFICATE-----
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
        NOMAD_ADDR = "http://${NOMAD_IP_http}:4646"
      }

      config {
        image = "connector:dev"

        ports   = ["http", "grpc"]
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
