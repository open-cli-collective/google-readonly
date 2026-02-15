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
