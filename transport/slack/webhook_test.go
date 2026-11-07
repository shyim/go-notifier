package slack

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/shyim/go-notifier"
)

func TestWebhookSend(t *testing.T) {
	opts := NewOptions().Block(NewSectionBlock().Text("Details")).ThreadTs("123.456").UnfurlLinks(false).Recipient("C123").Username("Bot")
	msg := notifier.NewChatMessage("Hello webhook!").WithOptions("slack", opts)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/services/secret" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "" {
			t.Error("unexpected Authorization header")
		}
		if r.Header.Get("Content-Type") != "application/json; charset=utf-8" {
			t.Error("missing JSON content type")
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["text"] != "Hello webhook!" || body["thread_ts"] != "123.456" || body["unfurl_links"] != false {
			t.Errorf("unexpected payload: %v", body)
		}
		if blocks, ok := body["blocks"].([]any); !ok || len(blocks) != 1 {
			t.Errorf("missing blocks: %v", body)
		}
		for _, key := range []string{"channel", "recipient_id", "username"} {
			if _, ok := body[key]; ok {
				t.Errorf("unexpected %s", key)
			}
		}
		io.WriteString(w, "ok")
	}))
	defer server.Close()
	transport := NewWebhookTransport(server.URL+"/services/secret", server.Client())
	sent, err := transport.Send(context.Background(), msg)
	if err != nil {
		t.Fatal(err)
	}
	if sent.GetOriginalMessage() != msg || sent.GetMessageID() != "" || sent.GetTransport() != transport.String() {
		t.Fatalf("unexpected sent message: %#v", sent)
	}
	if strings.Contains(transport.String(), "secret") {
		t.Error("transport name leaks webhook secret")
	}
	if opts.ToMap()["username"] != "Bot" || opts.ToMap()["text"] != nil {
		t.Error("send mutated options")
	}
}

func TestWebhookResponses(t *testing.T) {
	for _, tc := range []struct {
		name    string
		status  int
		body    string
		wantErr string
	}{
		{"success", 200, "ok\n", ""},
		{"invalid payload", 400, "invalid_payload", "invalid_payload"},
		{"revoked", 404, "no_service", "status 404"},
		{"rate limited", 429, "too_many_requests", "status 429"},
		{"unexpected success body", 200, "invalid_token", "invalid_token"},
		{"empty body", 200, "", "webhook error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := newMockClient(func(r *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(tc.body)), Header: make(http.Header)}, nil
			})
			_, err := NewWebhookTransport("https://hooks.slack.com/services/test", client).Send(context.Background(), notifier.NewChatMessage("hello"))
			if tc.wantErr == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected %q, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestWebhookFactory(t *testing.T) {
	webhookURL := "https://hooks.slack.com/services/T/B/secret"
	transport, err := notifier.NewTransportFromDSN("slack://default?webhook_url=" + url.QueryEscape(webhookURL))
	if err != nil {
		t.Fatal(err)
	}
	if transport.(*Transport).webhookURL != webhookURL {
		t.Fatal("incorrect webhook URL")
	}
	for _, value := range []string{"relative/path", "ftp://example.com/test", "https:///test", "https://user:pass@example.com/test", "https://example.com/test#fragment", ":%"} {
		if _, err := notifier.NewTransportFromDSN("slack://default?webhook_url=" + url.QueryEscape(value)); err == nil {
			t.Errorf("accepted invalid URL %q", value)
		}
	}
}

func TestWebhookFailures(t *testing.T) {
	client := newMockClient(func(r *http.Request) (*http.Response, error) {
		t.Error("unexpected HTTP request")
		return nil, errors.New("unexpected request")
	})
	transport := NewWebhookTransport("https://hooks.slack.com/services/test", client)
	if _, err := transport.Send(context.Background(), &unsupportedMessage{}); err == nil {
		t.Fatal("accepted unsupported message")
	}
	msg := notifier.NewChatMessage("scheduled").WithOptions("slack", NewOptions().PostAt(time.Now()))
	if _, err := transport.Send(context.Background(), msg); err == nil || !strings.Contains(err.Error(), "post_at") {
		t.Fatalf("expected scheduling error, got %v", err)
	}
	client = newMockClient(func(r *http.Request) (*http.Response, error) { return nil, r.Context().Err() })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := NewWebhookTransport("https://hooks.slack.com/services/test", client).Send(ctx, notifier.NewChatMessage("hello")); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
	client = newMockClient(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: errorReader{}}, nil
	})
	if _, err := NewWebhookTransport("https://hooks.slack.com/services/test", client).Send(context.Background(), notifier.NewChatMessage("hello")); err == nil || !strings.Contains(err.Error(), "read webhook response") {
		t.Fatalf("expected read error, got %v", err)
	}
}
