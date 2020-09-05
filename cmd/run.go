package cmd

import (
	"net"
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/connector/http"
	"github.com/shipyard-run/connector/protos/shipyard"
	"github.com/shipyard-run/connector/remote"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the connector",
	Long:  `Runs the connector with the given options`,
	Run: func(cmd *cobra.Command, args []string) {
		// Do Stuff Here

		lo := hclog.LoggerOptions{}
		lo.Level = hclog.LevelFromString(logLevel)
		l := hclog.New(&lo)

		s := remote.New(l.Named("grpc_server"))
		grpcServer := grpc.NewServer()
		shipyard.RegisterRemoteConnectionServer(grpcServer, s)

		// create a listener for the server
		lis, err := net.Listen("tcp", grpcBindAddr)
		if err != nil {
			l.Error("Unable to list on address", "bind_addr", grpcBindAddr)
			os.Exit(1)
		}

		// start the http server in the background
		l.Info("Starting HTTP server", "bind_addr", httpBindAddr)
		httpS := http.NewLocalServer(grpcBindAddr, httpBindAddr, l)
		httpS.Serve()

		// start the gRPC server
		l.Info("Starting gRPC server", "bind_addr", grpcBindAddr)
		grpcServer.Serve(lis)
	},
}

var grpcBindAddr string
var httpBindAddr string
var pathCertRoot string
var pathCertServer string
var pathKeyServer string
var logLevel string

func init() {
	runCmd.Flags().StringVarP(&grpcBindAddr, "grpc-bind", "", ":9090", "Bind address for the gRPC API")
	runCmd.Flags().StringVarP(&grpcBindAddr, "http-bind", "", ":9091", "Bind address for the HTTP API")
	runCmd.Flags().StringVarP(&pathCertRoot, "root-cert-path", "", "", "Path for the PEM encoded TLS root certificate")
	runCmd.Flags().StringVarP(&pathCertServer, "server-cert-path", "", "", "Path for the servers PEM encoded TLS certificate")
	runCmd.Flags().StringVarP(&pathCertServer, "server-key-path", "", "", "Path for the servers PEM encoded TLS certificate")
	runCmd.Flags().StringVarP(&logLevel, "log-level", "", "", "Log output level [debug, trace, info]")
}
