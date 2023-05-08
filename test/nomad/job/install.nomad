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
MIIGGDCCBACgAwIBAgIQJ79+ZBEQCoHEUezhbU1AozANBgkqhkiG9w0BAQsFADAq
MREwDwYDVQQKEwhTaGlweWFyZDEVMBMGA1UEAxMMQ29ubmVjdG9yIENBMB4XDTIz
MDUwMzE1NDIxM1oXDTI0MDEyMzE1NDIxM1owLDERMA8GA1UEChMIU2hpcHlhcmQx
FzAVBgNVBAMTDkNvbm5lY3RvciBMZWFmMIICIjANBgkqhkiG9w0BAQEFAAOCAg8A
MIICCgKCAgEAtWam84fXKTsDNUWRQegWsRkbnBuirBEkt049s+VIJWoX7uhMKK5v
PbOoJC6W4ekwUVtYzDul1hslH1zg74HX8T0ohRiE5YEMSZSZ38C2ui/PdTJv5WLC
RBSB6w/hVO4BnZpHQ9YU225yAtRErqgF7Al4DyP780oDMkGF/7oC3bqBEEsRBpQe
PxTEHK9jsiKMlOr+QuOHs4LISnTQLL5cir6C+3WawUA26AYlhctlmYbLOi12rmQI
rQkMyr6HJmZZabN7ILWbIwlPa5L9yL3WL0emsx1Cp8MqM4uAoaRri6uOUGeVbdYM
SbtqUJzJl9kWIZxY++c3e4EweFfufwYbWNHFMlk03J8wsIOEG3+Z5qmROM4GEEVN
AA7sP5P4GSfVG6weVGPWDizR1CKCJ3TTEc8Pavco+Br6NgzCB3qSUNmr359v5zP3
u9Y2u331fPb8hLRLAvwf/Ul7jMCSNKI0LD30KVWUkenNDQb2+DfwewHqLuE+SMsG
SH+uKDMPHaPwDKFb/2vRl6OCeIRHCkmzJmrNcc73N/HY7VrcoeTu5U8Gv9u41cTl
TM/Fj7k0ODeFA5PumViqW3ORGhJOJQYrSvryctEEL4LZXexKILN8d4G1MSchZtMU
jr3kpWofLoNtvsLNgggiae2yyt4FBv/JRmQi4LgsydwTs//BGKAjh6cCAwEAAaOC
ATYwggEyMA4GA1UdDwEB/wQEAwIHgDAdBgNVHSUEFjAUBggrBgEFBQcDAQYIKwYB
BQUHAwIwDAYDVR0TAQH/BAIwADAfBgNVHSMEGDAWgBTk7SiTh3LhZncJ5Q/QMXEg
+KbHDzCB0QYDVR0RBIHJMIHGgglsb2NhbGhvc3SCD2xvY2FsaG9zdDozMDA5MIIP
bG9jYWxob3N0OjMwMDkxggU6OTA5MIIFOjkwOTGCCWNvbm5lY3RvcoIrc2VydmVy
LmRldi5ub21hZC1jbHVzdGVyLnNoaXB5YXJkLnJ1bjozMDA5MIIrc2VydmVyLmRl
di5ub21hZC1jbHVzdGVyLnNoaXB5YXJkLnJ1bjozMDA5MYIOY29ubmVjdG9yOjkw
OTCCDmNvbm5lY3Rvcjo5MDkxhwR/AAABMA0GCSqGSIb3DQEBCwUAA4ICAQAfwz0g
s/eikZ/mn9TPncA/dHYIaTuFEsaP2YzHm2vdChSXX/KJkW7dxy+GCYPxafvLu0c4
+qXb2VfuJQzi9ldwQ5lydrRPn7f3baKvhnrSdSzVQr9yWZzkQa+JEQ4JNbFTJZWo
yE3fAkeeZOTB3N5hLxOVPniF7VZHBlDXu46uzt6YoBDmxYyLvKjE4dFrsNEGhpRs
dgDlTueK/Vpo/9xsI2btrC0yZ32xVcSScO8nM4LWQE8gZRuJj9aIVkfhf4hS7gwO
PCuF+iqeaKFBkDHm2vvxDH7TcT7PuV9a592oysBs+YyZyDag6yHL5HUrP1fo192n
Q0DafkgLJx+DQ8uviUSLmKiC1sgQXHYZZvJaqS+xXsdjTpR8n4y9tSIUp8SJzDxv
pMI16Jk7Np/wNf6r83P+dmQVoCImwT8qRH/xxvMg4eQEfZWdRNwZnBtBr2yNm6yY
5SH6NVykWBsUJ6mZAc9cmXRDAfAWg0CELQUkddnIccQ6L4+uWD4V1mpcNiwkF5OY
ewxWO9AjccSyR4hhVVjJl0X5/dadyd++YtgOhFP1egfMKi35TFmS5NkgfSlRpVLF
WggcATwvv44USi1AL9KbU0XobrbxE+bwKO9iljkpajrfYr+XHWeCTu+fzwKcHRdz
VcQIvLq5u+cNNsZIym/Zrmtg+h3NZ9boOqqeDA==
-----END CERTIFICATE-----
        EOH

        destination = "local/certs/server.cert"
      }
      
      template {
        data = <<-EOH
-----BEGIN RSA PRIVATE KEY-----
MIIJKAIBAAKCAgEAtWam84fXKTsDNUWRQegWsRkbnBuirBEkt049s+VIJWoX7uhM
KK5vPbOoJC6W4ekwUVtYzDul1hslH1zg74HX8T0ohRiE5YEMSZSZ38C2ui/PdTJv
5WLCRBSB6w/hVO4BnZpHQ9YU225yAtRErqgF7Al4DyP780oDMkGF/7oC3bqBEEsR
BpQePxTEHK9jsiKMlOr+QuOHs4LISnTQLL5cir6C+3WawUA26AYlhctlmYbLOi12
rmQIrQkMyr6HJmZZabN7ILWbIwlPa5L9yL3WL0emsx1Cp8MqM4uAoaRri6uOUGeV
bdYMSbtqUJzJl9kWIZxY++c3e4EweFfufwYbWNHFMlk03J8wsIOEG3+Z5qmROM4G
EEVNAA7sP5P4GSfVG6weVGPWDizR1CKCJ3TTEc8Pavco+Br6NgzCB3qSUNmr359v
5zP3u9Y2u331fPb8hLRLAvwf/Ul7jMCSNKI0LD30KVWUkenNDQb2+DfwewHqLuE+
SMsGSH+uKDMPHaPwDKFb/2vRl6OCeIRHCkmzJmrNcc73N/HY7VrcoeTu5U8Gv9u4
1cTlTM/Fj7k0ODeFA5PumViqW3ORGhJOJQYrSvryctEEL4LZXexKILN8d4G1MSch
ZtMUjr3kpWofLoNtvsLNgggiae2yyt4FBv/JRmQi4LgsydwTs//BGKAjh6cCAwEA
AQKCAgAt9srK3lq4iclwUCZUStilGzWRwrbfXqCtCdg8oxY61L0nvhi+HiT1v3YV
ZPC6YXnqw3iml16X99zaK5CbX402BUclImdaN+7DHjI3Lf+fAcpRaexMdU/ALGoX
A7kW6g/ivVrdZ3t1dnDRIrQchVqqymNvgrCunsxciZnIiHt9b2qQlFTGE/XuCfb/
Rbm/Q13XxguTK1ARPkw+AYdWLw4H4eoSiWQjH4BKHnSXiEhANJV+MlLmMVa5cZea
L9jS9BAn5mCGkz2yDQPgwCgqG2AQLtmgfQOMurkQwoJfcugFRf0Thouofxox/Jkd
v/ycy1b+QT2S5q16T+vWMoGuEgAPfhAOo+xDi0AUTzdrrVKrIUZfXAz/RA0H9AUV
4Wxg+c4wBI49zI3b8IBwGF9Yz/eBSqDpyX12Xt3hatwTS+G71q9S0Cj7vCzpO25m
3zVUd1Xt/N/yIJPITGhBH4U1gSaGOoRN6pvPOA7WHCtXVQi0FrMd3Cu7Zh4yjIUF
zUII+vDquxBoeCNEi12Bkar1hJc11GgzX6aSmXYShgeI8dkHYcXLKLW27Zxad+Wa
4jh9OegHJVHeeHR6EqXM/yxNcyq9XOPJVr7LRgFnskAacMdNRr+tkMLimytK/0jB
5qJIQgwEqHRHDesv3hmvayJX//3PO/eSE/vL0N8iA7JRJt8SgQKCAQEAzRIKjPfA
NdKvbQjOcqf8lDLwlyvP+vJHHJeQl9rpiuCau6ONJgHFw9rwRqWfizpW6CPF3oUj
t1NthUDJutmYOLw0+RlZQYkSOhYQxVmsMWQAshT4zvf/GoWOVXx2msXtA2wfBHT7
vN52p9SBCXOkwiw4rGeUgt73x0RzSTRt606R4wJ46wTOPti6brVVtLoPAkhvlTPX
+wQ920L4C16Scr5f9yxscDyyUh312NCLrIKI0RrwY7BGsC19can0fTvGb+7Ftx0B
NuW+SfjuVZACK4HpTgyXqvpkoMZebL63ebh7NC4RJUztVPdkbhu1daTZf8WIfTMR
sRmK0ohnyISzIQKCAQEA4nPA6TUCpLG4dTu2Db3FJbeP/DRCzXLNZys/Hh4ygiy+
j8FBWxA+chW+cmSbKZGOv4tpPqMHuEg3a+MRyGZ+90eD7rRZBWJWukGfMiVsvemV
gkoIz8wxHYg9At3rkAw9q8UcvZCHGcDkNR5nRDFnapSLoV/aRPvskPRK6HA/+FzS
NwJuefWRNQbHn8gq7EdWT9pyvBeDHMZhLtmIaAo3IURiAy6n/gASw3UtqHgnti+x
ttFCIF/B9hwt1REkHiE98WP2MjpF/o1jdHEg1zDX6BVuIoRHCBGu3lEZ9Agl0h0A
4/ubJ26Sb9MWMvWyEfYQzLQyxq2g6PtJuQJ5gswpxwKCAQAf4Qoa1/jdZR84SAIv
+MVfFHwqQ/lU/YzoePdVZAaiPEBRox8yJVxlEggAM4cV0b/o3obIDNJ8kU+ZQ3UY
wvLS/w8NGk+xzGk09nEs/L+z/ePNy0zSf+L8cH6r82lMrjAmNAyuWLE5ryuq83IL
0hpuxQkaZA/GOHs0UwPJAYmE5vXu4FeD0X9ubaqtwyrLqZDjvfb6rtCIiSREjaiZ
u93wUIACoLlKyWS/N0Ecr27HJpO2TgXIuYKDqM6zeMQ1I7G5fNjnmm6x5g0q2rPS
QUzVDqECLRr2zW4PQEc1iIBlP7SHbBHmRosuhjbqlwwieboGDuMk82dwrJPUHrhj
h52BAoIBAQClaQYNksImiQaC46XcrbSXE1liUM5HAceVx5ooJsigG4zqtrBFkzz5
2nYtWt2X5JHPykaLEUzvSBjrfoabynqNp7hwIV4xN57AGHTvjTS8GCY0cF21Y6Kw
vrZKJM4Pf1GA6c9PjIWSwzourtGhlzDCQlUoADsQTrCDRV5+IJgpk6udsPH/tedm
Q1iHlw/7XTRnydorGEWWPDX6ob0oueWBMFEjn+3n9CfAjBRYzcO8KWR3dK0HtsqY
OgckboviUkfLzkekcrpz8NUn1ga2CSB8j0LOha7Y7wm7rKP3hAgUTUk8PqobiIIA
msDJYny67/FfhXTdeTBjXkKAmJUnfHg7AoIBAFzaziiStB/Bb8dkVyXJYRvnqmkg
MqdRzAIABHhkckj2oyqxYRsO9DdQBm106oGOY9dbHJTv/ZhuzNFWSP0QPS5g/VXl
jkbngk2YPXTmY/yKSXfyuWmJ7hKwDn4Jb0b8rHSHT+LQUtrlL+kjTWEiCOs/6YaB
pBGkoe7QHM8m/uVFoeFxrhvaqDpE2z7XjV/JDIZrLWZmAOGDiTI58AzCwQJTHJfF
HBzOmIK/yUo49UYAMLfvek0+gpe9PWqTQVNGnkjiajAAomN36WIwLjsaOxj2uat1
Dz36pa6cg/ZLRil0hvZ7yKHdaTtj+vBHlvN2kZdXbgegy3Qi0/tHp0T4oQk=
-----END RSA PRIVATE KEY-----
        EOH

        destination = "local/certs/server.key"
      }
      
      template {
        data = <<-EOH
-----BEGIN CERTIFICATE-----
MIIFIDCCAwigAwIBAgIQOWce5qW2vMa5NyZT7sGX8zANBgkqhkiG9w0BAQsFADAq
MREwDwYDVQQKEwhTaGlweWFyZDEVMBMGA1UEAxMMQ29ubmVjdG9yIENBMB4XDTIz
MDUwMzE1NDIxMFoXDTI0MDEyMzE1NDIxMFowKjERMA8GA1UEChMIU2hpcHlhcmQx
FTATBgNVBAMTDENvbm5lY3RvciBDQTCCAiIwDQYJKoZIhvcNAQEBBQADggIPADCC
AgoCggIBAJ26MxCoq0UlHp4y6fA7LuBu6T2bfR9KLYOiPxG4smcoTovjv+6iGtsp
BJlW202Wl7/U8wYp2vRVlCguIoFrwUIUmcAhwiYMHxSG4p9KRChcTXax8xLt/1xH
vTu2g2aU/2FA+FsDS90Av0iMo5db54i74NCKRSWSwsnTC7G9JgScIWoD1w4QxGN5
cTFYp/y2caRnc3ugWD67SDr1dLeDI89SOuwb6OzLe6kE3NnClCB9O7W/k7+t53PG
eiQssPCrsB1irrXDZRHIA6VyIUzBfojy997QoMn7/QDAsx4XvnmXaA4H9q430aSM
IXn5Klxe0lBiIY8kJ7dmgIag8BFwgpmcfqwj+3pWJf3w52Sh1HZ7ts4ga3vEJsZE
q6KM082ss5otDkYP9s2wqxx7kC9fw9NPMhwCiwvl1bZLg7NQtZHj1I2B2rVMVFpe
pLXG2HmjxK6vVapbDtm8Kg3Q8/X2M5GQCwZL8EKOkpNAlZQviSOOptjRRoMNGW3L
KF6hHiIRQGOK+8cV92DstvOcLjDOwXqMT72eA01+nkj9PvTaYk2g+1QKi73P+Q8m
ia8igRkK5/zhag+4PvpO5MmGg/9Le1LjkH49rDucjo4CpGbdlURnjeaY9bZLjV6K
0pR/cvnblaCkmYM2dqIFkaUJqd5hnU7RBYbvWz1uXRcMrNibJivHAgMBAAGjQjBA
MA4GA1UdDwEB/wQEAwIChDAPBgNVHRMBAf8EBTADAQH/MB0GA1UdDgQWBBTk7SiT
h3LhZncJ5Q/QMXEg+KbHDzANBgkqhkiG9w0BAQsFAAOCAgEAR71qcV9fl4d1j/kw
L3PUViTEhrXCG8hNGgL+GExDT2zU/hKvX1qa+48JMp06Nq/vYAxm58yom/VknxSv
L3GC6F0TyIkmVQEQfX+ap5ZUj+lVFRBAaVnFPkfmqO48fRSnOYh0WuEhkdB0DVNo
C4f6i9OcUn5+IfiLqNvZtgg6p8kbpDRtrChfZTpvpUNlqswqK+v+JxNrGdL+kk0g
9ehAxOE2nPmj0D3yiCTGBNCAo+0FdyDC/ulmGDUSlkxpp0/y5G/pC49MyHDawKlY
fuRepUsGn3g9OTu6DLUvoZCJ6g/aFE0y0A46iVLWtDUplZupIFdFH2QrYu3nhAdE
GCyHW7n+YYA/lCetDMfb0u+6jQ4JcuhDQT48KxEJRlpIGb1oT+uZndXzeuBGJXXu
D6xMPTXwCnQ55egGByJM8NG7fqzSksAH5M7a38fQp0RJI4uoK6AJvbQBS0wme8AR
WnaPHSo6St2YiFsKxsM9LeoEfAArm8fDJ1I2UA88OVbQgZExikvswlqvj8VsA89d
pEDYEZymJqO0gqkJgsmIp2WZtvrr0aPYcAR8YhdDtOAnPfacwK0Zk1IPVwksG24X
1MSpOtHJZRXRqJYFDOc69frZNUIhL/KMIhK64nj7yrFbQvvonebKFAJ6BzfepRD9
pPzF7yeva8BikKzHLddHfssF3KE=
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
