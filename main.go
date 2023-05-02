package main

import (
	"fmt"

	"github.com/jumppad-labs/connector/cmd"
)

var Version = "dev"

func main() {
	fmt.Println("Connector Version:", Version)
	cmd.Execute()
}
