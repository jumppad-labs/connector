package cmd

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/hashicorp/go-hclog"
	"github.com/jumppad-labs/connector/http"
	"github.com/jumppad-labs/connector/integrations"
	"github.com/jumppad-labs/connector/integrations/k8s"
	"github.com/jumppad-labs/connector/integrations/local"
	"github.com/jumppad-labs/connector/integrations/nomad"
	"github.com/jumppad-labs/connector/protos/shipyard"
	"github.com/jumppad-labs/connector/remote"
	"github.com/soheilhy/cmux"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

var bindAddr string
var pathCertCA string
var pathCertServer string
var pathKeyServer string
var logLevel string
var integration string
var namespace string

func init() {
	runCmd.Flags().StringVarP(&bindAddr, "bind", "", ":9090", "Bind address for the application")
	runCmd.Flags().StringVarP(&pathCertCA, "ca-path", "", "", "Path for the PEM encoded self signed CA certificate")
	runCmd.Flags().StringVarP(&pathCertServer, "cert-path", "", "", "Path for the servers PEM encoded TLS certificate")
	runCmd.Flags().StringVarP(&pathKeyServer, "key-path", "", "", "Path for the servers PEM encoded Private Key")
	runCmd.Flags().StringVarP(&logLevel, "log-level", "", "info", "Log output level [debug, trace, info]")
	runCmd.Flags().StringVarP(&integration, "integration", "", "", "Integration to use [kubernetes]")
	runCmd.Flags().StringVarP(&namespace, "namespace", "", "shipyard", "Kubernetes namespace when using Kubernetes integration, default: shipyard")
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the connector",
	Long:  `Runs the connector with the given options`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Do Stuff Here

		lo := hclog.LoggerOptions{}
		lo.Level = hclog.LevelFromString(logLevel)
		logger := hclog.New(&lo)

		// create the integration
		var in integrations.Integration
		switch integration {
		case "kubernetes":
			in = k8s.New(logger.Named("k8s_integration"), namespace)
			logger.Info("Loading integration Kubernetes")
		case "nomad":
			in = nomad.New(logger.Named("nomad_integration"))
			logger.Info("Loading integration Nomad")
		default:
			in = local.New(logger.Named("local_integration"))
		}

		// setup the listener
		l, err := net.Listen("tcp", bindAddr)
		if err != nil {
			logger.Error("Unable to listen at", "address", bindAddr, "error", err)
			os.Exit(1)
		}

		// if we are using TLS wrap the listener in a TLS listener
		if pathCertServer != "" && pathKeyServer != "" {
			logger.Info("Enabling TLS for HTTP endpoint")

			var certificate tls.Certificate
			certificate, err = tls.LoadX509KeyPair(pathCertServer, pathKeyServer)
			if err != nil {
				logger.Error("Error loading certificates", "error", err)
				os.Exit(1)
			}

			// Get the SystemCertPool, continue with an empty pool on error
			rootCAs, _ := x509.SystemCertPool()
			if rootCAs == nil {
				rootCAs = x509.NewCertPool()
			}

			// Read in the cert file
			if pathCertCA != "" {
				certs, err := ioutil.ReadFile(pathCertCA)
				if err != nil {
					log.Fatalf("Failed to append %q to RootCAs: %v", pathCertCA, err)
				}

				// Append our cert to the system pool
				if ok := rootCAs.AppendCertsFromPEM(certs); !ok {
					log.Println("No certs appended, using system certs only")
				}
			}

			config := &tls.Config{
				Certificates: []tls.Certificate{certificate},
				Rand:         rand.Reader,
				RootCAs:      rootCAs,
			}

			// Create TLS listener.
			l = tls.NewListener(l, config)
		}

		// create a cmux
		// cmux allows us to have a grpc and a http server listening on the same port
		m := cmux.New(l)
		httpListener := m.Match(cmux.HTTP1Fast())
		grpcListener := m.Match(cmux.Any())

		grpcServer := grpc.NewServer()
		s := remote.New(logger.Named("grpc_server"), nil, nil, in)

		shipyard.RegisterRemoteConnectionServer(grpcServer, s)

		// create a listener for the server
		logger.Info("Starting gRPC server", "bind_addr", bindAddr)

		// start the gRPC server
		go grpcServer.Serve(grpcListener)

		httpS := http.NewLocalServer(httpListener, logger.Named("http_server"))

		// start the http server in the background
		logger.Info("Starting HTTP server", "bind_addr", bindAddr)
		go httpS.Serve()

		// start the multiplex listener
		go m.Serve()

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

		// Block until a signal is received.
		sig := <-c
		log.Println("Got signal:", sig)

		s.Shutdown()
		httpS.Close()

		m.Close()

		return nil
	},
}
