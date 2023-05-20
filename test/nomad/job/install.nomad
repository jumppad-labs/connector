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
MIIGGTCCBAGgAwIBAgIRAL39HBPSPY42Ut2IL8hnSM8wDQYJKoZIhvcNAQELBQAw
KjERMA8GA1UEChMIU2hpcHlhcmQxFTATBgNVBAMTDENvbm5lY3RvciBDQTAeFw0y
MzA1MDMxMDQ0MzNaFw0yNDAxMjMxMDQ0MzNaMCwxETAPBgNVBAoTCFNoaXB5YXJk
MRcwFQYDVQQDEw5Db25uZWN0b3IgTGVhZjCCAiIwDQYJKoZIhvcNAQEBBQADggIP
ADCCAgoCggIBAOxBJpW2KAfWGdxR0qGK7OT4PPDJUSGcZ74kmEpGLS3INqQgYBp6
4McQPEa62eGVp8XufqDLDpulYJwQu+AjxL0M+5udJ87GWgbVdEgwUZqXPOtsiWqe
UU4ULs8dlZj5i8Hpxv4Xyp5h7U+a9X8khBJNvZZpJpA0R0Yxnd8Q/Wliah/kopXA
P97S07iRyxsSqRZdy9XEnwL4gbMlG4F8fgcmwKBKT894iHz9Zfb3MdM/fmkgje6z
N2Pa7PawjFJPGSGNo1p3xJEPcjr2XSkMmIRRXdtahRuzEjPVTfOaIqgtG7t9HfmB
3vsW14mvhMx3hzle4DBnYVr3RD9zCUYq7rUMgfDnVFx4xdc5ycv8G3I1UpNu5jLj
tEIG43t5r+auUUcAjAvi2jDISEMxGiciXd/76E5kNRAcP6qziO/bCjN8VEk0Hy6H
S+oRxmL9rZgbqT1CpQwrubi/b+TQDIS3VPCVcRb33T69EQCQg7roJb/EUrYznLiW
VO9gz3B4YnBF5urCXqwFx39m+5lVrssxoDj98xC21ujwzEHB545bGIvnmBiXrpJj
//3xaxCsCSm2qGiGrPaowOBpbvydcR4YZY3oeviSC1JFlkMjqOav2B5STUlaxpkB
ppXm36LTwJs8f4RSW0htSihgNg1XwM4oYtbm9uW250PLanme7wECZGsLAgMBAAGj
ggE2MIIBMjAOBgNVHQ8BAf8EBAMCB4AwHQYDVR0lBBYwFAYIKwYBBQUHAwEGCCsG
AQUFBwMCMAwGA1UdEwEB/wQCMAAwHwYDVR0jBBgwFoAUy5maAzN7gC0FVC2+fzo/
gkaTAX4wgdEGA1UdEQSByTCBxoIJbG9jYWxob3N0gg9sb2NhbGhvc3Q6MzAwOTCC
D2xvY2FsaG9zdDozMDA5MYIFOjkwOTCCBTo5MDkxggljb25uZWN0b3KCK3NlcnZl
ci5kZXYubm9tYWQtY2x1c3Rlci5zaGlweWFyZC5ydW46MzAwOTCCK3NlcnZlci5k
ZXYubm9tYWQtY2x1c3Rlci5zaGlweWFyZC5ydW46MzAwOTGCDmNvbm5lY3Rvcjo5
MDkwgg5jb25uZWN0b3I6OTA5MYcEfwAAATANBgkqhkiG9w0BAQsFAAOCAgEAJOv4
s0tRXMFCvcfsXy6wpXrQTd3KM6oCdL8b6fRkP15Z1Cl5UpxGJo4KZWYXZDHYrUkN
047DMasG+r5MiegJ215fhXqZ8MP5dcsuEcfcuFOO/RX+seG/++8KAwvnMJWWNfdW
Dh8gm3n8mXaZHvPQIZm3hho2h8gq7OIqWTlk9InoJEDJKLFeocgzozCXNYdyo3To
ehePxV2oornssAuTzuH6Wm2RG6PKNWgwpMKIx11E5F4kst7uDpnU0qyQJXZwdKf8
cXZJlLigWvb0MEwnaF1owX6AUqmLQYO/5pkP6cx9Lau8MngzrUjmiI8VMIjyP/AP
k+3lFoe4nesv4mZLLQJXZCq84aGbWtZADifLf75a64HPIkDLGRKEnNfTrvOpPjZY
Hr2I9W5cWdckMoCwT/mO85iIbkLk5t83s6FCne7XbE3gZ8PYSuZNaxOfIxT5ynaG
y08ytXjCGupgUTQJp9lJm4JffJT3NuSziy81La/wa4H6mRvYJR7GveH4ZDI0mitg
MJH0d/GQJ60Yps+ESQ9iDA+KlbKVvy/ARaaB6M/oP6JNrz7zkTesE+beDAoq167/
LslX4/BM+kgCvAgdW3TZx6L83SPF1GKBab9tos3zBKA4+36y7qyCaEgju3MvHWzB
ZMKB2S7van3Z1GEzu+IU8gNR7f3bUjiXAschKdU=
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
MIIFITCCAwmgAwIBAgIRAIJjkbPEroauxH5te5L+KD4wDQYJKoZIhvcNAQELBQAw
KjERMA8GA1UEChMIU2hpcHlhcmQxFTATBgNVBAMTDENvbm5lY3RvciBDQTAeFw0y
MzA1MDMxMDQ0MzJaFw0yNDAxMjMxMDQ0MzJaMCoxETAPBgNVBAoTCFNoaXB5YXJk
MRUwEwYDVQQDEwxDb25uZWN0b3IgQ0EwggIiMA0GCSqGSIb3DQEBAQUAA4ICDwAw
ggIKAoICAQDksnAxLi9hnZmiczTYkOyL+81O24qwa1UUFb6yidvldgE/n6YpwLYv
f5/BLlIT10jwq3PVnWL670jjrbnl+dh2F/Jbw4ghMTdnA2J1v6gHbP1wx8ooK0l0
Z3SsjzGQKEpYUVF0k7kwQSq7IOdTgekqgeooPZOLp0BfDatCgX8KoJYy+ChLigss
5TRJP/gsdhPy993yysH0fm35N9AJnRJhxbnABn838diddRpKo533f1uo15aJnqEk
pKp7xWY8YjKW8Qm/fKN0abOdQe3CIHpNBOwznXUb9yT1INZMxCITA5+gjJjhx5Dl
B/3Em4fhoobwlUu3IZoyq82l+Sq4oQOil9JdrhWhqLiIQY03qhLGVzIo/H606uPs
jk44hfRKW5EepUWSGWNvoJbFu6gaMytCq7BX/VoK+lN6D+8gTsUfaayeysp20mRD
nRZlz82EGOY5sNktuyajXQvlWmaDd/EvZS4vF34lUXW8U9xbElkLjvdgmL7ppB2h
J5/EU40rimfUQbyhHg2hyLPmKPOyZYJuK76MfpRNKWmG7OCkHbPILWj0hQvzTfwX
6HCyvmNYOG1EVnSDFB+HmpH3RzhQZHZZwt93KwlYzWAbXHz/qIiBW5haQEAMMZ6e
hIRezpTZflb/9j6N2I599qrRrrRSKx+T3JY5RB3Wqav+0qJ8wsLYAQIDAQABo0Iw
QDAOBgNVHQ8BAf8EBAMCAoQwDwYDVR0TAQH/BAUwAwEB/zAdBgNVHQ4EFgQUy5ma
AzN7gC0FVC2+fzo/gkaTAX4wDQYJKoZIhvcNAQELBQADggIBANnV8gfZ+v6eM2Vq
JODMA18iWM4bEcKWRTZvxwfxp6n+oXjUR5TsdLyO+FVkEc4u0835OGI19ptCcZLj
sRSDhdWANFhKMZcf77vFwVrObeOwI1Q2ORHFXc6/wdb3qsVPPk3zPDc3zqpkN94h
Q4P7Oi+qLeLfIX0W/Ce2J+myuIws5emEXx+zlNrm/xEI+F5IUOwbZNdCYkPNEnuu
KEG8uYcShHkaRMs+nGHzsnJrhJsdDSrerRGt5lnNySAbxO4J5WIa+DJiyIKpZvHr
nl2ULTwjNpRKV+L94T8Yd7W1sXcBLpilfnG0K/MbJpX662Hh1wiYIMh9iS8rxrhi
zTkMjY81KX08SmKahzHFTm74qYdenQHu7b5xkGzCvAdB+izwIiYkXdyNstp6EVSs
1splQ1WtP3nYxokrKWz9tzzLN5WJMpaLBzLO4Lxclxn9Ud1U3CoUYLXacICVXu5F
7hvJVUVuSlebPuUiHdGz6qT29JzbkQBhF/h/VGV1VD9Bw/VbKALbkNdHICAvpsCd
PEFrRC+4nDyD5UB6IBKL4aU3V48fGXW3QpRvCzrCMUle1rO2QBuSFdNRjrOZWgfn
dj68INZKDBUWDwRBGFNNUiEerYLRq4A6V7kf43K5AzyjIVIfnbPFibRhUFC7Hth6
qmXNYqI/1HACsHzTQhCeq4j0cPpJ
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
        command = "/connector"
        args = [
          "run",
          "--grpc-bind=:30090",
          "--http-bind=:30091",
          "--log-level=trace",
          "--ca-path=local/certs/ca.cert",
          "--cert-path=local/certs/server.cert",
          "--key-path=local/certs/server.key",
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
