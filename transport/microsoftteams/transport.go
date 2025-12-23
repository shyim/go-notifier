package microsoftteams

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/shyim/go-notifier"
)

// Transport sends messages via Microsoft Teams Webhook API.
type Transport struct {
	*notifier.AbstractTransport
	webhookURL string
}

// NewTransport creates a new Microsoft Teams transport.
func NewTransport(webhookURL string, client *http.Client) *Transport {
	if client == nil {
		client = http.DefaultClient
	}
	return &Transport{
		AbstractTransport: notifier.NewAbstractTransport(client),
		webhookURL:        webhookURL,
	}
}

func (t *Transport) String() string {
	endpoint := t.getEndpoint()
	return fmt.Sprintf("microsoftteams://%s", endpoint)
}

func (t *Transport) Supports(message notifier.MessageInterface) bool {
	_, ok := message.(*notifier.ChatMessage)
	return ok
}

func (t *Transport) Send(ctx context.Context, message notifier.MessageInterface) (*notifier.SentMessage, error) {
	chatMsg, ok := message.(*notifier.ChatMessage)
	if !ok {
		return nil, fmt.Errorf("microsoftteams: unsupported message type %T, expected ChatMessage", message)
	}

	options := make(map[string]any)
	if opts, ok := chatMsg.GetOptions("microsoftteams").(*Options); ok {
		options = opts.ToMap()
	}

	// Teams expects "text" field for simple messages
	// If no theme color or title is set, use simple text format
	if _, hasTitle := options["title"]; !hasTitle {
		options["text"] = chatMsg.GetSubject()
	} else {
		// Use MessageCard format for rich messages
		sections := []map[string]any{
			{
				"activityTitle":    chatMsg.GetSubject(),
				"activitySubtitle": options["subtitle"],
				"activityText":     options["text"],
			},
		}

		// Remove individual fields and use sections
		delete(options, "subtitle")
		delete(options, "text")

		options["sections"] = sections
	}

	// Filter out empty values
	filteredOptions := make(map[string]any)
	for k, v := range options {
		if !isEmptyValue(v) {
			filteredOptions[k] = v
		}
	}

	jsonBody, err := json.Marshal(filteredOptions)
	if err != nil {
		return nil, fmt.Errorf("microsoftteams: marshal options: %w", err)
	}

	endpoint := t.webhookURL
	if endpoint == "" {
		endpoint = t.getEndpoint()
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("microsoftteams: create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := t.AbstractTransport.GetClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("microsoftteams: send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Teams returns 200 on success, but body is empty
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("microsoftteams: API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	sentMessage := notifier.NewSentMessage(message, t.String())
	return sentMessage, nil
}

func (t *Transport) getEndpoint() string {
	endpoint := t.GetEndpoint()
	if endpoint == "" || endpoint == "localhost" {
		return "webhook.office.com"
	}
	return endpoint
}

func isEmptyValue(v any) bool {
	switch val := v.(type) {
	case string:
		return val == ""
	case []any:
		return len(val) == 0
	case map[string]any:
		return len(val) == 0
	case nil:
		return true
	default:
		return false
	}
}
