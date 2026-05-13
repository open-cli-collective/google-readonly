// Package main is the entry point for the gro CLI.
//
// Distribution is fully automated: merges to main with feat:/fix: prefixes
// trigger auto-release, which runs GoReleaser (handling Homebrew + binary
// artifacts) and emits a release-published event that fans out to the
// chocolatey and winget publish workflows.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/open-cli-collective/google-readonly/internal/cmd/root"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	root.ExecuteContext(ctx)
}
