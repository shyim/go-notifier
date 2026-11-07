package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/shyim/go-notifier"
)

// Transport sends messages via Slack API.
type Transport struct {
	*notifier.AbstractTransport
	accessToken string
	channel     string
	webhookURL  string
}

// NewWebhookTransport creates a Slack incoming webhook transport.
// Messages use the channel configured for the webhook and the "slack" options key.
func NewWebhookTransport(webhookURL string, client *http.Client) *Transport {
	return &Transport{
		AbstractTransport: notifier.NewAbstractTransport(client),
		webhookURL:        webhookURL,
	}
}

// NewTransport creates a new Slack transport.
func NewTransport(accessToken, channel string, client *http.Client) *Transport {
	if client == nil {
		client = http.DefaultClient
	}

	return &Transport{
		AbstractTransport: notifier.NewAbstractTransport(client),
		accessToken:       accessToken,
		channel:           channel,
	}
}

func (t *Transport) String() string {
	if t.webhookURL != "" {
		// The webhook path contains credentials; never include it in the transport name.
		u, err := url.Parse(t.webhookURL)
		if err != nil {
			return "slack://default?webhook=true"
		}
		return fmt.Sprintf("slack://%s?webhook=true", u.Host)
	}
	endpoint := t.getEndpoint()
	query := ""
	if t.channel != "" {
		query = fmt.Sprintf("?channel=%s", t.channel)
	}
	return fmt.Sprintf("slack://%s%s", endpoint, query)
}

func (t *Transport) Supports(message notifier.MessageInterface) bool {
	_, ok := message.(*notifier.ChatMessage)
	return ok
}

func (t *Transport) Send(ctx context.Context, message notifier.MessageInterface) (*notifier.SentMessage, error) {
	chatMsg, ok := message.(*notifier.ChatMessage)
	if !ok {
		return nil, fmt.Errorf("slack: unsupported message type %T, expected ChatMessage", message)
	}

	chatID := chatMsg.GetRecipientId()
	if chatID == "" && t.channel != "" {
		chatID = t.channel
	}

	options := make(map[string]any)
	if opts, ok := chatMsg.GetOptions("slack").(*Options); ok {
		for key, value := range opts.ToMap() {
			options[key] = value
		}
	}

	options["channel"] = chatID
	options["text"] = chatMsg.GetSubject()
	if t.webhookURL != "" {
		for _, key := range []string{"post_at", "ts"} {
			if _, exists := options[key]; exists {
				return nil, fmt.Errorf("slack: incoming webhooks do not support %s", key)
			}
		}
		for _, key := range []string{"channel", "recipient_id", "username", "icon_emoji", "icon_url", "as_user"} {
			delete(options, key)
		}
	}

	// Determine API method
	apiMethod := "chat.postMessage"
	if _, ok := options["ts"]; ok {
		apiMethod = "chat.update"
	}
	if _, ok := options["post_at"]; ok {
		apiMethod = "chat.scheduleMessage"
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
		return nil, fmt.Errorf("slack: marshal options: %w", err)
	}

	endpoint := fmt.Sprintf("https://%s/api/%s", t.getEndpoint(), apiMethod)
	if t.webhookURL != "" {
		endpoint = t.webhookURL
	}
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("slack: create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	if t.webhookURL == "" {
		req.Header.Set("Authorization", "Bearer "+t.accessToken)
	}

	resp, err := t.AbstractTransport.GetClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("slack: send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if t.webhookURL != "" {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("slack: read webhook response: %w", err)
		}
		if resp.StatusCode != http.StatusOK || strings.TrimSpace(string(body)) != "ok" {
			return nil, fmt.Errorf("slack: webhook error (status %d): %s", resp.StatusCode, body)
		}
		return notifier.NewSentMessage(message, t.String()), nil
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("slack: API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		OK      bool   `json:"ok"`
		Channel string `json:"channel"`
		TS      string `json:"ts"`
		Error   string `json:"error"`
		Errors  string `json:"errors"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("slack: decode response: %w", err)
	}

	if !result.OK {
		errMsg := result.Error
		if result.Errors != "" {
			errMsg += " (" + result.Errors + ")"
		}
		return nil, fmt.Errorf("slack: %s", errMsg)
	}

	sentMessage := notifier.NewSentMessage(message, t.String())
	sentMessage.SetMessageID(result.TS)
	sentMessage.SetInfo("channel_id", result.Channel)

	return sentMessage, nil
}

func (t *Transport) getEndpoint() string {
	endpoint := t.GetEndpoint()
	if endpoint == "" || endpoint == "localhost" {
		return "slack.com"
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
