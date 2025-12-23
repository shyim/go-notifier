package discord

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/shyim/go-notifier"
)

func init() {
	notifier.RegisterTransportFactory(NewTransportFactory(nil))
}

// TransportFactory creates Discord transports from DSN.
type TransportFactory struct {
	client *http.Client
}

// NewTransportFactory creates a new Discord transport factory.
func NewTransportFactory(client *http.Client) *TransportFactory {
	if client == nil {
		client = http.DefaultClient
	}
	return &TransportFactory{
		client: client,
	}
}

// Create creates a Discord transport from a DSN.
// DSN format: discord://<token>@default?webhook_id=<webhook_id>
// Example: discord://abc123@default?webhook_id=123456789012345678
func (f *TransportFactory) Create(dsn *notifier.DSN) (notifier.TransportInterface, error) {
	scheme := dsn.GetScheme()
	if scheme != "discord" {
		return nil, fmt.Errorf("unsupported scheme: scheme \"%s\" not supported (supported: %s). DSN: %s", scheme, strings.Join(f.GetSupportedSchemes(), ", "), dsn.GetOriginalDSN())
	}

	token := dsn.GetUser()
	if token == "" {
		return nil, fmt.Errorf("incomplete DSN: Missing token. DSN: %s", dsn.GetOriginalDSN())
	}

	webhookID := dsn.GetOption("webhook_id")
	if webhookID == "" {
		return nil, fmt.Errorf("missing required option: webhook_id")
	}

	host := dsn.GetHost()
	if host == "default" {
		host = ""
	}
	port := dsn.GetPort()

	transport := NewTransport(webhookID, token, f.client)
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
	return []string{"discord"}
}
