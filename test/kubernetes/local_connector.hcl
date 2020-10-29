exec_local "local_connector" {
	depends_on = ["exec_local.certs"]

	cmd = "../../install/kubernetes/run_local.sh"
	working_directory = "../../install/kubernetes/"
}
