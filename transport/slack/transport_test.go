package slack

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/shyim/go-notifier"
)

func TestTransportSupports(t *testing.T) {
	transport := NewTransport("xoxb-test-token", "", nil)

	// Should support ChatMessage
	msg := notifier.NewChatMessage("Hello")
	if !transport.Supports(msg) {
		t.Error("Transport should support ChatMessage")
	}

	// Should support ChatMessage with SlackOptions
	opts := NewOptions().Username("bot")
	msgWithOpts := notifier.NewChatMessage("Hello").WithOptions("slack", opts)
	if !transport.Supports(msgWithOpts) {
		t.Error("Transport should support ChatMessage with SlackOptions")
	}
}

func TestTransportString(t *testing.T) {
	tests := []struct {
		token    string
		channel  string
		expected string
	}{
		{"xoxb-test", "", "slack://slack.com"},
		{"xoxb-test", "C123", "slack://slack.com?channel=C123"},
	}

	for _, tt := range tests {
		transport := NewTransport(tt.token, tt.channel, nil)
		if transport.String() != tt.expected {
			t.Errorf("Expected %s, got %s", tt.expected, transport.String())
		}
	}
}

func TestOptions(t *testing.T) {
	opts := NewOptions().
		Recipient("C123").
		Username("MyBot").
		IconEmoji(":robot_face:").
		AsUser(true).
		Mrkdwn(true).
		Parse("full")

	m := opts.ToMap()
	if m["recipient_id"] != "C123" {
		t.Error("Recipient not set correctly")
	}
	if m["username"] != "MyBot" {
		t.Error("Username not set correctly")
	}
	if m["icon_emoji"] != ":robot_face:" {
		t.Error("IconEmoji not set correctly")
	}
	if m["as_user"] != true {
		t.Error("AsUser not set correctly")
	}
	if m["mrkdwn"] != true {
		t.Error("Mrkdwn not set correctly")
	}
	if m["parse"] != "full" {
		t.Error("Parse not set correctly")
	}
}

func TestSectionBlock(t *testing.T) {
	block := NewSectionBlock().
		Text("Hello World", true).
		Field("Field 1", true).
		Field("Field 2", false)

	m := block.ToMap()
	if m["type"] != "section" {
		t.Error("Block type not set")
	}
	if m["text"] == nil {
		t.Error("Text not set")
	}
	if m["fields"] == nil {
		t.Error("Fields not set")
	}
}

func TestDividerBlock(t *testing.T) {
	block := NewDividerBlock()
	m := block.ToMap()
	if m["type"] != "divider" {
		t.Error("Divider type not set")
	}
}

func TestContextBlock(t *testing.T) {
	block := NewContextBlock().
		Elements(
			NewImageElement("https://example.com/image.png", "Alt text"),
		)

	m := block.ToMap()
	if m["type"] != "context" {
		t.Error("Context type not set")
	}
	if m["elements"] == nil {
		t.Error("Elements not set")
	}
}

func TestImageBlock(t *testing.T) {
	block := NewImageBlock("https://example.com/image.png", "Alt text")
	m := block.ToMap()
	if m["type"] != "image" {
		t.Error("Image type not set")
	}
	if m["image_url"] != "https://example.com/image.png" {
		t.Error("Image URL not set")
	}
}

func TestHeaderBlock(t *testing.T) {
	block := NewHeaderBlock("Header Text")
	m := block.ToMap()
	if m["type"] != "header" {
		t.Error("Header type not set")
	}
}

func TestActionsBlock(t *testing.T) {
	block := NewActionsBlock().Elements(
		NewButtonElement("Click Me").ActionId("button_1").Url("https://example.com"),
	)
	m := block.ToMap()
	if m["type"] != "actions" {
		t.Error("Actions type not set")
	}
	if m["elements"] == nil {
		t.Error("Elements not set")
	}
}

func TestButtonElement(t *testing.T) {
	elem := NewButtonElement("Approve").
		ActionId("approve_btn").
		Value("yes").
		Style("primary").
		Url("https://example.com")

	m := elem.ToMap()
	if m["type"] != "button" {
		t.Error("Button type not set")
	}
	if m["action_id"] != "approve_btn" {
		t.Error("ActionId not set")
	}
	if m["value"] != "yes" {
		t.Error("Value not set")
	}
	if m["style"] != "primary" {
		t.Error("Style not set")
	}
	if m["url"] != "https://example.com" {
		t.Error("URL not set")
	}
}

