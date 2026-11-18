resource "exec" "certs" {
  image {
    name = variable.connector_image
  }

  # the script writes to ./certs
  working_directory = "/"
  script            = file("./generate_certs.sh")

  volume {
    source      = "./certs"
    destination = "/certs"
  }
}
