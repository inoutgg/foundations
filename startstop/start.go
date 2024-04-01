package startstop

import (
	"context"
	"os"
	"os/signal"
)

type Starter interface {
	// Start typically launches a cancellable blocking operation.
	Start(context.Context) error
	Stop(context.Context) error
}

// StartBlocking starts the given Starter and blocks the goroutine until
// SIGTERM signal is received.
func StartBlocking(ctx context.Context, starter Starter) error {
	serviceCtx, serviceCancel := context.WithCancelCause(ctx)
	defer serviceCancel(nil)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	go func(ctx context.Context) {
		<-sig
		err := starter.Stop(ctx)
		serviceCancel(err)
	}(serviceCtx)

	go func(ctx context.Context) {
		if err := starter.Start(serviceCtx); err != nil {
			serviceCancel(err)
		}
	}(serviceCtx)

	// Wait for server to shutdown
	<-serviceCtx.Done()

	return context.Cause(serviceCtx)
}
