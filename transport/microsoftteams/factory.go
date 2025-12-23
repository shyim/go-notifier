package microsoftteams

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/shyim/go-notifier"
)

func init() {
	notifier.RegisterTransportFactory(NewTransportFactory(nil))
}

// TransportFactory creates Microsoft Teams transports from DSN.
type TransportFactory struct {
	client *http.Client
}

// NewTransportFactory creates a new Microsoft Teams transport factory.
func NewTransportFactory(client *http.Client) *TransportFactory {
	if client == nil {
		client = http.DefaultClient
	}
	return &TransportFactory{
		client: client,
	}
}

// Create creates a Microsoft Teams transport from a DSN.
// DSN format: microsoftteams://<webhook_id>@default?token=<token>
// Example: microsoftteams://abc123@default?token=def456/ghi789
//
// To get the webhook_id and token from your Teams webhook URL:
// https://outlook.office.com/webhook/abc123/IncomingWebhook/def456/ghi789
// webhook_id = abc123
// token = def456/ghi789
func (f *TransportFactory) Create(dsn *notifier.DSN) (notifier.TransportInterface, error) {
	scheme := dsn.GetScheme()
	if scheme != "microsoftteams" {
		return nil, fmt.Errorf("unsupported scheme: scheme \"%s\" not supported (supported: %s). DSN: %s", scheme, strings.Join(f.GetSupportedSchemes(), ", "), dsn.GetOriginalDSN())
	}

	webhookID := dsn.GetUser()
	if webhookID == "" {
		return nil, fmt.Errorf("incomplete DSN: Missing webhook ID. DSN: %s", dsn.GetOriginalDSN())
	}

	token := dsn.GetOption("token")
	if token == "" {
		return nil, fmt.Errorf("missing required option: token")
	}

	// Construct the full webhook URL
	webhookURL := fmt.Sprintf("https://outlook.office.com/webhook/%s/IncomingWebhook/%s", webhookID, token)

	transport := NewTransport(webhookURL, f.client)

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
	return []string{"microsoftteams"}
}
