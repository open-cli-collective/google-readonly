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

	"github.com/open-cli-collective/google-cli-common/config"

	"github.com/open-cli-collective/google-readonly/internal/appidentity"
	"github.com/open-cli-collective/google-readonly/internal/cmd/root"
)

func main() {
	// Register this CLI's identity before any config/keychain/auth call: it
	// stamps the config dir, keyring service, env-var prefixes, and scope set
	// the shared google-cli-common library resolves against.
	config.Register(appidentity.Identity())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	root.ExecuteContext(ctx)
}
