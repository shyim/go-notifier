package gotify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/shyim/go-notifier"
)

// Transport sends messages via Gotify API.
type Transport struct {
	*notifier.AbstractTransport
	token string
}

// NewTransport creates a new Gotify transport.
func NewTransport(token string, client *http.Client) *Transport {
	if client == nil {
		client = http.DefaultClient
	}
	return &Transport{
		AbstractTransport: notifier.NewAbstractTransport(client),
		token:             token,
	}
}

func (t *Transport) String() string {
	endpoint := t.getEndpoint()
	return fmt.Sprintf("gotify://%s", endpoint)
}

func (t *Transport) Supports(message notifier.MessageInterface) bool {
	_, ok := message.(*notifier.ChatMessage)
	return ok
}

func (t *Transport) Send(ctx context.Context, message notifier.MessageInterface) (*notifier.SentMessage, error) {
	chatMsg, ok := message.(*notifier.ChatMessage)
	if !ok {
		return nil, fmt.Errorf("gotify: unsupported message type %T, expected ChatMessage", message)
	}

	options := make(map[string]any)
	if opts, ok := chatMsg.GetOptions("gotify").(*Options); ok {
		options = opts.ToMap()
	}

	// Gotify API expects title and message
	if _, ok := options["title"]; !ok {
		options["title"] = "Notification"
	}
	options["message"] = chatMsg.GetSubject()

	// Filter out empty values
	filteredOptions := make(map[string]any)
	for k, v := range options {
		if !isEmptyValue(v) {
			filteredOptions[k] = v
		}
	}

	jsonBody, err := json.Marshal(filteredOptions)
	if err != nil {
		return nil, fmt.Errorf("gotify: marshal options: %w", err)
	}

	endpoint := fmt.Sprintf("https://%s/message", t.getEndpoint())
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("gotify: create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Gotify-Key", t.token)

	resp, err := t.AbstractTransport.GetClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("gotify: send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gotify: API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		ID int `json:"id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("gotify: decode response: %w", err)
	}

	sentMessage := notifier.NewSentMessage(message, t.String())
	sentMessage.SetMessageID(fmt.Sprintf("%d", result.ID))
	sentMessage.SetInfo("priority", filteredOptions["priority"])
	sentMessage.SetInfo("title", filteredOptions["title"])

	return sentMessage, nil
}

func (t *Transport) getEndpoint() string {
	endpoint := t.GetEndpoint()
	if endpoint == "" || endpoint == "localhost" {
		// No default - Gotify is self-hosted and requires a real host
		// This will result in an invalid URL that will fail clearly
		return "gotify-server-required.com"
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
