package notifier

import (
	"context"
	"fmt"
)

// Notifier sends messages through transports.
type Notifier struct {
	transports []TransportInterface
}

// NewNotifier creates a new Notifier with the given transports.
func NewNotifier(transports ...TransportInterface) *Notifier {
	return &Notifier{
		transports: transports,
	}
}

// Send sends a message using the first transport that supports it.
func (n *Notifier) Send(ctx context.Context, message MessageInterface) (*SentMessage, error) {
	if len(n.transports) == 0 {
		return nil, fmt.Errorf("no transports configured")
	}

	// If message specifies a transport, find it
	if transportName := message.GetTransport(); transportName != "" {
		for _, transport := range n.transports {
			if transport.String() == transportName && transport.Supports(message) {
				return transport.Send(ctx, message)
			}
		}
		return nil, fmt.Errorf("transport %q not found or does not support message", transportName)
	}

	// Otherwise, use the first transport that supports the message
	for _, transport := range n.transports {
		if transport.Supports(message) {
			return transport.Send(ctx, message)
		}
	}

	return nil, fmt.Errorf("no transport supports this message")
}

// SendAll sends a message to all transports that support it.
func (n *Notifier) SendAll(ctx context.Context, message MessageInterface) ([]*SentMessage, error) {
	if len(n.transports) == 0 {
		return nil, fmt.Errorf("no transports configured")
	}

	var results []*SentMessage
	for _, transport := range n.transports {
		if transport.Supports(message) {
			sent, err := transport.Send(ctx, message)
			if err != nil {
				return results, err
			}
			results = append(results, sent)
		}
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no transport supports this message")
	}

	return results, nil
}
