package gotify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/shyim/go-notifier"
)

// createTestTransport creates a transport configured for use with httptest.Server
// It uses a custom RoundTripper to replace https:// with http:// for test servers
func createTestTransport(token string, server *httptest.Server) *Transport {
	// Create a custom transport that rewrites HTTPS to HTTP for testing
	client := &http.Client{
		Transport: &testRoundTripper{
			serverURL: server.URL,
			base:      server.Client().Transport,
		},
	}

	transport := NewTransport(token, client)
	// Extract host from server URL (remove http:// prefix)
	serverHost := strings.TrimPrefix(server.URL, "http://")
	transport.SetHost(serverHost)
	return transport
}

// testRoundTripper rewrites HTTPS requests to HTTP for testing with httptest.Server
type testRoundTripper struct {
	serverURL string
	base      http.RoundTripper
}

func (t *testRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Rewrite https:// to http:// and use the test server URL
	if req.URL.Scheme == "https" {
		req.URL.Scheme = "http"
		// Replace the host with the test server's host
		req.URL.Host = strings.TrimPrefix(t.serverURL, "http://")
	}
	return t.base.RoundTrip(req)
}

func TestTransportSupports(t *testing.T) {
	transport := NewTransport("token", nil)

	// Should support ChatMessage
	msg := notifier.NewChatMessage("Hello")
	if !transport.Supports(msg) {
		t.Error("Transport should support ChatMessage")
	}

	// Should support ChatMessage with GotifyOptions
	opts := NewOptions().Title("Test")
	msgWithOpts := notifier.NewChatMessage("Hello").WithOptions("gotify", opts)
	if !transport.Supports(msgWithOpts) {
		t.Error("Transport should support ChatMessage with GotifyOptions")
	}
}

func TestTransportString(t *testing.T) {
	transport := NewTransport("token", nil)
	transport.SetHost("gotify.example.com")

	expected := "gotify://gotify.example.com"
	if transport.String() != expected {
		t.Errorf("Expected %s, got %s", expected, transport.String())
	}
}

// TestTransportSendSuccess tests successful HTTP POST with httptest.NewServer
func TestTransportSendSuccess(t *testing.T) {
	const expectedToken = "test-token-123"
	const expectedMessageID = 42

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify HTTP method
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		// Verify endpoint path
		if r.URL.Path != "/message" {
			t.Errorf("Expected /message endpoint, got %s", r.URL.Path)
		}

		// Verify headers
		if contentType := r.Header.Get("Content-Type"); contentType != "application/json" {
			t.Errorf("Expected Content-Type: application/json, got %s", contentType)
		}
		if token := r.Header.Get("X-Gotify-Key"); token != expectedToken {
			t.Errorf("Expected X-Gotify-Key: %s, got %s", expectedToken, token)
		}

		// Verify request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("Failed to read request body: %v", err)
		}

		var payload map[string]interface{}
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("Failed to unmarshal request body: %v", err)
		}

		// Verify required fields
		if message, ok := payload["message"].(string); !ok || message != "Test message" {
			t.Errorf("Expected message 'Test message', got %v", payload["message"])
		}
		if title, ok := payload["title"].(string); !ok || title != "Test Title" {
			t.Errorf("Expected title 'Test Title', got %v", payload["title"])
		}
		if priority, ok := payload["priority"].(float64); !ok || int(priority) != 8 {
			t.Errorf("Expected priority 8, got %v", payload["priority"])
		}

		// Verify extras
		if extras, ok := payload["extras"].(map[string]interface{}); ok {
			if customKey, ok := extras["custom_key"].(string); !ok || customKey != "custom_value" {
				t.Errorf("Expected extras.custom_key 'custom_value', got %v", extras["custom_key"])
			}
		} else {
			t.Error("Expected extras map in payload")
		}

		// Send success response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := fmt.Sprintf(`{"id": %d}`, expectedMessageID)
		w.Write([]byte(response))
	}))
	defer server.Close()

	transport := createTestTransport(expectedToken, server)

	opts := NewOptions().
		Title("Test Title").
		Priority(8).
		AddExtra("custom_key", "custom_value")

	msg := notifier.NewChatMessage("Test message").
		WithOptions("gotify", opts)

	sentMsg, err := transport.Send(context.Background(), msg)
	if err != nil {
		t.Fatalf("Expected successful send, got error: %v", err)
	}

	if sentMsg == nil {
		t.Fatal("Expected non-nil SentMessage")
	}

	if sentMsg.GetMessageID() != "42" {
		t.Errorf("Expected message ID '42', got '%s'", sentMsg.GetMessageID())
	}

	if priority := sentMsg.GetInfo("priority"); priority != 8 {
		t.Errorf("Expected priority info 8, got %v", priority)
	}

	if title := sentMsg.GetInfo("title"); title != "Test Title" {
		t.Errorf("Expected title info 'Test Title', got %v", title)
	}
}

