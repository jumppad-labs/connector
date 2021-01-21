package main

import (
	"fmt"

	"github.com/shipyard-run/connector/cmd"
)

var Version = "dev"

func main() {
	fmt.Println("Connector Version:", Version)
	cmd.Execute()
}
