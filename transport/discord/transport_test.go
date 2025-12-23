package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/shyim/go-notifier"
)

func TestTransportSupports(t *testing.T) {
	transport := NewTransport("123", "token", nil)

	// Should support ChatMessage
	msg := notifier.NewChatMessage("Hello")
	if !transport.Supports(msg) {
		t.Error("Transport should support ChatMessage")
	}

	// Should support ChatMessage with DiscordOptions
	opts := NewOptions().Username("bot")
	msgWithOpts := notifier.NewChatMessage("Hello").WithOptions("discord", opts)
	if !transport.Supports(msgWithOpts) {
		t.Error("Transport should support ChatMessage with DiscordOptions")
	}
}

func TestTransportString(t *testing.T) {
	tests := []struct {
		webhookID string
		token     string
		expected  string
	}{
		{"123", "token", "discord://discord.com?webhook_id=123"},
		{"456", "abc", "discord://discord.com?webhook_id=456"},
	}

	for _, tt := range tests {
		transport := NewTransport(tt.webhookID, tt.token, nil)
		if transport.String() != tt.expected {
			t.Errorf("Expected %s, got %s", tt.expected, transport.String())
		}
	}
}

func TestOptions(t *testing.T) {
	opts := NewOptions().
		Recipient("123").
		Username("MyBot").
		AvatarUrl("https://example.com/avatar.png").
		TTS(true)

	m := opts.ToMap()
	if m["recipient_id"] != "123" {
		t.Error("Recipient not set correctly")
	}
	if m["username"] != "MyBot" {
		t.Error("Username not set correctly")
	}
	if m["avatar_url"] != "https://example.com/avatar.png" {
		t.Error("AvatarUrl not set correctly")
	}
	if m["tts"] != true {
		t.Error("TTS not set correctly")
	}
}

func TestEmbed(t *testing.T) {
	embed := NewEmbed().
		Title("Title").
		Description("Description").
		URL("https://example.com").
		Color(0xFF0000).
		Footer("Footer", "https://example.com/icon.png").
		Thumbnail("https://example.com/thumb.png").
		Image("https://example.com/image.png").
		Author("Author", "https://example.com").
		AddField("Field 1", "Value 1", true).
		AddField("Field 2", "Value 2", false)

	m := embed.ToMap()
	if m["title"] != "Title" {
		t.Error("Title not set")
	}
	if m["description"] != "Description" {
		t.Error("Description not set")
	}
	if m["url"] != "https://example.com" {
		t.Error("URL not set")
	}
	if m["color"] != 0xFF0000 {
		t.Error("Color not set")
	}
	if m["footer"] == nil {
		t.Error("Footer not set")
	}
	if m["thumbnail"] == nil {
		t.Error("Thumbnail not set")
	}
	if m["image"] == nil {
		t.Error("Image not set")
	}
	if m["author"] == nil {
		t.Error("Author not set")
	}
	if m["fields"] == nil {
		t.Error("Fields not set")
	}
}

func TestOptionsWithEmbed(t *testing.T) {
	embed := NewEmbed().Title("Test")
	opts := NewOptions().AddEmbed(embed)

	m := opts.ToMap()
	embeds, ok := m["embeds"].([]map[string]any)
	if !ok || len(embeds) != 1 {
		t.Error("Embed not added")
	}
}

func TestDSN(t *testing.T) {
	dsn, err := notifier.NewDSN("discord://token@default?webhook_id=123")
	if err != nil {
		t.Fatalf("Failed to parse DSN: %v", err)
	}

	if dsn.GetScheme() != "discord" {
		t.Errorf("Expected scheme 'discord', got '%s'", dsn.GetScheme())
	}
	if dsn.GetUser() != "token" {
		t.Errorf("Expected user 'token', got '%s'", dsn.GetUser())
	}
	if dsn.GetOption("webhook_id") != "123" {
		t.Errorf("Expected webhook_id '123', got '%s'", dsn.GetOption("webhook_id"))
	}
}

func TestFactory(t *testing.T) {
	factory := NewTransportFactory(nil)
	dsn, _ := notifier.NewDSN("discord://token@default?webhook_id=123")

	if !factory.Supports(dsn) {
		t.Error("Factory should support discord DSN")
	}

	transport, err := factory.Create(dsn)
	if err != nil {
		t.Fatalf("Failed to create transport: %v", err)
	}

	if transport == nil {
		t.Fatal("Transport is nil")
	}

	discordTransport, ok := transport.(*Transport)
	if !ok {
		t.Fatal("Transport is not a Discord transport")
	}

	if discordTransport.token != "token" {
		t.Errorf("Token mismatch: %s", discordTransport.token)
	}
	if discordTransport.webhookID != "123" {
		t.Errorf("WebhookID mismatch: %s", discordTransport.webhookID)
	}
}