// TestTransportSendHTTPErrors tests various HTTP error responses
func TestTransportSendHTTPErrors(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectedErrMsg string
	}{
		{
			name:           "401 Unauthorized",
			statusCode:     http.StatusUnauthorized,
			responseBody:   `{"error": "invalid token"}`,
			expectedErrMsg: "gotify: API error (status 401)",
		},
		{
			name:           "403 Forbidden",
			statusCode:     http.StatusForbidden,
			responseBody:   `{"error": "forbidden"}`,
			expectedErrMsg: "gotify: API error (status 403)",
		},
		{
			name:           "404 Not Found",
			statusCode:     http.StatusNotFound,
			responseBody:   `{"error": "not found"}`,
			expectedErrMsg: "gotify: API error (status 404)",
		},
		{
			name:           "500 Internal Server Error",
			statusCode:     http.StatusInternalServerError,
			responseBody:   `{"error": "internal server error"}`,
			expectedErrMsg: "gotify: API error (status 500)",
		},
		{
			name:           "503 Service Unavailable",
			statusCode:     http.StatusServiceUnavailable,
			responseBody:   "Service temporarily unavailable",
			expectedErrMsg: "gotify: API error (status 503)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			transport := createTestTransport("token", server)

			msg := notifier.NewChatMessage("Test message")
			_, err := transport.Send(context.Background(), msg)

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

// TestTransportSendNetworkError tests network-level errors using custom RoundTripper
func TestTransportSendNetworkError(t *testing.T) {
	tests := []struct {
		name              string
		roundTripper      http.RoundTripper
		expectedErrSubstr string
	}{
		{
			name: "Connection refused",
			roundTripper: &mockRoundTripper{
				err: errors.New("connection refused"),
			},
			expectedErrSubstr: "connection refused",
		},
		{
			name: "Timeout",
			roundTripper: &mockRoundTripper{
				err: errors.New("timeout"),
			},
			expectedErrSubstr: "timeout",
		},
		{
			name: "DNS resolution failure",
			roundTripper: &mockRoundTripper{
				err: errors.New("no such host"),
			},
			expectedErrSubstr: "no such host",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &http.Client{
				Transport: tt.roundTripper,
			}

			transport := NewTransport("token", client)
			transport.SetHost("gotify.example.com")

			msg := notifier.NewChatMessage("Test message")
			_, err := transport.Send(context.Background(), msg)

			if err == nil {
				t.Fatal("Expected error, got nil")
			}

			if !strings.Contains(err.Error(), "gotify: send request") {
				t.Errorf("Expected error to start with 'gotify: send request', got: %v", err)
			}

			if !strings.Contains(err.Error(), tt.expectedErrSubstr) {
				t.Errorf("Expected error to contain '%s', got: %v", tt.expectedErrSubstr, err)
			}
		})
	}
}

// TestTransportSendInvalidJSONResponse tests handling of invalid JSON responses
func TestTransportSendInvalidJSONResponse(t *testing.T) {
	tests := []struct {
		name         string
		responseBody string
		description  string
	}{
		{
			name:         "Malformed JSON",
			responseBody: `{"id": invalid}`,
			description:  "malformed JSON syntax",
		},
		{
			name:         "Non-JSON response",
			responseBody: `This is not JSON`,
			description:  "plain text instead of JSON",
		},
		{
			name:         "Empty response",
			responseBody: ``,
			description:  "empty response body",
		},
		{
			name:         "JSON with wrong id type",
			responseBody: `{"id": "not-a-number"}`,
			description:  "id field is string instead of int",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			transport := createTestTransport("token", server)

			msg := notifier.NewChatMessage("Test message")
			_, err := transport.Send(context.Background(), msg)

			if err == nil {
				t.Fatalf("Expected error for %s, got nil", tt.description)
			}

			if !strings.Contains(err.Error(), "gotify: decode response") {
				t.Errorf("Expected decode error, got: %v", err)
			}
		})
	}
}

