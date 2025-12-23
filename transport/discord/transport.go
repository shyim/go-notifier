package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/shyim/go-notifier"
)

// Transport sends messages via Discord Webhook API.
type Transport struct {
	*notifier.AbstractTransport
	webhookID string
	token     string
}

// NewTransport creates a new Discord transport.
func NewTransport(webhookID, token string, client *http.Client) *Transport {
	if client == nil {
		client = http.DefaultClient
	}
	return &Transport{
		AbstractTransport: notifier.NewAbstractTransport(client),
		webhookID:         webhookID,
		token:             token,
	}
}

func (t *Transport) String() string {
	endpoint := t.getEndpoint()
	return fmt.Sprintf("discord://%s?webhook_id=%s", endpoint, t.webhookID)
}

func (t *Transport) Supports(message notifier.MessageInterface) bool {
	_, ok := message.(*notifier.ChatMessage)
	return ok
}

func (t *Transport) Send(ctx context.Context, message notifier.MessageInterface) (*notifier.SentMessage, error) {
	chatMsg, ok := message.(*notifier.ChatMessage)
	if !ok {
		return nil, fmt.Errorf("discord: unsupported message type %T, expected ChatMessage", message)
	}

	options := make(map[string]any)
	if opts, ok := chatMsg.GetOptions("discord").(*Options); ok {
		options = opts.ToMap()
	}

	options["content"] = chatMsg.GetSubject()

	// Filter out empty values
	filteredOptions := make(map[string]any)
	for k, v := range options {
		if !isEmptyValue(v) {
			filteredOptions[k] = v
		}
	}

	jsonBody, err := json.Marshal(filteredOptions)
	if err != nil {
		return nil, fmt.Errorf("discord: marshal options: %w", err)
	}

	endpoint := fmt.Sprintf("https://%s/api/webhooks/%s/%s", t.getEndpoint(), t.webhookID, t.token)
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("discord: create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := t.AbstractTransport.GetClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("discord: send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Discord returns 204 on success
	if resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("discord: API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	sentMessage := notifier.NewSentMessage(message, t.String())
	return sentMessage, nil
}

func (t *Transport) getEndpoint() string {
	endpoint := t.GetEndpoint()
	if endpoint == "" || endpoint == "localhost" {
		return "discord.com"
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
