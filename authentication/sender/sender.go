package sender

import "context"

// Message is a message to be sent.
type Message struct {
	Email   string
	Payload any
}

// Sender is an interface for sending email messages.
type Sender interface {
	// Send sends the given message.
	Send(ctx context.Context, message Message) error
}