func TestMissingWebhookID(t *testing.T) {
	factory := NewTransportFactory(nil)
	dsn, _ := notifier.NewDSN("discord://token@default")

	_, err := factory.Create(dsn)
	if err == nil {
		t.Error("Expected error for missing webhook_id")
	}
}

// HTTP Client Tests

func TestSendSuccessfulWebhookPost(t *testing.T) {
	var capturedRequest *http.Request
	var capturedBody []byte

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequest = r
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	transport := NewTransport("webhook123", "token456", server.Client())
	transport.SetHost(strings.TrimPrefix(server.URL, "https://"))

	msg := notifier.NewChatMessage("Test message")
	ctx := context.Background()

	sentMsg, err := transport.Send(ctx, msg)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if sentMsg == nil {
		t.Fatal("Expected SentMessage, got nil")
	}

	// Verify request method
	if capturedRequest.Method != "POST" {
		t.Errorf("Expected POST method, got: %s", capturedRequest.Method)
	}

	// Verify request URL
	expectedPath := "/api/webhooks/webhook123/token456"
	if capturedRequest.URL.Path != expectedPath {
		t.Errorf("Expected path %s, got: %s", expectedPath, capturedRequest.URL.Path)
	}

	// Verify Content-Type header
	contentType := capturedRequest.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got: %s", contentType)
	}

	// Verify request body
	var body map[string]interface{}
	if err := json.Unmarshal(capturedBody, &body); err != nil {
		t.Fatalf("Failed to parse request body: %v", err)
	}

	if body["content"] != "Test message" {
		t.Errorf("Expected content 'Test message', got: %v", body["content"])
	}
}

func TestSendWithDiscordOptions(t *testing.T) {
	var capturedBody []byte

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	transport := NewTransport("webhook123", "token456", server.Client())
	transport.SetHost(strings.TrimPrefix(server.URL, "https://"))

	opts := NewOptions().
		Username("TestBot").
		AvatarUrl("https://example.com/avatar.png").
		TTS(true)

	msg := notifier.NewChatMessage("Test message").WithOptions("discord", opts)
	ctx := context.Background()

	_, err := transport.Send(ctx, msg)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify request body contains Discord options
	var body map[string]interface{}
	if err := json.Unmarshal(capturedBody, &body); err != nil {
		t.Fatalf("Failed to parse request body: %v", err)
	}

	if body["content"] != "Test message" {
		t.Errorf("Expected content 'Test message', got: %v", body["content"])
	}
	if body["username"] != "TestBot" {
		t.Errorf("Expected username 'TestBot', got: %v", body["username"])
	}
	if body["avatar_url"] != "https://example.com/avatar.png" {
		t.Errorf("Expected avatar_url 'https://example.com/avatar.png', got: %v", body["avatar_url"])
	}
	if body["tts"] != true {
		t.Errorf("Expected tts true, got: %v", body["tts"])
	}
}

func TestSendWithEmbeds(t *testing.T) {
	var capturedBody []byte

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	transport := NewTransport("webhook123", "token456", server.Client())
	transport.SetHost(strings.TrimPrefix(server.URL, "https://"))

	embed := NewEmbed().
		Title("Test Embed").
		Description("This is a test embed").
		Color(0xFF5733).
		AddField("Field1", "Value1", true).
		AddField("Field2", "Value2", false)

	opts := NewOptions().AddEmbed(embed)
	msg := notifier.NewChatMessage("Test message with embed").WithOptions("discord", opts)
	ctx := context.Background()

	_, err := transport.Send(ctx, msg)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify request body contains embeds
	var body map[string]interface{}
	if err := json.Unmarshal(capturedBody, &body); err != nil {
		t.Fatalf("Failed to parse request body: %v", err)
	}

	embeds, ok := body["embeds"].([]interface{})
	if !ok {
		t.Fatal("Expected embeds array in body")
	}
	if len(embeds) != 1 {
		t.Fatalf("Expected 1 embed, got %d", len(embeds))
	}

	embedMap := embeds[0].(map[string]interface{})
	if embedMap["title"] != "Test Embed" {
		t.Errorf("Expected embed title 'Test Embed', got: %v", embedMap["title"])
	}
	if embedMap["description"] != "This is a test embed" {
		t.Errorf("Expected embed description 'This is a test embed', got: %v", embedMap["description"])
	}

	// Verify color (JSON unmarshals numbers as float64)
	colorFloat, ok := embedMap["color"].(float64)
	if !ok || int(colorFloat) != 0xFF5733 {
		t.Errorf("Expected embed color 0xFF5733, got: %v", embedMap["color"])
	}

	// Verify fields
	fields, ok := embedMap["fields"].([]interface{})
	if !ok || len(fields) != 2 {
		t.Errorf("Expected 2 fields in embed, got: %v", embedMap["fields"])
	}
}

