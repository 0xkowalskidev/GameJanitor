package main

import (
	"os"

	"github.com/0xkowalskidev/gamejanitor/cmd/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
