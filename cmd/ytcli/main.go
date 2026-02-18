package main

import (
	"os"

	"github.com/CoastalFuturist/ytcli/internal/cli"
)

// Canonical CLI entrypoint used by local builds and release packaging.
func main() {
	os.Exit(cli.Main(os.Args[1:], os.Stdout, os.Stderr))
}