func TestImageElement(t *testing.T) {
	elem := NewImageElement("https://example.com/icon.png", "Icon")
	m := elem.ToMap()
	if m["type"] != "image" {
		t.Error("Image element type not set")
	}
}

func TestUpdateMessageOptions(t *testing.T) {
	opts := NewUpdateMessageOptions("C123", "1234567890.123456")
	m := opts.ToMap()
	if m["channel"] != "C123" {
		t.Error("Channel not set")
	}
	if m["ts"] != "1234567890.123456" {
		t.Error("TS not set")
	}
}

func TestDSN(t *testing.T) {
	dsn, err := notifier.NewDSN("slack://xoxb-test-token@default?channel=C123")
	if err != nil {
		t.Fatalf("Failed to parse DSN: %v", err)
	}

	if dsn.GetScheme() != "slack" {
		t.Errorf("Expected scheme 'slack', got '%s'", dsn.GetScheme())
	}
	if dsn.GetUser() != "xoxb-test-token" {
		t.Errorf("Expected user 'xoxb-test-token', got '%s'", dsn.GetUser())
	}
	if dsn.GetOption("channel") != "C123" {
		t.Errorf("Expected channel 'C123', got '%s'", dsn.GetOption("channel"))
	}
}

func TestFactory(t *testing.T) {
	factory := NewTransportFactory(nil)
	dsn, _ := notifier.NewDSN("slack://xoxb-1234567890-1234567890123-abcdefghijklmnopqrstuvwx@default?channel=C123")

	if !factory.Supports(dsn) {
		t.Error("Factory should support slack DSN")
	}

	transport, err := factory.Create(dsn)
	if err != nil {
		t.Fatalf("Failed to create transport: %v", err)
	}

	if transport == nil {
		t.Fatal("Transport is nil")
	}

	slackTransport, ok := transport.(*Transport)
	if !ok {
		t.Fatal("Transport is not a Slack transport")
	}

	if slackTransport.accessToken != "xoxb-1234567890-1234567890123-abcdefghijklmnopqrstuvwx" {
		t.Errorf("Token mismatch: %s", slackTransport.accessToken)
	}
	if slackTransport.channel != "C123" {
		t.Errorf("Channel mismatch: %s", slackTransport.channel)
	}
}

func TestInvalidToken(t *testing.T) {
	// NewTransport no longer panics - validation moved to factory
	// Test that factory returns error for invalid token
	factory := NewTransportFactory(nil)
	dsn, _ := notifier.NewDSN("slack://invalid-token@default")

	_, err := factory.Create(dsn)
	if err == nil {
		t.Error("Factory should return error for invalid token")
	}
}

func TestValidToken(t *testing.T) {
	factory := NewTransportFactory(nil)
	dsn, _ := notifier.NewDSN("slack://xoxb-1234567890-1234567890123-abcdefghijklmnopqrstuvwx@default")

	transport, err := factory.Create(dsn)
	if err != nil {
		t.Fatalf("Factory should accept valid token, got error: %v", err)
	}
	if transport == nil {
		t.Fatal("Transport should not be nil")
	}
}

func TestOptionsWithBlocks(t *testing.T) {
	opts := NewOptions().
		Block(NewSectionBlock().Text("Hello")).
		Block(NewDividerBlock()).
		Block(NewHeaderBlock("Title"))

	m := opts.ToMap()
	blocks, ok := m["blocks"].([]map[string]any)
	if !ok {
		t.Error("Blocks not stored correctly")
	}
	if len(blocks) != 3 {
		t.Errorf("Expected 3 blocks, got %d", len(blocks))
	}
}

// HTTP Client Tests

// mockRoundTripper is a custom RoundTripper for mocking HTTP requests
type mockRoundTripper struct {
	handler func(req *http.Request) (*http.Response, error)
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.handler(req)
}

// Helper function to create a mock HTTP client
func newMockClient(handler func(req *http.Request) (*http.Response, error)) *http.Client {
	return &http.Client{
		Transport: &mockRoundTripper{handler: handler},
	}
}

// Helper function to create a successful Slack API response
func createSuccessResponse() *http.Response {
	body := map[string]any{
		"ok":      true,
		"channel": "C123",
		"ts":      "1234567890.123456",
	}
	jsonBody, _ := json.Marshal(body)
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(string(jsonBody))),
		Header:     make(http.Header),
	}
}

