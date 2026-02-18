package main

import (
	"os"

	"github.com/CoastalFuturist/ytcli/internal/cli"
)

// Compatibility shim so `go install github.com/CoastalFuturist/ytcli@latest`
// installs the CLI from the module root.
func main() {
	os.Exit(cli.Main(os.Args[1:], os.Stdout, os.Stderr))
}
