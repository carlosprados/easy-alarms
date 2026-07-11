package main

import (
	"os"

	"easy-alarms/internal/cli"
)

// version is injected via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	os.Exit(cli.Execute(version))
}
