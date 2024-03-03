package shutdown

import (
	"context"
	"os"
	"os/signal"
)

type Starter interface {
	// Start typically launches a cancellable blocking operation.
	Start(context.Context) error
	Shutdown(context.Context) error
}

// StartWithShutdown starts the given Starter and blocks the goroutine until
// SIGTERM signal is received.
func StartWithShutdown(ctx context.Context, starter Starter) error {
	serverCtx, serverCancel := context.WithCancelCause(ctx)
	defer serverCancel(nil)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	go func(serverCtx context.Context) {
		<-sig
		err := starter.Shutdown(serverCtx)
		serverCancel(err)
	}(serverCtx)

	go func(ctx context.Context) {
		if err := starter.Start(serverCtx); err != nil {
			serverCancel(err)
		}
	}(serverCtx)

	// Wait for server to shutdown
	<-serverCtx.Done()

	return serverCtx.Err()
}
