package telegram

import (
	"encoding/json"
)

// Options implements MessageOptionsInterface for Telegram.
type Options struct {
	options map[string]any
	upload  map[string]string
}

func NewOptions() *Options {
	return &Options{
		options: make(map[string]any),
		upload:  make(map[string]string),
	}
}

func (o *Options) ToMap() map[string]any {
	if len(o.upload) > 0 {
		o.options["upload"] = o.upload
	}
	return o.options
}

func (o *Options) GetRecipientId() string {
	if id, ok := o.options["recipient_id"].(string); ok {
		return id
	}
	// Backward compatibility with chat_id
	if id, ok := o.options["chat_id"].(string); ok {
		return id
	}
	return ""
}

// Recipient sets the chat ID for the message.
func (o *Options) Recipient(id string) *Options {
	o.options["recipient_id"] = id
	return o
}

// ChatID sets the chat ID for the message (deprecated, use Recipient).
func (o *Options) ChatID(id string) *Options {
	o.options["recipient_id"] = id
	return o
}

// ParseMode sets the parse mode for the message.
// Options: "HTML", "Markdown", "MarkdownV2"
func (o *Options) ParseMode(mode string) *Options {
	o.options["parse_mode"] = mode
	return o
}

// DisableWebPagePreview disables link previews in the message.
func (o *Options) DisableWebPagePreview(disable bool) *Options {
	o.options["disable_web_page_preview"] = disable
	return o
}

// DisableNotification sends the message silently.
func (o *Options) DisableNotification(disable bool) *Options {
	o.options["disable_notification"] = disable
	return o
}

// ProtectContent protects the message from forwarding.
func (o *Options) ProtectContent(protect bool) *Options {
	o.options["protect_content"] = protect
	return o
}

// ReplyTo sets the message ID to reply to.
func (o *Options) ReplyTo(messageID int) *Options {
	o.options["reply_to_message_id"] = messageID
	return o
}

// MessageThreadID sets the thread ID for forum topics.
func (o *Options) MessageThreadID(threadID int) *Options {
	o.options["message_thread_id"] = threadID
	return o
}

// Edit edits an existing message.
func (o *Options) Edit(messageID int) *Options {
	o.options["message_id"] = messageID
	return o
}

// AnswerCallbackQuery answers a callback query.
func (o *Options) AnswerCallbackQuery(callbackQueryID string, showAlert ...bool) *Options {
	o.options["callback_query_id"] = callbackQueryID
	if len(showAlert) > 0 && showAlert[0] {
		o.options["show_alert"] = true
	}
	return o
}

// Photo sends a photo (URL or file ID).
func (o *Options) Photo(url string) *Options {
	o.options["photo"] = url
	return o
}

// UploadPhoto uploads a photo from a local file path.
func (o *Options) UploadPhoto(path string) *Options {
	o.upload["photo"] = path
	return o
}

// Document sends a document (URL or file ID).
func (o *Options) Document(url string) *Options {
	o.options["document"] = url
	return o
}

// UploadDocument uploads a document from a local file path.
func (o *Options) UploadDocument(path string) *Options {
	o.upload["document"] = path
	return o
}

// Video sends a video (URL or file ID).
func (o *Options) Video(url string) *Options {
	o.options["video"] = url
	return o
}

// UploadVideo uploads a video from a local file path.
func (o *Options) UploadVideo(path string) *Options {
	o.upload["video"] = path
	return o
}

// Audio sends an audio (URL or file ID).
func (o *Options) Audio(url string) *Options {
	o.options["audio"] = url
	return o
}

// UploadAudio uploads an audio from a local file path.
func (o *Options) UploadAudio(path string) *Options {
	o.upload["audio"] = path
	return o
}

// Animation sends an animation (URL or file ID).
func (o *Options) Animation(url string) *Options {
	o.options["animation"] = url
	return o
}

// UploadAnimation uploads an animation from a local file path.
func (o *Options) UploadAnimation(path string) *Options {
	o.upload["animation"] = path
	return o
}

// Sticker sends a sticker (URL or file ID).
func (o *Options) Sticker(url string, emoji ...string) *Options {
	o.options["sticker"] = url
	if len(emoji) > 0 {
		o.options["emoji"] = emoji[0]
	}
	return o
}

// UploadSticker uploads a sticker from a local file path.
func (o *Options) UploadSticker(path string, emoji ...string) *Options {
	o.upload["sticker"] = path
	if len(emoji) > 0 {
		o.options["emoji"] = emoji[0]
	}
	return o
}

// Location sends a location.
func (o *Options) Location(latitude, longitude float64) *Options {
	o.options["location"] = map[string]float64{
		"latitude":  latitude,
		"longitude": longitude,
	}
	return o
}

