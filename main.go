package main

import (
	"github.com/skalt/git-cc/cmd"
)

// provided by goreleaser; see .goreleaser.yml
var version string

func main() {
	cmd.Cmd.Execute()
}
