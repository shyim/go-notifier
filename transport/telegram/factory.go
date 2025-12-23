package telegram

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/shyim/go-notifier"
)

func init() {
	notifier.RegisterTransportFactory(NewTransportFactory(nil))
}

// TransportFactory creates Telegram transports from DSN.
type TransportFactory struct {
	client *http.Client
}

// NewTransportFactory creates a new Telegram transport factory.
func NewTransportFactory(client *http.Client) *TransportFactory {
	if client == nil {
		client = http.DefaultClient
	}
	return &TransportFactory{
		client: client,
	}
}

// Create creates a Telegram transport from a DSN.
// DSN format: telegram://<token>@default?channel=<channel_id>
// Example: telegram://123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11@default?channel=-1001234567890
func (f *TransportFactory) Create(dsn *notifier.DSN) (notifier.TransportInterface, error) {
	scheme := dsn.GetScheme()
	if scheme != "telegram" {
		return nil, fmt.Errorf("unsupported scheme: scheme \"%s\" not supported (supported: %s). DSN: %s", scheme, strings.Join(f.GetSupportedSchemes(), ", "), dsn.GetOriginalDSN())
	}

	token := dsn.GetUser()
	if token == "" {
		return nil, fmt.Errorf("incomplete DSN: Missing token. DSN: %s", dsn.GetOriginalDSN())
	}

	// Support both user:password and user-only formats
	password := dsn.GetPassword()
	if password != "" {
		token = token + ":" + password
	}

	channel := dsn.GetOption("channel")
	host := dsn.GetHost()
	if host == "default" {
		host = ""
	}
	port := dsn.GetPort()

	transport := NewTransport(token, channel, f.client)
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
	return []string{"telegram"}
}