// TestTransportSendJSONWithoutIDField tests handling of JSON response without id field
// Note: Go's JSON decoder sets missing int fields to 0, so this is a valid response
func TestTransportSendJSONWithoutIDField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	transport := createTestTransport("token", server)

	msg := notifier.NewChatMessage("Test message")
	sentMsg, err := transport.Send(context.Background(), msg)

	// This should succeed because missing id field defaults to 0
	if err != nil {
		t.Fatalf("Expected success for JSON without id field, got error: %v", err)
	}

	// Verify that the message ID is "0" (the zero value for int)
	if sentMsg.GetMessageID() != "0" {
		t.Errorf("Expected message ID '0' for missing id field, got '%s'", sentMsg.GetMessageID())
	}
}

// TestTransportSendResponseBodyReadError tests handling of body read errors
func TestTransportSendResponseBodyReadError(t *testing.T) {
	// Use a custom RoundTripper that returns a response with an error-producing body
	client := &http.Client{
		Transport: &mockRoundTripper{
			response: &http.Response{
				StatusCode: http.StatusOK,
				Body:       &errorReader{err: errors.New("read error")},
				Header:     make(http.Header),
			},
		},
	}

	transport := NewTransport("token", client)
	transport.SetHost("gotify.example.com")

	msg := notifier.NewChatMessage("Test message")
	_, err := transport.Send(context.Background(), msg)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !strings.Contains(err.Error(), "gotify: decode response") {
		t.Errorf("Expected decode error, got: %v", err)
	}
}

// TestTransportSendContextCancellation tests context cancellation
func TestTransportSendContextCancellation(t *testing.T) {
	// Create a server that delays the response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id": 1}`))
	}))
	defer server.Close()

	transport := createTestTransport("token", server)

	// Create a context that cancels immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	msg := notifier.NewChatMessage("Test message")
	_, err := transport.Send(ctx, msg)

	if err == nil {
		t.Fatal("Expected error due to context cancellation, got nil")
	}

	if !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("Expected context cancellation error, got: %v", err)
	}
}

// TestTransportSendContextTimeout tests context timeout
func TestTransportSendContextTimeout(t *testing.T) {
	// Create a server that delays the response longer than the timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id": 1}`))
	}))
	defer server.Close()

	transport := createTestTransport("token", server)

	// Create a context with a short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	msg := notifier.NewChatMessage("Test message")
	_, err := transport.Send(ctx, msg)

	if err == nil {
		t.Fatal("Expected error due to context timeout, got nil")
	}

	if !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("Expected context timeout error, got: %v", err)
	}
}

// TestTransportSendDefaultTitle tests that default title is set when not provided
func TestTransportSendDefaultTitle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload map[string]interface{}
		json.Unmarshal(body, &payload)

		if title, ok := payload["title"].(string); !ok || title != "Notification" {
			t.Errorf("Expected default title 'Notification', got %v", payload["title"])
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id": 1}`))
	}))
	defer server.Close()

	transport := createTestTransport("token", server)

	// Message without options (no title set)
	msg := notifier.NewChatMessage("Test message")
	_, err := transport.Send(context.Background(), msg)

	if err != nil {
		t.Fatalf("Expected successful send, got error: %v", err)
	}
}

