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
MIIGGDCCBACgAwIBAgIQKdAx+aIrWbtJfr3t/UeasTANBgkqhkiG9w0BAQsFADAq
MREwDwYDVQQKEwhTaGlweWFyZDEVMBMGA1UEAxMMQ29ubmVjdG9yIENBMB4XDTIz
MDUwMzEwMDEyMFoXDTI0MDEyMzEwMDEyMFowLDERMA8GA1UEChMIU2hpcHlhcmQx
FzAVBgNVBAMTDkNvbm5lY3RvciBMZWFmMIICIjANBgkqhkiG9w0BAQEFAAOCAg8A
MIICCgKCAgEAp/FRfHkm5tW6Ka4UfC+juMklYiohBlpHBqTHfhBHvJiCaTpB8c9g
d9Sn5CvAfZPTjN4ps5kNCPiGPchn6vGXwBGfkmQAg44sIbEbjsxhgeuur0pe5VKV
cgmhBb0QJIzbLTpKGTNIRSAK3dKbRk0+vjYG+Ov4Le2sfCJHdtF4swJY4G99nlxh
AX+PkWyqWt7Pjbgq5lZYmlPRPHMNJDa2IZOWyrEDLB4MTJzyHq+a7RmmBP381dOZ
m0y3b3lWjrEcCr1w6lFX2y4sOuoIVgomiOahDVhJeYE3x6UJoo4LS1fVKmB45PnY
Qj3AE0oUKKJcEpJSmNjIdeRsixbqUvpW0kjqK6Sccrl1P7MM6y4daQjT6LpvkCkP
/LQfcBSSUfRivI+OeiTmDQ0r1fLDSAbTMmAL8+Q1ZbGSCOGglqOVd5bSsvIPhXTt
ZWy7rkROvZ9tBZpH5qGsUZ4Eo2DwJAdxT+pUaRSx8HQN+jhr1rljuzfmv1x4PEPk
STJdSJTneq5k7lkOI5OU5mXk1LZQP1SShMHok4mEjGHiXM8llToRKmBybU7gnZpd
fANwDYBQSilv4S/nH1RwTy76naJewO80xtwucuvqHKZDiTPx0X5/yXzlgxff/n98
5lXilCv0vUngTqUfBqN3d/dWyHTvUZUNCJo+8QdVb8YLugNl0xOLFTMCAwEAAaOC
ATYwggEyMA4GA1UdDwEB/wQEAwIHgDAdBgNVHSUEFjAUBggrBgEFBQcDAQYIKwYB
BQUHAwIwDAYDVR0TAQH/BAIwADAfBgNVHSMEGDAWgBRa19QOwsNXP8CEb4tmtSc1
ga631TCB0QYDVR0RBIHJMIHGgglsb2NhbGhvc3SCD2xvY2FsaG9zdDozMDA5MIIP
bG9jYWxob3N0OjMwMDkxggU6OTA5MIIFOjkwOTGCCWNvbm5lY3RvcoIrc2VydmVy
LmRldi5ub21hZC1jbHVzdGVyLnNoaXB5YXJkLnJ1bjozMDA5MIIrc2VydmVyLmRl
di5ub21hZC1jbHVzdGVyLnNoaXB5YXJkLnJ1bjozMDA5MYIOY29ubmVjdG9yOjkw
OTCCDmNvbm5lY3Rvcjo5MDkxhwR/AAABMA0GCSqGSIb3DQEBCwUAA4ICAQBLIRS8
5Fs+1X6Ol55Jq4Vvs9wVwUHsVtBpTQPhbHS4+HJs3Txx/ovM97FdaQcCzfminowv
6ZMKT5/CtXqzFofk9bYVYRtlpC/CdwF7iFKIUnhKd58tk71uiU33H+oNSEUY8qP6
K51KlXyLbtcaCwYt6s0X1CXV27GraVEyXRpKtsriWAmx49Jku6gUUWOPU1TocYy1
TEgHr/XTmrwHhFtiM/RAdVgJzglAuKLUwyjSrAC5LDE1eDnO3T7e0zz/khjnbq8b
yvTiQKaIEn4Tc6KVvMJWqLnyk0raki5b4qc9qHlkdmzOxlRAFaLoh11ZX7Ox2V8H
fZrdoBq5h07X4DIywqPBoc3+W+9h4kPdXUyY0poeb8t+QJl4C02tm5cL9zBmAPhI
rvgRr//EoVwPbUzhmbH1APMjjByAX8nQtvAkJQJkm/KmYxB3Hj9b+be50DX/M/HK
BoeBLJGBEnEge3CmZqOE4ftjpaB2psu810sVg0UotqZ1P8uX+MQSqaCQU4p8xK9w
sOUiBnM4pyk8IaKz0P7lZDBJw9Vd59Rcdzhs2LiKO1gB/1Z5o1vgy/p7VYzDB03s
UPFYQkNhvXiZ4/zaPIszi4oQYpP9P3R6LtPcd+lQfYWxfYxWwFkhF01Rtb58OxVf
vzVE0wxLER0kpusVnrE2xIYNwPVRYsT523kCpg==
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
MIIFITCCAwmgAwIBAgIRAJr4Ed2leAyT9uyjxg7+apAwDQYJKoZIhvcNAQELBQAw
KjERMA8GA1UEChMIU2hpcHlhcmQxFTATBgNVBAMTDENvbm5lY3RvciBDQTAeFw0y
MzA1MDMxMDAxMTZaFw0yNDAxMjMxMDAxMTZaMCoxETAPBgNVBAoTCFNoaXB5YXJk
MRUwEwYDVQQDEwxDb25uZWN0b3IgQ0EwggIiMA0GCSqGSIb3DQEBAQUAA4ICDwAw
ggIKAoICAQCkQfbVtOmdbE8tTu271sOpVqJcxwqKuuUHslMST0mIIemHkZaumBSN
7ojT3ASbkpbjXzLBxmKZc5eyxgIltN10XWSEMAe2IxxeESbGi+vFY341Yjcx1Ii4
g25MF1VBcU5/3x5cPbw4aoq1UMsYUtbIR309M41nwTsAxraeGYke9nD6Qa3oBQY5
ZSq4V+oirzcMUjeBuT8tZfvKMHk+UQ56C60OEwqY2lX/IySxFdfLnzRq6oDWO1cd
N6NJrEUcTVCW1NgOvZcYH/L3kO3XgyO88TxAWDE70JYJYlv6KUpLnL4Q1IDIXNIF
pTx28zL1juuoY00y1uA/5BYykQXeQn/wpGClcYbgbd+m3pqxzUCu4q00ILMC4vVl
vngsS1Q/81/rJXe3uSlSZtomyh1MkfA2M4pZ4W3+nawJInGSEio9R8WnqOFHYlOS
HqwwvyISyNDwRAvObyOy4vEw3nbvfmwjFSFc0lIixPBNXA/mkDUe0jtQbtT5aHal
NgaWxz20dkhqa6NZBBzUCc0vd5ar996z1rKzRuOdg+UdMN9s7jTs5QcaiYh9vpYK
nU4g+hREeb8dDga5GFxfuDd6l0PGT3txLLSIyvZgG7vl/pH7PEsDQWmcre0NbiXt
j9Cp2e9kHZitBHpKc5hh9SVkEoVk6E4++OEHfrIlzpPCpbKiU+/SKwIDAQABo0Iw
QDAOBgNVHQ8BAf8EBAMCAoQwDwYDVR0TAQH/BAUwAwEB/zAdBgNVHQ4EFgQUWtfU
DsLDVz/AhG+LZrUnNYGut9UwDQYJKoZIhvcNAQELBQADggIBAJnB3qrOF1E67Oja
97l2945X9EgJwPWeR0hqh0GwCzq8JfIhB/5Oc0fpdpDUd4YZDWtMGYOn4bClMtIp
G8CFSw7o65d3/u1oqjtacMlh5PSfNZovYTPRr2hstYS+qmfhuKhggcfUNQ7wGV2b
JHzMo+EWP6GEdMCgF33qBGRKLefpzKgehxs6sTcQAftCYBe7AKKpMDhbfsnKMaFB
jWNWyvhhykz8Q6XptV9Y66C+3wvGPQduUYnwbJpZBeyIlg0h55xzWWfmgipcqwZz
xg9A4FQQiz5a/IO3SBiOCrv/cHrESlNeWnAJvQxFMLgZbpsfNV0e67dc85ZfBy5g
D4nAjTm3g75jphHTq4677D6qLlq+srOLCPOkwm8EnpGN/y+gYmtcd9efUMj3IxiP
2DPIC2z0f0jyJ8cXz438xu1lV2pjGjkv/kc9X3+dYLUvK3h4pBeR7tlSWGVaPyiN
ZGaDiKiq/AMityZxw/89JTwPCx6IPcP0mC7Fn3Ph0OL8U0ShWOWs9y2JPfnzG8Bg
wErRYIASuD7o25V9QLcXcoUfms1i8E1TAAoSflqtir7FmCq7H73QzBmYxxPUy2Zp
sZL5PySKARdGm8IopWLX08pIs0iRdm7UdCwa87k4SoOsiiXhOzuaXiSn+zC6ANov
AVENfgDzN0Lc+Ci86vXE7PEzkmLf
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
