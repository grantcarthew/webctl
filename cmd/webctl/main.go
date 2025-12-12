package main

import (
	"os"

	"github.com/grantcarthew/webctl/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
