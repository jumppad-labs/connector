package cmd

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"

	"github.com/hashicorp/go-hclog"
	"github.com/jumppad-labs/connector/http"
	"github.com/jumppad-labs/connector/integrations"
	"github.com/jumppad-labs/connector/integrations/k8s"
	"github.com/jumppad-labs/connector/integrations/local"
	"github.com/jumppad-labs/connector/integrations/nomad"
	"github.com/jumppad-labs/connector/protos/shipyard"
	"github.com/jumppad-labs/connector/remote"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the connector",
	Long:  `Runs the connector with the given options`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Do Stuff Here

		lo := hclog.LoggerOptions{}
		lo.Level = hclog.LevelFromString(logLevel)
		l := hclog.New(&lo)

		// create the integration
		var in integrations.Integration
		switch integration {
		case "kubernetes":
			in = k8s.New(l.Named("k8s_integration"), namespace)
		case "nomad":
			in = nomad.New(l.Named("nomad_integration"))
		default:
			in = local.New(l.Named("local_integration"))
		}

		grpcServer := grpc.NewServer()
		s := remote.New(l.Named("grpc_server"), nil, nil, in)

		// do we need to set up the server to use TLS?
		if pathCertServer != "" && pathKeyServer != "" && pathCertRoot != "" {
			certificate, err := tls.LoadX509KeyPair(pathCertServer, pathKeyServer)
			if err != nil {
				return fmt.Errorf("could not load server key pair: %s", err)
			}

			// Create a certificate pool from the certificate authority
			certPool := x509.NewCertPool()
			ca, err := ioutil.ReadFile(pathCertRoot)
			if err != nil {
				return fmt.Errorf("could not read ca certificate: %s", err)
			}

			// Append the client certificates from the CA
			if ok := certPool.AppendCertsFromPEM(ca); !ok {
				return errors.New("failed to append client certs")
			}

			clientAuth := tls.RequireAndVerifyClientCert
			if !verifyClient {
				clientAuth = tls.NoClientCert
			}

			creds := credentials.NewTLS(&tls.Config{
				ClientAuth:   clientAuth,
				Certificates: []tls.Certificate{certificate},
				ClientCAs:    certPool,
			})

			grpcServer = grpc.NewServer(grpc.Creds(creds))
			s = remote.New(l.Named("grpc_server"), certPool, &certificate, in)
		}

		shipyard.RegisterRemoteConnectionServer(grpcServer, s)

		// create a listener for the server
		l.Info("Starting gRPC server", "bind_addr", grpcBindAddr)
		lis, err := net.Listen("tcp", grpcBindAddr)
		if err != nil {
			l.Error("Unable to list on address", "bind_addr", grpcBindAddr)
			os.Exit(1)
		}

		// start the gRPC server
		go grpcServer.Serve(lis)

		// start the http server in the background
		l.Info("Starting HTTP server", "bind_addr", httpBindAddr)
		httpS := http.NewLocalServer(pathCertRoot, pathKeyRoot, pathCertServer, pathKeyServer, grpcBindAddr, httpBindAddr, l)
		err = httpS.Serve()
		if err != nil {
			l.Error("Unable to start HTTP server", "error", err)
			os.Exit(1)
		}

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		signal.Notify(c, os.Kill)

		// Block until a signal is received.
		sig := <-c
		log.Println("Got signal:", sig)

		s.Shutdown()

		return nil
	},
}

var grpcBindAddr string
var httpBindAddr string
var pathCertRoot string
var pathKeyRoot string
var pathCertServer string
var pathKeyServer string
var logLevel string
var integration string
var namespace string
var verifyClient bool
var disableLocalExpose bool

func init() {
	runCmd.Flags().StringVarP(&grpcBindAddr, "grpc-bind", "", ":9090", "Bind address for the gRPC API")
	runCmd.Flags().StringVarP(&httpBindAddr, "http-bind", "", ":9091", "Bind address for the HTTP API")
	runCmd.Flags().StringVarP(&pathCertRoot, "root-cert-path", "", "", "Path for the PEM encoded TLS root certificate")
	runCmd.Flags().StringVarP(&pathKeyRoot, "root-cert-key", "", "", "Path for the PEM encoded TLS root key needed to generate certificates")
	runCmd.Flags().StringVarP(&pathCertServer, "server-cert-path", "", "", "Path for the servers PEM encoded TLS certificate")
	runCmd.Flags().StringVarP(&pathKeyServer, "server-key-path", "", "", "Path for the servers PEM encoded Private Key")
	runCmd.Flags().StringVarP(&logLevel, "log-level", "", "info", "Log output level [debug, trace, info]")
	runCmd.Flags().StringVarP(&integration, "integration", "", "", "Integration to use [kubernetes]")
	runCmd.Flags().StringVarP(&namespace, "namespace", "", "shipyard", "Kubernetes namespace when using Kubernetes integration, default: shipyard")
	runCmd.Flags().BoolVarP(&verifyClient, "disableLocalExpose", "", false, "Disable exposing local services to remote connections")
	runCmd.Flags().BoolVarP(&verifyClient, "disable-remote-expose", "", true, "Verify client cert has been signed by same root as CA")
}
