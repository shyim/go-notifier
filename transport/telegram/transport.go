package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/shyim/go-notifier"
)

const optionCaption = "caption"

// Transport sends messages via Telegram Bot API.
type Transport struct {
	*notifier.AbstractTransport
	token       string
	chatChannel string
}

// NewTransport creates a new Telegram transport.
func NewTransport(token, chatChannel string, client *http.Client) *Transport {
	if client == nil {
		client = http.DefaultClient
	}
	return &Transport{
		AbstractTransport: notifier.NewAbstractTransport(client),
		token:             token,
		chatChannel:       chatChannel,
	}
}

func (t *Transport) String() string {
	endpoint := t.getEndpoint()
	if endpoint == "" {
		endpoint = "api.telegram.org"
	}
	if t.chatChannel != "" {
		return fmt.Sprintf("telegram://%s?channel=%s", endpoint, t.chatChannel)
	}
	return fmt.Sprintf("telegram://%s", endpoint)
}

func (t *Transport) Supports(message notifier.MessageInterface) bool {
	_, ok := message.(*notifier.ChatMessage)
	return ok
}

func (t *Transport) Send(ctx context.Context, message notifier.MessageInterface) (*notifier.SentMessage, error) {
	chatMsg, ok := message.(*notifier.ChatMessage)
	if !ok {
		return nil, fmt.Errorf("telegram: unsupported message type %T, expected ChatMessage", message)
	}

	chatID := chatMsg.GetRecipientId()
	if chatID == "" && t.chatChannel != "" {
		chatID = t.chatChannel
	}

	options := make(map[string]any)
	if opts, ok := chatMsg.GetOptions("telegram").(*Options); ok {
		options = opts.ToMap()
	}

	// Telegram API uses 'chat_id' but we store it as 'recipient_id' for consistency
	options["chat_id"] = chatID
	// Remove recipient_id as it's not a Telegram API parameter
	delete(options, "recipient_id")
	text := chatMsg.GetSubject()

	// Handle parse mode and markdown escaping
	parseMode, hasParseMode := options["parse_mode"].(string)
	if !hasParseMode || parseMode == "MarkdownV2" {
		options["parse_mode"] = "MarkdownV2"
		// Escape special characters for MarkdownV2
		text = escapeMarkdownV2(text)
	}

	// Handle file uploads
	var body io.Reader
	var contentType string
	upload, hasUpload := options["upload"].(map[string]string)
	if hasUpload {
		var err error
		body, contentType, err = t.createMultipartBody(options, upload, text)
		if err != nil {
			return nil, fmt.Errorf("telegram: create multipart body: %w", err)
		}
		// Remove upload from options as it's now in the body
		delete(options, "upload")
	} else {
		// Determine the method and text option
		method := t.getPath(options)
		textOption := t.getTextOption(options)

		if textOption != "" {
			options[textOption] = text
		}

		// Filter out empty options
		filteredOptions := make(map[string]any)
		for k, v := range options {
			if v != nil {
				filteredOptions[k] = v
			}
		}

		// Extract location coordinates to top-level for Telegram API
		if loc, ok := filteredOptions["location"].(map[string]float64); ok {
			filteredOptions["latitude"] = loc["latitude"]
			filteredOptions["longitude"] = loc["longitude"]
			delete(filteredOptions, "location")
		}

		// Extract venue coordinates to top-level for Telegram API
		if venue, ok := filteredOptions["venue"].(map[string]any); ok {
			filteredOptions["latitude"] = venue["latitude"]
			filteredOptions["longitude"] = venue["longitude"]
			filteredOptions["title"] = venue["title"]
			filteredOptions["address"] = venue["address"]
			delete(filteredOptions, "venue")
		}

		// Extract contact fields to top-level for Telegram API
		if contact, ok := filteredOptions["contact"].(map[string]string); ok {
			filteredOptions["phone_number"] = contact["phone_number"]
			filteredOptions["first_name"] = contact["first_name"]
			if lastName, exists := contact["last_name"]; exists {
				filteredOptions["last_name"] = lastName
			}
			delete(filteredOptions, "contact")
		}

		jsonBody, err := json.Marshal(filteredOptions)
		if err != nil {
			return nil, fmt.Errorf("telegram: marshal options: %w", err)
		}
		body = bytes.NewReader(jsonBody)
		contentType = "application/json"

		// Update endpoint with method
		endpoint := fmt.Sprintf("https://%s/bot%s/%s", t.getEndpoint(), t.token, method)
		return t.doRequest(ctx, endpoint, body, contentType, message)
	}

	// For uploads, we need to determine the method first
	method := t.getPath(options)
	endpoint := fmt.Sprintf("https://%s/bot%s/%s", t.getEndpoint(), t.token, method)
	return t.doRequest(ctx, endpoint, body, contentType, message)
}

