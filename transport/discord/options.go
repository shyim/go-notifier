package discord

import (
	"encoding/json"
	"time"
)

// Options implements MessageOptionsInterface for Discord.
type Options struct {
	options map[string]any
	embeds  []map[string]any
}

func NewOptions() *Options {
	return &Options{
		options: make(map[string]any),
		embeds:  make([]map[string]any, 0),
	}
}

func (o *Options) ToMap() map[string]any {
	if len(o.embeds) > 0 {
		o.options["embeds"] = o.embeds
	}
	return o.options
}

func (o *Options) GetRecipientId() string {
	if id, ok := o.options["recipient_id"].(string); ok {
		return id
	}
	return ""
}

// Recipient sets the webhook recipient ID.
func (o *Options) Recipient(id string) *Options {
	o.options["recipient_id"] = id
	return o
}

// Username sets the webhook username.
func (o *Options) Username(username string) *Options {
	o.options["username"] = username
	return o
}

// AvatarUrl sets the webhook avatar URL.
func (o *Options) AvatarUrl(url string) *Options {
	o.options["avatar_url"] = url
	return o
}

// TTS enables text-to-speech.
func (o *Options) TTS(tts bool) *Options {
	o.options["tts"] = tts
	return o
}

// AddEmbed adds an embed to the message.
func (o *Options) AddEmbed(embed *Embed) *Options {
	if len(o.embeds) >= 10 {
		// Discord limit
		return o
	}
	o.embeds = append(o.embeds, embed.ToMap())
	return o
}

// MarshalJSON implements json.Marshaler.
func (o *Options) MarshalJSON() ([]byte, error) {
	return json.Marshal(o.options)
}

// Embed represents a Discord embed.
type Embed struct {
	options map[string]any
	fields  []map[string]any
}

func NewEmbed() *Embed {
	return &Embed{
		options: make(map[string]any),
		fields:  make([]map[string]any, 0),
	}
}

func (e *Embed) ToMap() map[string]any {
	if len(e.fields) > 0 {
		e.options["fields"] = e.fields
	}
	return e.options
}

// Title sets the embed title.
func (e *Embed) Title(title string) *Embed {
	e.options["title"] = title
	return e
}

// Description sets the embed description.
func (e *Embed) Description(description string) *Embed {
	e.options["description"] = description
	return e
}

// URL sets the embed URL.
func (e *Embed) URL(url string) *Embed {
	e.options["url"] = url
	return e
}

// Timestamp sets the embed timestamp.
func (e *Embed) Timestamp(timestamp time.Time) *Embed {
	e.options["timestamp"] = timestamp.Format(time.RFC3339)
	return e
}

// Color sets the embed color (hex).
func (e *Embed) Color(color int) *Embed {
	e.options["color"] = color
	return e
}

// Footer sets the embed footer.
func (e *Embed) Footer(text string, iconUrl ...string) *Embed {
	footer := map[string]any{"text": text}
	if len(iconUrl) > 0 {
		footer["icon_url"] = iconUrl[0]
	}
	e.options["footer"] = footer
	return e
}

// Thumbnail sets the embed thumbnail.
func (e *Embed) Thumbnail(url string) *Embed {
	e.options["thumbnail"] = map[string]any{"url": url}
	return e
}

// Image sets the embed image.
func (e *Embed) Image(url string) *Embed {
	e.options["image"] = map[string]any{"url": url}
	return e
}

// Author sets the embed author.
func (e *Embed) Author(name string, url ...string) *Embed {
	author := map[string]any{"name": name}
	if len(url) > 0 {
		author["url"] = url[0]
	}
	e.options["author"] = author
	return e
}

// AddField adds a field to the embed.
func (e *Embed) AddField(name, value string, inline ...bool) *Embed {
	field := map[string]any{"name": name, "value": value}
	if len(inline) > 0 {
		field["inline"] = inline[0]
	}
	e.fields = append(e.fields, field)
	return e
}