// TestTransportSendEmptyValuesFiltered tests that empty values are filtered out
func TestTransportSendEmptyValuesFiltered(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload map[string]interface{}
		json.Unmarshal(body, &payload)

		// Empty string should be filtered
		if _, exists := payload["empty_string"]; exists {
			t.Error("Expected empty string to be filtered out")
		}

		// Empty array should be filtered
		if _, exists := payload["empty_array"]; exists {
			t.Error("Expected empty array to be filtered out")
		}

		// Empty map should be filtered
		if _, exists := payload["empty_map"]; exists {
			t.Error("Expected empty map to be filtered out")
		}

		// Nil value should be filtered
		if _, exists := payload["nil_value"]; exists {
			t.Error("Expected nil value to be filtered out")
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id": 1}`))
	}))
	defer server.Close()

	transport := createTestTransport("token", server)

	// Create options with empty values
	opts := NewOptions().Title("Test")
	opts.options["empty_string"] = ""
	opts.options["empty_array"] = []any{}
	opts.options["empty_map"] = map[string]any{}
	opts.options["nil_value"] = nil

	msg := notifier.NewChatMessage("Test message").WithOptions("gotify", opts)
	_, err := transport.Send(context.Background(), msg)

	if err != nil {
		t.Fatalf("Expected successful send, got error: %v", err)
	}
}

// TestTransportSendUnsupportedMessageType tests sending unsupported message type
func TestTransportSendUnsupportedMessageType(t *testing.T) {
	transport := NewTransport("token", nil)
	transport.SetHost("gotify.example.com")

	// Create a mock message that's not a ChatMessage
	msg := &mockUnsupportedMessage{content: "test"}

	_, err := transport.Send(context.Background(), msg)

	if err == nil {
		t.Fatal("Expected error for unsupported message type, got nil")
	}

	if !strings.Contains(err.Error(), "unsupported message type") {
		t.Errorf("Expected unsupported message type error, got: %v", err)
	}
}

// TestTransportSendEndpointConstruction tests correct HTTPS endpoint construction
func TestTransportSendEndpointConstruction(t *testing.T) {
	tests := []struct {
		name         string
		host         string
		expectedPath string
	}{
		{
			name:         "Standard domain",
			host:         "gotify.example.com",
			expectedPath: "/message",
		},
		{
			name:         "Domain with port",
			host:         "gotify.example.com:8080",
			expectedPath: "/message",
		},
		{
			name:         "Localhost with port",
			host:         "127.0.0.1:8080",
			expectedPath: "/message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestReceived := false
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestReceived = true
				if r.URL.Path != tt.expectedPath {
					t.Errorf("Expected path %s, got %s", tt.expectedPath, r.URL.Path)
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"id": 1}`))
			}))
			defer server.Close()

			transport := createTestTransport("token", server)

			msg := notifier.NewChatMessage("Test message")
			_, err := transport.Send(context.Background(), msg)

			if err != nil {
				t.Fatalf("Expected successful send, got error: %v", err)
			}

			if !requestReceived {
				t.Error("Expected request to be received by server")
			}
		})
	}
}

// TestTransportSendAllOptionsIncluded tests that all Gotify options are included in request
func TestTransportSendAllOptionsIncluded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload map[string]interface{}
		json.Unmarshal(body, &payload)

		// Verify all expected fields
		expectedFields := map[string]interface{}{
			"message":  "Test message",
			"title":    "Custom Title",
			"priority": float64(5),
		}

		for key, expectedValue := range expectedFields {
			actualValue, exists := payload[key]
			if !exists {
				t.Errorf("Expected field '%s' to exist in payload", key)
				continue
			}
			if actualValue != expectedValue {
				t.Errorf("Expected %s=%v, got %v", key, expectedValue, actualValue)
			}
		}

		// Verify extras
		if extras, ok := payload["extras"].(map[string]interface{}); ok {
			if key1 := extras["key1"]; key1 != "value1" {
				t.Errorf("Expected extras.key1='value1', got %v", key1)
			}
			if key2 := extras["key2"]; key2 != "value2" {
				t.Errorf("Expected extras.key2='value2', got %v", key2)
			}
		} else {
			t.Error("Expected extras to be present in payload")
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id": 123}`))
	}))
	defer server.Close()

	transport := createTestTransport("token", server)

	opts := NewOptions().
		Title("Custom Title").
		Priority(5).
		AddExtra("key1", "value1").
		AddExtra("key2", "value2")

	msg := notifier.NewChatMessage("Test message").WithOptions("gotify", opts)
	_, err := transport.Send(context.Background(), msg)

	if err != nil {
		t.Fatalf("Expected successful send, got error: %v", err)
	}
}

