package main

import (
	"fmt"
	"net"
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/env"
	"github.com/shipyard-run/connector/http"
	"github.com/shipyard-run/connector/protos/shipyard"
	"github.com/shipyard-run/connector/remote"
	"google.golang.org/grpc"
)

var grpcBindAddr = env.String("BIND_ADDR_GRPC", false, "localhost:9090", "Bind address for the gRPC server")
var httpBindAddr = env.String("BIND_ADDR_HTTP", false, "localhost:9091", "Bind address for the HTTP server")
var logLevel = env.String("LOG_LEVEL", false, "info", "Log level for output log [info, debug, trace]")

func main() {
	err := env.Parse()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	lo := hclog.LoggerOptions{}
	lo.Level = hclog.LevelFromString(*logLevel)
	l := hclog.New(&lo)

	s := remote.New(l.Named("grpc_server"))
	grpcServer := grpc.NewServer()
	shipyard.RegisterRemoteConnectionServer(grpcServer, s)

	// create a listener for the server
	lis, err := net.Listen("tcp", *grpcBindAddr)
	if err != nil {
		l.Error("Unable to list on address", "bind_addr", *grpcBindAddr)
		os.Exit(1)
	}

	// start the http server in the background
	l.Info("Starting HTTP server", "bind_addr", *httpBindAddr)
	httpS := http.NewLocalServer(*grpcBindAddr, *httpBindAddr, l)
	httpS.Serve()

	// start the gRPC server
	l.Info("Starting gRPC server", "bind_addr", *grpcBindAddr)
	grpcServer.Serve(lis)
}
