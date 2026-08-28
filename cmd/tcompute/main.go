package main

import (
	"fmt"
	"os"

	"github.com/TashanGKD/tashan-compute/internal/cli"
)

func main() {
	if err := cli.NewRoot(cli.Dependencies{}).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
