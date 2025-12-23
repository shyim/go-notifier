package slack

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/shyim/go-notifier"
)

func init() {
	notifier.RegisterTransportFactory(NewTransportFactory(nil))
}

// TransportFactory creates Slack transports from DSN.
type TransportFactory struct {
	client *http.Client
}

// NewTransportFactory creates a new Slack transport factory.
func NewTransportFactory(client *http.Client) *TransportFactory {
	if client == nil {
		client = http.DefaultClient
	}
	return &TransportFactory{
		client: client,
	}
}

// validTokenPattern validates Slack token format.
var validTokenPattern = regexp.MustCompile(`^xox(b-|p-|a-2)`)

// Create creates a Slack transport from a DSN.
// DSN format: slack://<token>@default?channel=<channel_id>
// Example: slack://xoxb-1234567890-1234567890123-abcdefghijklmnopqrstuvwx@default?channel=C1234567890
func (f *TransportFactory) Create(dsn *notifier.DSN) (notifier.TransportInterface, error) {
	scheme := dsn.GetScheme()
	if scheme != "slack" {
		return nil, fmt.Errorf("unsupported scheme: scheme \"%s\" not supported (supported: %s). DSN: %s", scheme, strings.Join(f.GetSupportedSchemes(), ", "), dsn.GetOriginalDSN())
	}

	accessToken := dsn.GetUser()
	if accessToken == "" {
		return nil, fmt.Errorf("incomplete DSN: Missing access token. DSN: %s", dsn.GetOriginalDSN())
	}

	// Validate token format
	if !validTokenPattern.MatchString(accessToken) {
		return nil, fmt.Errorf("incomplete DSN: Invalid Slack token format. Must start with xoxb-, xoxp-, or xoxa-2. DSN: %s", dsn.GetOriginalDSN())
	}

	channel := dsn.GetOption("channel")
	host := dsn.GetHost()
	if host == "default" {
		host = ""
	}
	port := dsn.GetPort()

	transport := NewTransport(accessToken, channel, f.client)
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
	return []string{"slack"}
}