// mockRoundTripper is a custom http.RoundTripper for testing
type mockRoundTripper struct {
	response *http.Response
	err      error
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

// errorReader is an io.ReadCloser that always returns an error
type errorReader struct {
	err error
}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, e.err
}

func (e *errorReader) Close() error {
	return nil
}

// mockUnsupportedMessage is a mock message type that implements MessageInterface
// but is not a ChatMessage
type mockUnsupportedMessage struct {
	content string
}

func (m *mockUnsupportedMessage) GetRecipientId() string {
	return ""
}

func (m *mockUnsupportedMessage) GetSubject() string {
	return m.content
}

func (m *mockUnsupportedMessage) GetOptions(transportKey string) notifier.MessageOptionsInterface {
	return nil
}

func (m *mockUnsupportedMessage) GetTransport() string {
	return ""
}

func TestOptions(t *testing.T) {
	opts := NewOptions().
		Title("Title").
		Priority(8).
		Extras(map[string]any{"key": "value"}).
		AddExtra("key2", "value2")

	m := opts.ToMap()
	if m["title"] != "Title" {
		t.Error("Title not set")
	}
	if m["priority"] != 8 {
		t.Error("Priority not set")
	}
	if m["extras"] == nil {
		t.Error("Extras not set")
	}
}

func TestPriorityClamping(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{-5, 0},
		{0, 0},
		{5, 5},
		{10, 10},
		{15, 10},
	}

	for _, tt := range tests {
		opts := NewOptions().Priority(tt.input)
		if opts.ToMap()["priority"] != tt.expected {
			t.Errorf("Priority %d should be clamped to %d", tt.input, tt.expected)
		}
	}
}

func TestDSN(t *testing.T) {
	dsn, err := notifier.NewDSN("gotify://A1b2C3d4@gotify.example.com")
	if err != nil {
		t.Fatalf("Failed to parse DSN: %v", err)
	}

	if dsn.GetScheme() != "gotify" {
		t.Errorf("Expected scheme 'gotify', got '%s'", dsn.GetScheme())
	}
	if dsn.GetUser() != "A1b2C3d4" {
		t.Errorf("Expected user 'A1b2C3d4', got '%s'", dsn.GetUser())
	}
	if dsn.GetHost() != "gotify.example.com" {
		t.Errorf("Expected host 'gotify.example.com', got '%s'", dsn.GetHost())
	}
}

func TestFactory(t *testing.T) {
	factory := NewTransportFactory(nil)
	dsn, _ := notifier.NewDSN("gotify://token@gotify.example.com")

	if !factory.Supports(dsn) {
		t.Error("Factory should support gotify DSN")
	}

	transport, err := factory.Create(dsn)
	if err != nil {
		t.Fatalf("Failed to create transport: %v", err)
	}

	if transport == nil {
		t.Fatal("Transport is nil")
	}

	gotifyTransport, ok := transport.(*Transport)
	if !ok {
		t.Fatal("Transport is not a Gotify transport")
	}

	if gotifyTransport.token != "token" {
		t.Errorf("Token mismatch: %s", gotifyTransport.token)
	}
}

func TestMissingToken(t *testing.T) {
	factory := NewTransportFactory(nil)
	dsn, _ := notifier.NewDSN("gotify://@gotify.example.com")

	_, err := factory.Create(dsn)
	if err == nil {
		t.Error("Expected error for missing token")
	}
}

func TestMissingHost(t *testing.T) {
	factory := NewTransportFactory(nil)
	dsn, err := notifier.NewDSN("gotify://token@")
	if err != nil {
		// DSN parsing should fail for invalid host
		return
	}

	_, err = factory.Create(dsn)
	if err == nil {
		t.Error("Expected error for missing host")
	}
}