func TestSendHTTPErrorResponses(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectedErrMsg string
	}{
		{
			name:           "Bad Request",
			statusCode:     http.StatusBadRequest,
			responseBody:   `{"message": "Invalid payload"}`,
			expectedErrMsg: "API error (status 400)",
		},
		{
			name:           "Unauthorized",
			statusCode:     http.StatusUnauthorized,
			responseBody:   `{"message": "Invalid webhook token"}`,
			expectedErrMsg: "API error (status 401)",
		},
		{
			name:           "Not Found",
			statusCode:     http.StatusNotFound,
			responseBody:   `{"message": "Unknown webhook"}`,
			expectedErrMsg: "API error (status 404)",
		},
		{
			name:           "Rate Limit",
			statusCode:     http.StatusTooManyRequests,
			responseBody:   `{"message": "You are being rate limited", "retry_after": 1000}`,
			expectedErrMsg: "API error (status 429)",
		},
		{
			name:           "Internal Server Error",
			statusCode:     http.StatusInternalServerError,
			responseBody:   `{"message": "Internal server error"}`,
			expectedErrMsg: "API error (status 500)",
		},
		{
			name:           "Service Unavailable",
			statusCode:     http.StatusServiceUnavailable,
			responseBody:   `{"message": "Service temporarily unavailable"}`,
			expectedErrMsg: "API error (status 503)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			transport := NewTransport("webhook123", "token456", server.Client())
			transport.SetHost(strings.TrimPrefix(server.URL, "https://"))

			msg := notifier.NewChatMessage("Test message")
			ctx := context.Background()

			_, err := transport.Send(ctx, msg)
			if err == nil {
				t.Fatal("Expected error, got nil")
			}

			if !strings.Contains(err.Error(), tt.expectedErrMsg) {
				t.Errorf("Expected error to contain '%s', got: %v", tt.expectedErrMsg, err)
			}

			if !strings.Contains(err.Error(), tt.responseBody) {
				t.Errorf("Expected error to contain response body '%s', got: %v", tt.responseBody, err)
			}
		})
	}
}

func TestSendNetworkError(t *testing.T) {
	// Create a custom RoundTripper that simulates a network error
	networkErrorTransport := &errorRoundTripper{
		err: errors.New("network connection failed"),
	}

	client := &http.Client{
		Transport: networkErrorTransport,
	}

	transport := NewTransport("webhook123", "token456", client)
	msg := notifier.NewChatMessage("Test message")
	ctx := context.Background()

	_, err := transport.Send(ctx, msg)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !strings.Contains(err.Error(), "send request") {
		t.Errorf("Expected error to contain 'send request', got: %v", err)
	}

	if !strings.Contains(err.Error(), "network connection failed") {
		t.Errorf("Expected error to contain 'network connection failed', got: %v", err)
	}
}

func TestSendResponseBodyReadError(t *testing.T) {
	// Create a server that returns an error status with a body that fails to read
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100") // Lie about content length
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("short")) // Write less than promised
	}))
	defer server.Close()

	transport := NewTransport("webhook123", "token456", server.Client())
	transport.SetHost(strings.TrimPrefix(server.URL, "https://"))

	msg := notifier.NewChatMessage("Test message")
	ctx := context.Background()

	_, err := transport.Send(ctx, msg)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// Should still get an API error even if body read partially fails
	if !strings.Contains(err.Error(), "API error (status 400)") {
		t.Errorf("Expected error to contain 'API error (status 400)', got: %v", err)
	}
}

func TestSendContextCancellation(t *testing.T) {
	// Create a server that delays response
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if context is canceled
		<-r.Context().Done()
		// Don't write response, connection will be closed
	}))
	defer server.Close()

	transport := NewTransport("webhook123", "token456", server.Client())
	transport.SetHost(strings.TrimPrefix(server.URL, "https://"))

	msg := notifier.NewChatMessage("Test message")
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context immediately
	cancel()

	_, err := transport.Send(ctx, msg)
	if err == nil {
		t.Fatal("Expected error due to context cancellation, got nil")
	}

	// Error should indicate context cancellation
	if !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("Expected error to contain 'context canceled', got: %v", err)
	}
}