func (t *Transport) doRequest(ctx context.Context, endpoint string, body io.Reader, contentType string, originalMessage notifier.MessageInterface) (*notifier.SentMessage, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("telegram: create request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)

	resp, err := t.AbstractTransport.GetClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("telegram: send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("telegram: API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		OK     bool `json:"ok"`
		Result struct {
			MessageID int `json:"message_id"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("telegram: decode response: %w", err)
	}

	sentMessage := notifier.NewSentMessage(originalMessage, t.String())
	if result.Result.MessageID != 0 {
		sentMessage.SetMessageID(fmt.Sprintf("%d", result.Result.MessageID))
	}

	return sentMessage, nil
}

func (t *Transport) createMultipartBody(options map[string]any, upload map[string]string, text string) (io.Reader, string, error) {
	buf := &bytes.Buffer{}
	writer := multipart.NewWriter(buf)

	// Add text if applicable
	textOption := t.getTextOption(options)
	if textOption != "" && text != "" {
		if err := writer.WriteField(textOption, text); err != nil {
			return nil, "", fmt.Errorf("write text field: %w", err)
		}
	}

	// Add other options
	for k, v := range options {
		if k == "upload" || k == "photo" || k == "document" || k == "video" || k == "audio" || k == "animation" || k == "sticker" {
			continue
		}
		var err error
		switch val := v.(type) {
		case string:
			err = writer.WriteField(k, val)
		case int:
			err = writer.WriteField(k, fmt.Sprintf("%d", val))
		case bool:
			err = writer.WriteField(k, fmt.Sprintf("%t", val))
		case float64:
			err = writer.WriteField(k, fmt.Sprintf("%f", val))
		case map[string]any:
			jsonVal, jsonErr := json.Marshal(val)
			if jsonErr != nil {
				return nil, "", fmt.Errorf("marshal field %s: %w", k, jsonErr)
			}
			err = writer.WriteField(k, string(jsonVal))
		}
		if err != nil {
			return nil, "", fmt.Errorf("write field %s: %w", k, err)
		}
	}

	// Add files
	for fieldName, filePath := range upload {
		if err := t.addFileToWriter(writer, fieldName, filePath); err != nil {
			return nil, "", fmt.Errorf("add file %s: %w", filePath, err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("close multipart writer: %w", err)
	}
	return buf, writer.FormDataContentType(), nil
}

func (t *Transport) addFileToWriter(writer *multipart.Writer, fieldName, filePath string) error {
	file, err := os.Open(filePath) //nolint:gosec // G304: file path comes from user-provided upload options
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	part, err := writer.CreateFormFile(fieldName, filepath.Base(filePath))
	if err != nil {
		return err
	}
	_, err = io.Copy(part, file)
	return err
}

func (t *Transport) getPath(options map[string]any) string {
	if _, ok := options["message_id"]; ok {
		return "editMessageText"
	}
	if _, ok := options["callback_query_id"]; ok {
		return "answerCallbackQuery"
	}
	if _, ok := options["photo"]; ok {
		return "sendPhoto"
	}
	if _, ok := options["location"]; ok {
		return "sendLocation"
	}
	if _, ok := options["audio"]; ok {
		return "sendAudio"
	}
	if _, ok := options["document"]; ok {
		return "sendDocument"
	}
	if _, ok := options["video"]; ok {
		return "sendVideo"
	}
	if _, ok := options["animation"]; ok {
		return "sendAnimation"
	}
	if _, ok := options["venue"]; ok {
		return "sendVenue"
	}
	if _, ok := options["contact"]; ok {
		return "sendContact"
	}
	if _, ok := options["sticker"]; ok {
		return "sendSticker"
	}
	return "sendMessage"
}

func (t *Transport) getTextOption(options map[string]any) string {
	if _, ok := options["photo"]; ok {
		return optionCaption
	}
	if _, ok := options["audio"]; ok {
		return optionCaption
	}
	if _, ok := options["document"]; ok {
		return optionCaption
	}
	if _, ok := options["video"]; ok {
		return optionCaption
	}
	if _, ok := options["animation"]; ok {
		return optionCaption
	}
	if _, ok := options["sticker"]; ok {
		return ""
	}
	if _, ok := options["location"]; ok {
		return ""
	}
	if _, ok := options["venue"]; ok {
		return ""
	}
	if _, ok := options["contact"]; ok {
		return ""
	}
	return "text"
}

func (t *Transport) getEndpoint() string {
	endpoint := t.GetEndpoint()
	if endpoint == "" || endpoint == "localhost" {
		return "api.telegram.org"
	}
	return endpoint
}

func escapeMarkdownV2(text string) string {
	// Escape special characters for MarkdownV2
	chars := []string{"_", "*", "[", "]", "(", ")", "~", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!"}
	for _, char := range chars {
		text = strings.ReplaceAll(text, char, "\\"+char)
	}
	return text
}