func TestHTTPClientSuccessfulPostMessage(t *testing.T) {
	var capturedRequest *http.Request
	client := newMockClient(func(req *http.Request) (*http.Response, error) {
		capturedRequest = req
		return createSuccessResponse(), nil
	})

	transport := NewTransport("xoxb-test-token", "C123", client)
	msg := notifier.NewChatMessage("Hello, Slack!")

	sentMsg, err := transport.Send(context.Background(), msg)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify sent message
	if sentMsg == nil {
		t.Fatal("Expected SentMessage, got nil")
	}
	if sentMsg.GetMessageID() != "1234567890.123456" {
		t.Errorf("Expected message ID '1234567890.123456', got '%s'", sentMsg.GetMessageID())
	}
	if sentMsg.GetInfo("channel_id") != "C123" {
		t.Errorf("Expected channel_id 'C123', got '%s'", sentMsg.GetInfo("channel_id"))
	}

	// Verify request headers
	if capturedRequest.Header.Get("Content-Type") != "application/json; charset=utf-8" {
		t.Errorf("Expected Content-Type 'application/json; charset=utf-8', got '%s'",
			capturedRequest.Header.Get("Content-Type"))
	}
	if capturedRequest.Header.Get("Authorization") != "Bearer xoxb-test-token" {
		t.Errorf("Expected Authorization 'Bearer xoxb-test-token', got '%s'",
			capturedRequest.Header.Get("Authorization"))
	}

	// Verify request URL
	expectedURL := "https://slack.com/api/chat.postMessage"
	if capturedRequest.URL.String() != expectedURL {
		t.Errorf("Expected URL '%s', got '%s'", expectedURL, capturedRequest.URL.String())
	}

	// Verify request method
	if capturedRequest.Method != "POST" {
		t.Errorf("Expected POST method, got '%s'", capturedRequest.Method)
	}

	// Verify request body
	bodyBytes, _ := io.ReadAll(capturedRequest.Body)
	var body map[string]any
	json.Unmarshal(bodyBytes, &body)
	if body["channel"] != "C123" {
		t.Errorf("Expected channel 'C123', got '%v'", body["channel"])
	}
	if body["text"] != "Hello, Slack!" {
		t.Errorf("Expected text 'Hello, Slack!', got '%v'", body["text"])
	}
}

func TestHTTPClientSuccessfulUpdateMessage(t *testing.T) {
	var capturedRequest *http.Request
	var capturedBody map[string]any

	client := newMockClient(func(req *http.Request) (*http.Response, error) {
		capturedRequest = req
		bodyBytes, _ := io.ReadAll(req.Body)
		json.Unmarshal(bodyBytes, &capturedBody)
		return createSuccessResponse(), nil
	})

	transport := NewTransport("xoxb-test-token", "C123", client)

	// Use regular Options and set ts field directly
	opts := NewOptions()
	opts.options["ts"] = "1234567890.123456"
	msg := notifier.NewChatMessage("Updated message").WithOptions("slack", opts)

	sentMsg, err := transport.Send(context.Background(), msg)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if sentMsg == nil {
		t.Fatal("Expected SentMessage, got nil")
	}

	// Verify endpoint is chat.update
	expectedURL := "https://slack.com/api/chat.update"
	if capturedRequest.URL.String() != expectedURL {
		t.Errorf("Expected URL '%s', got '%s'", expectedURL, capturedRequest.URL.String())
	}

	// Verify request body contains ts
	if capturedBody["ts"] != "1234567890.123456" {
		t.Errorf("Expected ts '1234567890.123456', got '%v'", capturedBody["ts"])
	}
}

func TestHTTPClientSuccessfulScheduleMessage(t *testing.T) {
	var capturedRequest *http.Request
	client := newMockClient(func(req *http.Request) (*http.Response, error) {
		capturedRequest = req
		return createSuccessResponse(), nil
	})

	transport := NewTransport("xoxb-test-token", "C123", client)
	opts := NewOptions().PostAt(time.Now().Add(1 * time.Hour))
	msg := notifier.NewChatMessage("Scheduled message").WithOptions("slack", opts)

	sentMsg, err := transport.Send(context.Background(), msg)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if sentMsg == nil {
		t.Fatal("Expected SentMessage, got nil")
	}

	// Verify endpoint is chat.scheduleMessage
	expectedURL := "https://slack.com/api/chat.scheduleMessage"
	if capturedRequest.URL.String() != expectedURL {
		t.Errorf("Expected URL '%s', got '%s'", expectedURL, capturedRequest.URL.String())
	}

	// Verify request body contains post_at
	bodyBytes, _ := io.ReadAll(capturedRequest.Body)
	var body map[string]any
	json.Unmarshal(bodyBytes, &body)
	if _, ok := body["post_at"]; !ok {
		t.Error("Expected post_at in request body")
	}
}