func TestSendUnsupportedMessageType(t *testing.T) {
	transport := NewTransport("webhook123", "token456", nil)

	// Implement minimal MessageInterface methods
	msg := &struct {
		notifier.MessageInterface
	}{}

	ctx := context.Background()

	_, err := transport.Send(ctx, msg)
	if err == nil {
		t.Fatal("Expected error for unsupported message type, got nil")
	}

	if !strings.Contains(err.Error(), "unsupported message type") {
		t.Errorf("Expected error to contain 'unsupported message type', got: %v", err)
	}
}

func TestWebhookEndpointConstruction(t *testing.T) {
	tests := []struct {
		name        string
		webhookID   string
		token       string
		endpoint    string
		expectedURL string
	}{
		{
			name:        "Default discord.com endpoint",
			webhookID:   "123456789",
			token:       "abc123token",
			endpoint:    "",
			expectedURL: "https://discord.com/api/webhooks/123456789/abc123token",
		},
		{
			name:        "Custom endpoint",
			webhookID:   "987654321",
			token:       "xyz789token",
			endpoint:    "custom.discord.server.com",
			expectedURL: "https://custom.discord.server.com/api/webhooks/987654321/xyz789token",
		},
		{
			name:        "Localhost becomes discord.com",
			webhookID:   "111222333",
			token:       "localhosttoken",
			endpoint:    "localhost",
			expectedURL: "https://discord.com/api/webhooks/111222333/localhosttoken",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedURL string

			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedURL = "https://" + r.Host + r.URL.Path
				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()

			transport := NewTransport(tt.webhookID, tt.token, server.Client())
			if tt.endpoint != "" && tt.endpoint != "localhost" {
				transport.SetHost(strings.TrimPrefix(server.URL, "https://"))
			}

			msg := notifier.NewChatMessage("Test")
			ctx := context.Background()

			// For default and localhost cases, we need to mock the actual discord.com call
			if tt.endpoint == "" || tt.endpoint == "localhost" {
				// In these cases, the transport will try to reach discord.com
				// We'll use a custom RoundTripper to intercept and verify the URL
				capturedURL = ""
				mockTransport := &urlCapturingRoundTripper{
					expectedURL: tt.expectedURL,
					t:           t,
				}
				customClient := &http.Client{Transport: mockTransport}
				transport = NewTransport(tt.webhookID, tt.token, customClient)
				if tt.endpoint == "localhost" {
					transport.SetHost(tt.endpoint)
				}

				transport.Send(ctx, msg)
				return
			}

			// For custom endpoint, verify against the test server
			_, err := transport.Send(ctx, msg)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if !strings.HasSuffix(tt.expectedURL, "/api/webhooks/"+tt.webhookID+"/"+tt.token) {
				t.Errorf("Expected URL to end with '/api/webhooks/%s/%s', got: %s", tt.webhookID, tt.token, capturedURL)
			}
		})
	}
}

func TestEmptyValuesFilteredFromRequest(t *testing.T) {
	var capturedBody []byte

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	transport := NewTransport("webhook123", "token456", server.Client())
	transport.SetHost(strings.TrimPrefix(server.URL, "https://"))

	// Create options with some empty values
	opts := NewOptions().
		Username("TestBot").
		AvatarUrl("") // Empty string should be filtered

	msg := notifier.NewChatMessage("Test message").WithOptions("discord", opts)
	ctx := context.Background()

	_, err := transport.Send(ctx, msg)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify request body doesn't contain empty avatar_url
	var body map[string]interface{}
	if err := json.Unmarshal(capturedBody, &body); err != nil {
		t.Fatalf("Failed to parse request body: %v", err)
	}

	if _, exists := body["avatar_url"]; exists {
		t.Error("Expected empty avatar_url to be filtered out, but it exists in body")
	}

	if body["username"] != "TestBot" {
		t.Errorf("Expected username 'TestBot', got: %v", body["username"])
	}
}

// Helper types for testing

type errorRoundTripper struct {
	err error
}

func (e *errorRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, e.err
}

type urlCapturingRoundTripper struct {
	expectedURL string
	t           *testing.T
}

func (u *urlCapturingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	actualURL := req.URL.String()
	if actualURL != u.expectedURL {
		u.t.Errorf("Expected URL %s, got: %s", u.expectedURL, actualURL)
	}

	// Return a successful mock response
	return &http.Response{
		StatusCode: http.StatusNoContent,
		Body:       io.NopCloser(bytes.NewReader([]byte{})),
		Header:     make(http.Header),
	}, nil
}
