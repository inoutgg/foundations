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
//
// An optional signal can be provided to override the default SIGTERM signal.
func StartBlocking(ctx context.Context, starter Starter, sg ...os.Signal) error {
	serviceCtx, serviceCancel := context.WithCancelCause(ctx)
	defer serviceCancel(nil)

	sig := os.Interrupt
	if len(sg) > 0 {
		sig = sg[0]
	}

	sigCh := make(chan os.Signal, 1)

	signal.Notify(sigCh, sig)

	go func(ctx context.Context) {
		<-sigCh
		err := starter.Stop(ctx)
		serviceCancel(err)
	}(serviceCtx)

	go func(ctx context.Context) {
		if err := starter.Start(serviceCtx); err != nil {
			serviceCancel(err)
		}
	}(serviceCtx)

	// Wait for the starter to shutdown
	<-serviceCtx.Done()

	return context.Cause(serviceCtx)
}
