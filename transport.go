package notifier

import (
	"context"
	"fmt"
	"net/http"
	"sync"
)

// Global transport factory registry
var (
	transportFactories   []TransportFactoryInterface
	transportFactoriesMu sync.RWMutex
)

// RegisterTransportFactory registers a transport factory globally.
// This is typically called from init() in transport packages.
func RegisterTransportFactory(factory TransportFactoryInterface) {
	transportFactoriesMu.Lock()
	defer transportFactoriesMu.Unlock()
	transportFactories = append(transportFactories, factory)
}

// NewTransportFromDSN creates a transport from a DSN string using registered factories.
func NewTransportFromDSN(dsnString string) (TransportInterface, error) {
	dsn, err := NewDSN(dsnString)
	if err != nil {
		return nil, err
	}

	transportFactoriesMu.RLock()
	defer transportFactoriesMu.RUnlock()

	for _, factory := range transportFactories {
		if factory.Supports(dsn) {
			return factory.Create(dsn)
		}
	}

	return nil, fmt.Errorf("no registered transport factory supports scheme: %s", dsn.GetScheme())
}

// TransportInterface represents a transport that can send messages.
type TransportInterface interface {
	// Send sends a message and returns the sent message with transport info.
	Send(ctx context.Context, message MessageInterface) (*SentMessage, error)
	// Supports checks if the transport supports the given message.
	Supports(message MessageInterface) bool
	// String returns the transport string representation.
	String() string
}

// TransportFactoryInterface creates transports from DSN.
type TransportFactoryInterface interface {
	// Create creates a transport from the given DSN.
	Create(dsn *DSN) (TransportInterface, error)
	// Supports checks if the factory supports the given DSN.
	Supports(dsn *DSN) bool
}

// AbstractTransport provides common transport functionality.
type AbstractTransport struct {
	client *http.Client
	host   string
	port   int
}

func NewAbstractTransport(client *http.Client) *AbstractTransport {
	if client == nil {
		client = http.DefaultClient
	}
	return &AbstractTransport{
		client: client,
	}
}

func (t *AbstractTransport) SetHost(host string) *AbstractTransport {
	t.host = host
	return t
}

func (t *AbstractTransport) SetPort(port int) *AbstractTransport {
	t.port = port
	return t
}

func (t *AbstractTransport) GetEndpoint() string {
	host := t.host
	if host == "" {
		host = "localhost"
	}
	if t.port > 0 {
		return fmt.Sprintf("%s:%d", host, t.port)
	}
	return host
}

func (t *AbstractTransport) GetDefaultHost() string {
	return "localhost"
}

func (t *AbstractTransport) GetClient() *http.Client {
	return t.client
}

// AbstractTransportFactory provides common factory functionality.
type AbstractTransportFactory struct {
	client *http.Client
}

func NewAbstractTransportFactory(client *http.Client) *AbstractTransportFactory {
	if client == nil {
		client = http.DefaultClient
	}
	return &AbstractTransportFactory{
		client: client,
	}
}

func (f *AbstractTransportFactory) GetClient() *http.Client {
	return f.client
}