func TestHTTPClientHTTPErrorResponses(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    string
	}{
		{
			name:       "401 Unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       `{"ok": false, "error": "invalid_auth"}`,
			wantErr:    "API error (status 401)",
		},
		{
			name:       "403 Forbidden",
			statusCode: http.StatusForbidden,
			body:       `{"ok": false, "error": "access_denied"}`,
			wantErr:    "API error (status 403)",
		},
		{
			name:       "404 Not Found",
			statusCode: http.StatusNotFound,
			body:       `{"ok": false, "error": "not_found"}`,
			wantErr:    "API error (status 404)",
		},
		{
			name:       "429 Rate Limited",
			statusCode: http.StatusTooManyRequests,
			body:       `{"ok": false, "error": "rate_limited"}`,
			wantErr:    "API error (status 429)",
		},
		{
			name:       "500 Internal Server Error",
			statusCode: http.StatusInternalServerError,
			body:       `{"ok": false, "error": "internal_error"}`,
			wantErr:    "API error (status 500)",
		},
		{
			name:       "503 Service Unavailable",
			statusCode: http.StatusServiceUnavailable,
			body:       `{"ok": false, "error": "service_unavailable"}`,
			wantErr:    "API error (status 503)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newMockClient(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: tt.statusCode,
					Body:       io.NopCloser(strings.NewReader(tt.body)),
					Header:     make(http.Header),
				}, nil
			})

			transport := NewTransport("xoxb-test-token", "C123", client)
			msg := notifier.NewChatMessage("Test message")

			_, err := transport.Send(context.Background(), msg)
			if err == nil {
				t.Fatal("Expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Expected error to contain '%s', got '%s'", tt.wantErr, err.Error())
			}
		})
	}
}

func TestHTTPClientNetworkErrors(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantErr string
	}{
		{
			name:    "Connection refused",
			err:     errors.New("connection refused"),
			wantErr: "send request",
		},
		{
			name:    "Timeout",
			err:     errors.New("timeout"),
			wantErr: "send request",
		},
		{
			name:    "DNS error",
			err:     errors.New("no such host"),
			wantErr: "send request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newMockClient(func(req *http.Request) (*http.Response, error) {
				return nil, tt.err
			})

			transport := NewTransport("xoxb-test-token", "C123", client)
			msg := notifier.NewChatMessage("Test message")

			_, err := transport.Send(context.Background(), msg)
			if err == nil {
				t.Fatal("Expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Expected error to contain '%s', got '%s'", tt.wantErr, err.Error())
			}
		})
	}
}

func TestHTTPClientInvalidJSONResponse(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr string
	}{
		{
			name:    "Malformed JSON",
			body:    `{"ok": true, "channel": "C123", "ts": `,
			wantErr: "decode response",
		},
		{
			name:    "Invalid JSON",
			body:    `not a json`,
			wantErr: "decode response",
		},
		{
			name:    "Empty response",
			body:    ``,
			wantErr: "decode response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newMockClient(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(tt.body)),
					Header:     make(http.Header),
				}, nil
			})

			transport := NewTransport("xoxb-test-token", "C123", client)
			msg := notifier.NewChatMessage("Test message")

			_, err := transport.Send(context.Background(), msg)
			if err == nil {
				t.Fatal("Expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Expected error to contain '%s', got '%s'", tt.wantErr, err.Error())
			}
		})
	}
}