// Venue sends a venue.
func (o *Options) Venue(latitude, longitude float64, title, address string) *Options {
	o.options["venue"] = map[string]any{
		"latitude":  latitude,
		"longitude": longitude,
		"title":     title,
		"address":   address,
	}
	return o
}

// Contact sends a contact.
func (o *Options) Contact(phoneNumber, firstName string, lastName ...string) *Options {
	contact := map[string]string{
		"phone_number": phoneNumber,
		"first_name":   firstName,
	}
	if len(lastName) > 0 {
		contact["last_name"] = lastName[0]
	}
	o.options["contact"] = contact
	return o
}

// ReplyMarkup sets the reply markup (keyboard, inline keyboard, etc.).
func (o *Options) ReplyMarkup(markup ReplyMarkup) *Options {
	o.options["reply_markup"] = markup.ToMap()
	return o
}

// HasSpoiler sets the spoiler flag (works with photos).
func (o *Options) HasSpoiler(spoiler bool) *Options {
	o.options["has_spoiler"] = spoiler
	return o
}

// MarshalJSON implements json.Marshaler for Options.
func (o *Options) MarshalJSON() ([]byte, error) {
	return json.Marshal(o.options)
}

// ReplyMarkup represents a Telegram reply markup.
type ReplyMarkup interface {
	ToMap() map[string]any
}

// InlineKeyboard represents an inline keyboard.
type InlineKeyboard struct {
	buttons [][]InlineKeyboardButton
}

func NewInlineKeyboard() *InlineKeyboard {
	return &InlineKeyboard{
		buttons: make([][]InlineKeyboardButton, 0),
	}
}

func (k *InlineKeyboard) AddRow(buttons ...InlineKeyboardButton) *InlineKeyboard {
	k.buttons = append(k.buttons, buttons)
	return k
}

func (k *InlineKeyboard) ToMap() map[string]any {
	rows := make([][]map[string]any, len(k.buttons))
	for i, row := range k.buttons {
		rows[i] = make([]map[string]any, len(row))
		for j, btn := range row {
			rows[i][j] = btn.ToMap()
		}
	}
	return map[string]any{
		"inline_keyboard": rows,
	}
}

// InlineKeyboardButton represents a button in an inline keyboard.
type InlineKeyboardButton struct {
	text         string
	callbackData string
	url          string
}

func NewInlineKeyboardButton(text string) InlineKeyboardButton {
	return InlineKeyboardButton{
		text: text,
	}
}

func (b InlineKeyboardButton) CallbackData(data string) InlineKeyboardButton {
	b.callbackData = data
	return b
}

func (b InlineKeyboardButton) URL(url string) InlineKeyboardButton {
	b.url = url
	return b
}

func (b *InlineKeyboardButton) ToMap() map[string]any {
	m := map[string]any{"text": b.text}
	if b.callbackData != "" {
		m["callback_data"] = b.callbackData
	}
	if b.url != "" {
		m["url"] = b.url
	}
	return m
}

// ReplyKeyboard represents a reply keyboard.
type ReplyKeyboard struct {
	buttons         [][]KeyboardButton
	resizeKeyboard  bool
	oneTimeKeyboard bool
	selective       bool
}

func NewReplyKeyboard() *ReplyKeyboard {
	return &ReplyKeyboard{
		buttons: make([][]KeyboardButton, 0),
	}
}

func (k *ReplyKeyboard) AddRow(buttons ...KeyboardButton) *ReplyKeyboard {
	k.buttons = append(k.buttons, buttons)
	return k
}

func (k *ReplyKeyboard) ResizeKeyboard(resize bool) *ReplyKeyboard {
	k.resizeKeyboard = resize
	return k
}

func (k *ReplyKeyboard) OneTimeKeyboard(oneTime bool) *ReplyKeyboard {
	k.oneTimeKeyboard = oneTime
	return k
}

func (k *ReplyKeyboard) Selective(selective bool) *ReplyKeyboard {
	k.selective = selective
	return k
}

func (k *ReplyKeyboard) ToMap() map[string]any {
	rows := make([][]map[string]any, len(k.buttons))
	for i, row := range k.buttons {
		rows[i] = make([]map[string]any, len(row))
		for j, btn := range row {
			rows[i][j] = btn.ToMap()
		}
	}
	m := map[string]any{"keyboard": rows}
	if k.resizeKeyboard {
		m["resize_keyboard"] = true
	}
	if k.oneTimeKeyboard {
		m["one_time_keyboard"] = true
	}
	if k.selective {
		m["selective"] = true
	}
	return m
}

// KeyboardButton represents a button in a reply keyboard.
type KeyboardButton struct {
	text string
}

func NewKeyboardButton(text string) KeyboardButton {
	return KeyboardButton{
		text: text,
	}
}

func (b KeyboardButton) ToMap() map[string]any {
	return map[string]any{"text": b.text}
}
