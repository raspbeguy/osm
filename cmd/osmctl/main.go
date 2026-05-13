package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/raspbeguy/osm/cmd/osmctl/cmd"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := cmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "osmctl:", err)
		os.Exit(1)
	}
}
