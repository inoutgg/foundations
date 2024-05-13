package sender

import "context"

// Message is a message to be sent.
type Message[TPayload any] struct {
	Email   string
	Name    string
	Payload TPayload
}

// Sender is an interface for sending email messages.
type Sender[T any] interface {
	// Send sends the given message.
	Send(ctx context.Context, message Message[T]) error
}