func TestHTTPClientSlackAPIErrors(t *testing.T) {
	tests := []struct {
		name    string
		body    map[string]any
		wantErr string
	}{
		{
			name: "Invalid channel",
			body: map[string]any{
				"ok":    false,
				"error": "channel_not_found",
			},
			wantErr: "channel_not_found",
		},
		{
			name: "Invalid token",
			body: map[string]any{
				"ok":    false,
				"error": "invalid_auth",
			},
			wantErr: "invalid_auth",
		},
		{
			name: "Missing scope",
			body: map[string]any{
				"ok":    false,
				"error": "missing_scope",
			},
			wantErr: "missing_scope",
		},
		{
			name: "Error with errors field",
			body: map[string]any{
				"ok":     false,
				"error":  "invalid_blocks",
				"errors": "block[0].text is required",
			},
			wantErr: "invalid_blocks (block[0].text is required)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newMockClient(func(req *http.Request) (*http.Response, error) {
				jsonBody, _ := json.Marshal(tt.body)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(string(jsonBody))),
					Header:     make(http.Header),
				}, nil
			})

			transport := NewTransport("xoxb-test-token", "C123", client)
			msg := notifier.NewChatMessage("Test message")

			_, err := transport.Send(context.Background(), msg)
			if err == nil {
				t.Fatal("Expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Expected error to contain '%s', got '%s'", tt.wantErr, err.Error())
			}
		})
	}
}

func TestHTTPClientContextCancellation(t *testing.T) {
	client := newMockClient(func(req *http.Request) (*http.Response, error) {
		// Check if context is already canceled
		select {
		case <-req.Context().Done():
			return nil, req.Context().Err()
		default:
		}
		// Simulate a slow response
		time.Sleep(100 * time.Millisecond)
		// Check again after sleep
		select {
		case <-req.Context().Done():
			return nil, req.Context().Err()
		default:
		}
		return createSuccessResponse(), nil
	})

	transport := NewTransport("xoxb-test-token", "C123", client)
	msg := notifier.NewChatMessage("Test message")

	// Create a context that cancels immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := transport.Send(ctx, msg)
	if err == nil {
		t.Fatal("Expected error due to context cancellation, got nil")
	}
	if !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("Expected 'context canceled' error, got '%s'", err.Error())
	}
}

func TestHTTPClientContextTimeout(t *testing.T) {
	client := newMockClient(func(req *http.Request) (*http.Response, error) {
		// Simulate a slow response with context checking
		select {
		case <-time.After(200 * time.Millisecond):
			return createSuccessResponse(), nil
		case <-req.Context().Done():
			return nil, req.Context().Err()
		}
	})

	transport := NewTransport("xoxb-test-token", "C123", client)
	msg := notifier.NewChatMessage("Test message")

	// Create a context with a short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := transport.Send(ctx, msg)
	if err == nil {
		t.Fatal("Expected error due to context timeout, got nil")
	}
	if !strings.Contains(err.Error(), "deadline exceeded") && !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("Expected 'deadline exceeded' error, got '%s'", err.Error())
	}
}

func TestHTTPClientWithHTTPTestServer(t *testing.T) {
	// This test demonstrates using a mock RoundTripper pattern
	// which is equivalent to httptest.NewServer but more flexible
	var capturedRequest *http.Request
	client := newMockClient(func(req *http.Request) (*http.Response, error) {
		capturedRequest = req
		return createSuccessResponse(), nil
	})

	transport := NewTransport("xoxb-test-token", "C123", client)
	msg := notifier.NewChatMessage("Hello from test server")

	sentMsg, err := transport.Send(context.Background(), msg)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if sentMsg == nil {
		t.Fatal("Expected SentMessage, got nil")
	}

	// Verify the request was properly formed
	if capturedRequest == nil {
		t.Fatal("Expected request to be captured")
	}

	if capturedRequest.Header.Get("Authorization") != "Bearer xoxb-test-token" {
		t.Errorf("Expected Authorization header 'Bearer xoxb-test-token', got '%s'",
			capturedRequest.Header.Get("Authorization"))
	}

	if capturedRequest.Header.Get("Content-Type") != "application/json; charset=utf-8" {
		t.Errorf("Expected Content-Type 'application/json; charset=utf-8', got '%s'",
			capturedRequest.Header.Get("Content-Type"))
	}
}

// errorReader is a custom reader that always fails
type errorReader struct{}

func (e errorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("read error")
}

func (e errorReader) Close() error {
	return nil
}

func TestHTTPClientResponseBodyReadError(t *testing.T) {
	client := newMockClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       errorReader{},
			Header:     make(http.Header),
		}, nil
	})

	transport := NewTransport("xoxb-test-token", "C123", client)
	msg := notifier.NewChatMessage("Test message")

	_, err := transport.Send(context.Background(), msg)
	if err == nil {
		t.Fatal("Expected error due to body read error, got nil")
	}
	if !strings.Contains(err.Error(), "decode response") {
		t.Errorf("Expected 'decode response' error, got '%s'", err.Error())
	}
}

