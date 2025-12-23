package gotify

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/shyim/go-notifier"
)

func init() {
	notifier.RegisterTransportFactory(NewTransportFactory(nil))
}

// TransportFactory creates Gotify transports from DSN.
type TransportFactory struct {
	client *http.Client
}

// NewTransportFactory creates a new Gotify transport factory.
func NewTransportFactory(client *http.Client) *TransportFactory {
	if client == nil {
		client = http.DefaultClient
	}
	return &TransportFactory{
		client: client,
	}
}

// Create creates a Gotify transport from a DSN.
// DSN format: gotify://<token>@<host> or gotify://<token>@default
// Example: gotify://A1b2C3d4@mygotify.com or gotify://A1b2C3d4@default
func (f *TransportFactory) Create(dsn *notifier.DSN) (notifier.TransportInterface, error) {
	scheme := dsn.GetScheme()
	if scheme != "gotify" {
		return nil, fmt.Errorf("unsupported scheme: scheme \"%s\" not supported (supported: %s). DSN: %s", scheme, strings.Join(f.GetSupportedSchemes(), ", "), dsn.GetOriginalDSN())
	}

	token := dsn.GetUser()
	if token == "" {
		return nil, fmt.Errorf("incomplete DSN: Missing token. DSN: %s", dsn.GetOriginalDSN())
	}

	host := dsn.GetHost()
	if host == "" {
		return nil, fmt.Errorf("incomplete DSN: Missing host. DSN: %s", dsn.GetOriginalDSN())
	}

	// Allow "default" for consistency with other transports
	// When "default" is used, host is left empty and getEndpoint() will use the default
	if host == "default" {
		host = ""
	}

	port := dsn.GetPort()

	transport := NewTransport(token, f.client)
	if host != "" {
		transport.SetHost(host)
	}
	if port > 0 {
		transport.SetPort(port)
	}

	return transport, nil
}

// Supports checks if the factory supports the given DSN.
func (f *TransportFactory) Supports(dsn *notifier.DSN) bool {
	for _, scheme := range f.GetSupportedSchemes() {
		if dsn.GetScheme() == scheme {
			return true
		}
	}
	return false
}

// GetSupportedSchemes returns the supported DSN schemes.
func (f *TransportFactory) GetSupportedSchemes() []string {
	return []string{"gotify"}
}
