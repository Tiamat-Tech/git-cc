package main

import (
	"fmt"
	"os"

	"github.com/skalt/git-cc/cmd"
)

// provided by goreleaser; see .goreleaser.yml & https://goreleaser.com/cookbooks/using-main.version/
var (
	version, commit, date string
)

func main() {
	if err := cmd.Cmd(version, commit, date).Execute(); err != nil {
		fmt.Fprint(os.Stderr, err.Error()+"\n")
		os.Exit(1)
	}
}
