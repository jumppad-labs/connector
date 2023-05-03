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
MIIFxzCCA6+gAwIBAgIQPIK4CAlk9cZ8DdmxpIuPbzANBgkqhkiG9w0BAQsFADAT
MREwDwYDVQQKEwhTaGlweWFyZDAeFw0yMzA1MDMwNjQwMzVaFw0yNDAxMjMwNjQw
MzVaMBMxETAPBgNVBAoTCFNoaXB5YXJkMIICIjANBgkqhkiG9w0BAQEFAAOCAg8A
MIICCgKCAgEArJ/AMeApJdaacsxsGijxX2RPANGwXu9zKvH7iEHZBE/OnlsAiISR
T8hRYBeXPv/ST70T+l7QNzokLrzIBu3aoNzuLZnG6GCPQfNS5e0Ni7EvTH3SP+na
qA96m8YsfdXDi4I6lC7c4gTHMO+m3UlG0vxoPq3iDy38cd9gUbpUGQJPWyshrGJS
OOI6uscJLxvi1c3YseMoe2TLR2XbTNIpbj5JCHPSItdKVdcAmg7p8FAMA/vGQBUx
C8xCzAo1jlyX6puRo/k1gE0wP+Gr5thXKFPhRr+AvgJtI4eExXClGdHEu5n+Zt4g
2xJT8SNDCkFCT6Qf9b2FDSnNxT7cwR2NnVyMkzYppkxPpB/CXZnZ8C2AI5Fr5xYZ
jSX/e0+gvW53rtlK0yTkxvzEBMjesdGX8Ez82xlMOSbvu7aJdzsWMXXGDqbNEie4
mzogQqbr/JvoDe+rRGw77ePj7iRHnqLTA4i9RFoMqHnsgm6xpiEib4G2qbgwMO6V
33LjCBhMLj/gix/HYRcIBB8qe14hTaB10Gnc6entxnMCwSOJp+agppQZEeTQA77e
hiF5kO37DIyuZq1Al0pJ0yhzpDxk61N8nS+TNY+0frRplK7QukiFT0ltX34LBwR5
trZ6DtoGgrbIXbP7XWpLfcK+KP4OOED1FQ9QHubZLxuevOWH0DNC2LkCAwEAAaOC
ARUwggERMA4GA1UdDwEB/wQEAwIHgDAdBgNVHSUEFjAUBggrBgEFBQcDAQYIKwYB
BQUHAwIwDAYDVR0TAQH/BAIwADCB0QYDVR0RBIHJMIHGgglsb2NhbGhvc3SCD2xv
Y2FsaG9zdDozMDA5MIIPbG9jYWxob3N0OjMwMDkxggU6OTA5MIIFOjkwOTGCCWNv
bm5lY3RvcoIrc2VydmVyLmRldi5ub21hZC1jbHVzdGVyLnNoaXB5YXJkLnJ1bjoz
MDA5MIIrc2VydmVyLmRldi5ub21hZC1jbHVzdGVyLnNoaXB5YXJkLnJ1bjozMDA5
MYIOY29ubmVjdG9yOjkwOTCCDmNvbm5lY3Rvcjo5MDkxhwR/AAABMA0GCSqGSIb3
DQEBCwUAA4ICAQBaYtE/3B1BXO0ULccb4us0smVXkf1T+5ikdZtz8zvjTn7lKVUj
inBAWUvwRfURehXN/i/kd0WiEURYIydl2HdLeGvWdNTgrWo9ne945FHcycrfHbOz
juaF5KzsSDXvcNK9Tvq19zXbPZvtt+bq9oE6v/RlxvLShDpz82PcRTcYBsENSZnl
2Gbl+IQyaamefC3cAFoBv/AcAPvADmfWhPL2RQLNaonfmMPEzgQW/yvKJ9lRpi/R
yh/VUZ0txn/c1+7ud/aijtqL6uo5+xzKi/dkpmEmPw4ZseCuOLE/ftlz+ZVzAOTf
xMXD7UuFJ6YF15R9hZ0T/3HJyvdmiYyWpI2yS9KONZwGzBRWpolR1NZhuOU7hrDO
QqIs4it/NNTEOt8tcFw6bJzP0iyDxdByHRkJDbupuOnTMzsBB+e0o/is0EfruQg6
rCpC5mG/tHql8LRf3UZA8nihfqKsC9PHRli6md6AT8f/o54F4R8kceKga1wMAxVf
VjmqDWRJ1K6NU5DJWGAdQOgm2iS7uAc4DCMsEkxOeyYci0HjlZofKbgROBPK0YD0
hvvlVprBSdBddsl0auW4z3NKG9lSQxvoD4zrylZtBcK/enS6a5D0Ow/E/1eTySD4
wSH3eJLqxy5wiNDhsAMp5CBnyRy5mt48dgoyzdVULg1JuFLKDGyljFkawg==
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
MIIE8jCCAtqgAwIBAgIQRE79RBGBq4ZQtnYo0/0xZjANBgkqhkiG9w0BAQsFADAT
MREwDwYDVQQKEwhTaGlweWFyZDAeFw0yMzA1MDMwNjQwMzRaFw0yNDAxMjMwNjQw
MzRaMBMxETAPBgNVBAoTCFNoaXB5YXJkMIICIjANBgkqhkiG9w0BAQEFAAOCAg8A
MIICCgKCAgEAnBPjPa7KBMWMnNF4hvGp5n5bAKY0ljV/rkeEfTGyb7umiIGO41yR
17OAVGtOKCwsQ+KIcBtRcTphKrr7XBSvI5KMvvrE9ld2yZ4d2n2srOdppHhDjwLY
02xc0B0oVYBtmgkf1p5xRwEp8DjWNaWimg+Dkp/8O+tDOWhAJ9QrFX4uDpgWplwM
ieWINbapP7qLY0l+D9eAyVOvZkuzgWTUhSs3Ak0Qey18U2nVw6KnpKgK+MJdhhN8
l8nu6lBxACykyjLRj6RNge4QSeohw86G7Z3EYUbDHQ9gdpK3zpWwYL/Oby9uodQj
auVpLXxMt822vxs57DO6tjlMoKWqAMyOYkIYUCbGpHTnqQSH6y/L5pAXIFIzLwlo
9NQLlLbvXru1qb1ZsNp/jOgGppZK1OrbSt0BywG0aTdesm7w+up6ud7t1fvLvfug
XbOwBcV47SLgAuQTCKLWmxCO8I9tjo0IyDvEff4sL6LvoIpk+XwMDRCf/aKZFzb8
JOcfcbjeC0gD/8Zd0WYLZBh2/uFmsfbNJcPI5kN2VA9CRPz8ks2MZmmAq70dc1em
AyFSXZlkZmH8oD48HpsYPW0u6zNCiPeeqJnY9qPydI/T9VSY67m6g/sHL6Vc8W3X
3vMvfxEjYikLfL6ngQxcecUc6zpjm3uFVadKgH6wxjVu5dc4ljMZq+ECAwEAAaNC
MEAwDgYDVR0PAQH/BAQDAgKEMA8GA1UdEwEB/wQFMAMBAf8wHQYDVR0OBBYEFDHc
r/FRSyZwoZKVCx6nJUhZZXuaMA0GCSqGSIb3DQEBCwUAA4ICAQBkqTOOaZHg6Wge
4fImV9S4fC8lB82h3BaQV+nyc5TOpiD+SwgSraQbXC9JeplTvMJZF6odDHw13UB3
tovAngapkMh0ZHIdRxRHz9fnhQfNiR/Lbr85FhHuvJgctFOFL/zj1DViraTFkHmT
q7MJi84to9AA8IMb8HEBm2OIF3YJonXpFWlEiHqzIQM7z44Upgu8H8WEwyZ60E7f
2p2AaVyNVthJkw/BZQl1bXyOIaiqhl7ERiRAynbVlCt/2+SZfs5N2vHMigUf7GAO
XueWDEbXqhDNqnyCP3qiq2xktkZk5SaGes3EGxmEBAh+G3E0Mua9Vzv89bPYPhaR
nLS8HxdFcXr53JhNrU6LWGVUL8upjXsBoBrKiFnfAwe6bIMGKoj8sfw8gHf9viLg
3dkNA8bxshAoXiiK8VGZEemoN6pAE+lh7mWdF3BgK/ryJMvW8Bm1zeF5upFpnJLV
rHbgE1BUqf4TYQhbKPJ0h+3scJcbSZCxAIIbmI3Lvi1LvQvXJGZkdIj6qlvxNin/
0Bo6GSGqKqfAZkEi/st5rOqiCmvv3OBUAg1/2xgdNbnnp+M0IiajNFt3BHweBRGb
/xvqmfNkUQOVWMZMifhYnrNpN99NsQzDNphMWRLM3uPwmbvTeWoL62KYWbJvid50
aIM7RRtZIKOQAzclx/I8Oq1f6E2J4w==
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
        LISTEN_ADDR = ":19090"
        NAME        = "Example1"
      }

      config {
        image = "connector:dev"

        ports = ["http", "grpc"]
        args = [
          "run",
          "--grpc-bind=:30090",
          "--http-bind=:30091",
          "--log-level=debug",
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
