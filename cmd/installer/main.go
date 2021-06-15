package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/media-streaming-mesh/msm-cni/internal/install"
	log "github.com/sirupsen/logrus"
)

// Entry point for CNI installer
func main() {

	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// start a thread to handle system signals
	// cancel the context on interrupt
	go func(sigChan chan os.Signal, cancel context.CancelFunc) {
		sig := <-sigChan
		log.Infof("Exit signal received: %s", sig)
		cancel()
	}(sigChan, cancel)

	// execute the cobra command with given context
	rootCmd := install.GetCommand()
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