func TestHTTPClientRequestBodyConstruction(t *testing.T) {
	tests := []struct {
		name         string
		message      *notifier.ChatMessage
		expectedBody map[string]any
	}{
		{
			name:    "Basic message",
			message: notifier.NewChatMessage("Hello"),
			expectedBody: map[string]any{
				"channel": "C123",
				"text":    "Hello",
			},
		},
		{
			name: "Message with custom channel",
			message: func() *notifier.ChatMessage {
				opts := NewOptions().Recipient("C456")
				msg := notifier.NewChatMessage("Hello")
				return msg.WithOptions("slack", opts)
			}(),
			expectedBody: map[string]any{
				"channel": "C456",
				"text":    "Hello",
			},
		},
		{
			name: "Message with options",
			message: func() *notifier.ChatMessage {
				opts := NewOptions().Username("TestBot").IconEmoji(":robot_face:")
				msg := notifier.NewChatMessage("Hello")
				return msg.WithOptions("slack", opts)
			}(),
			expectedBody: map[string]any{
				"channel":    "C123",
				"text":       "Hello",
				"username":   "TestBot",
				"icon_emoji": ":robot_face:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedBody map[string]any
			client := newMockClient(func(req *http.Request) (*http.Response, error) {
				bodyBytes, _ := io.ReadAll(req.Body)
				json.Unmarshal(bodyBytes, &capturedBody)
				return createSuccessResponse(), nil
			})

			transport := NewTransport("xoxb-test-token", "C123", client)
			_, err := transport.Send(context.Background(), tt.message)
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			for key, expectedValue := range tt.expectedBody {
				if capturedBody[key] != expectedValue {
					t.Errorf("Expected %s to be '%v', got '%v'", key, expectedValue, capturedBody[key])
				}
			}
		})
	}
}

// unsupportedMessage is a mock message that's not a ChatMessage
type unsupportedMessage struct{}

func (m *unsupportedMessage) GetSubject() string                                     { return "" }
func (m *unsupportedMessage) GetOptions(key string) notifier.MessageOptionsInterface { return nil }
func (m *unsupportedMessage) GetRecipientId() string                                 { return "" }
func (m *unsupportedMessage) GetTransport() string                                   { return "" }

func TestHTTPClientUnsupportedMessageType(t *testing.T) {
	client := newMockClient(func(req *http.Request) (*http.Response, error) {
		t.Error("HTTP request should not be made for unsupported message type")
		return createSuccessResponse(), nil
	})

	transport := NewTransport("xoxb-test-token", "C123", client)

	msg := &unsupportedMessage{}
	_, err := transport.Send(context.Background(), msg)
	if err == nil {
		t.Fatal("Expected error for unsupported message type, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported message type") {
		t.Errorf("Expected 'unsupported message type' error, got '%s'", err.Error())
	}
}

func TestHTTPClientEndpointConstruction(t *testing.T) {
	tests := []struct {
		name            string
		setupTransport  func() *Transport
		expectedPattern string
	}{
		{
			name: "Default endpoint",
			setupTransport: func() *Transport {
				return NewTransport("xoxb-test-token", "C123", nil)
			},
			expectedPattern: "https://slack.com/api/chat.postMessage",
		},
		{
			name: "Custom endpoint",
			setupTransport: func() *Transport {
				t := NewTransport("xoxb-test-token", "C123", nil)
				t.SetHost("custom.slack.com")
				return t
			},
			expectedPattern: "https://custom.slack.com/api/chat.postMessage",
		},
		{
			name: "Custom endpoint with port",
			setupTransport: func() *Transport {
				t := NewTransport("xoxb-test-token", "C123", nil)
				t.SetHost("custom.slack.com")
				t.SetPort(8080)
				return t
			},
			expectedPattern: "https://custom.slack.com:8080/api/chat.postMessage",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedURL string
			client := newMockClient(func(req *http.Request) (*http.Response, error) {
				capturedURL = req.URL.String()
				return createSuccessResponse(), nil
			})

			transport := tt.setupTransport()
			endpoint := transport.GetEndpoint()
			transport.AbstractTransport = notifier.NewAbstractTransport(client).
				SetHost(endpoint)

			msg := notifier.NewChatMessage("Test")
			_, err := transport.Send(context.Background(), msg)
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			if capturedURL != tt.expectedPattern {
				t.Errorf("Expected URL '%s', got '%s'", tt.expectedPattern, capturedURL)
			}
		})
	}
}
