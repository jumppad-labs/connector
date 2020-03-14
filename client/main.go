package main

import "github.com/shipyard-run/connector/protos/shipyard"

func main() {
	conn, err := grpc.Dial(":12344", grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		panic(err)
	}

	c := shipyard.NewRemoteConnectionClient(conn
}