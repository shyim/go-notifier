package slack

import (
	"encoding/json"
	"time"
)

// Options implements MessageOptionsInterface for Slack.
type Options struct {
	options map[string]any
	blocks  []map[string]any
}

func NewOptions() *Options {
	return &Options{
		options: make(map[string]any),
		blocks:  make([]map[string]any, 0),
	}
}

func (o *Options) ToMap() map[string]any {
	if len(o.blocks) > 0 {
		o.options["blocks"] = o.blocks
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

// AsUser sends the message as the user.
func (o *Options) AsUser(asUser bool) *Options {
	o.options["as_user"] = asUser
	return o
}

// PostAt schedules the message to be posted at a specific timestamp.
func (o *Options) PostAt(timestamp time.Time) *Options {
	o.options["post_at"] = timestamp.Unix()
	return o
}

// Block adds a block to the message.
func (o *Options) Block(block Block) *Options {
	if len(o.blocks) >= 50 {
		// Slack limit
		return o
	}
	o.blocks = append(o.blocks, block.ToMap())
	return o
}

// IconEmoji sets the emoji for the bot.
func (o *Options) IconEmoji(emoji string) *Options {
	o.options["icon_emoji"] = emoji
	return o
}

// IconUrl sets the icon URL for the bot.
func (o *Options) IconUrl(url string) *Options {
	o.options["icon_url"] = url
	return o
}

// LinkNames enables linking of names in the message.
func (o *Options) LinkNames(link bool) *Options {
	o.options["link_names"] = link
	return o
}

// Mrkdwn enables markdown formatting.
func (o *Options) Mrkdwn(mrkdwn bool) *Options {
	o.options["mrkdwn"] = mrkdwn
	return o
}

// Parse sets the parse mode.
func (o *Options) Parse(parse string) *Options {
	o.options["parse"] = parse
	return o
}

// UnfurlLinks controls link unfurling.
func (o *Options) UnfurlLinks(unfurl bool) *Options {
	o.options["unfurl_links"] = unfurl
	return o
}

// UnfurlMedia controls media unfurling.
func (o *Options) UnfurlMedia(unfurl bool) *Options {
	o.options["unfurl_media"] = unfurl
	return o
}

// Username sets the bot username.
func (o *Options) Username(username string) *Options {
	o.options["username"] = username
	return o
}

// ThreadTs sets the thread timestamp for threading messages.
func (o *Options) ThreadTs(threadTs string) *Options {
	o.options["thread_ts"] = threadTs
	return o
}

// MarshalJSON implements json.Marshaler.
func (o *Options) MarshalJSON() ([]byte, error) {
	return json.Marshal(o.options)
}

// Block represents a Slack block.
type Block interface {
	ToMap() map[string]any
}

// SectionBlock represents a section block.
type SectionBlock struct {
	options map[string]any
	fields  []map[string]any
}

func NewSectionBlock() *SectionBlock {
	return &SectionBlock{
		options: map[string]any{"type": "section"},
		fields:  make([]map[string]any, 0),
	}
}

func (b *SectionBlock) Text(text string, markdown ...bool) *SectionBlock {
	m := true
	if len(markdown) > 0 {
		m = markdown[0]
	}
	blockType := "mrkdwn"
	if !m {
		blockType = "plain_text"
	}
	b.options["text"] = map[string]any{
		"type": blockType,
		"text": text,
	}
	return b
}

func (b *SectionBlock) Field(text string, markdown ...bool) *SectionBlock {
	if len(b.fields) >= 10 {
		return b
	}
	m := true
	if len(markdown) > 0 {
		m = markdown[0]
	}
	fieldType := "mrkdwn"
	if !m {
		fieldType = "plain_text"
	}
	b.fields = append(b.fields, map[string]any{
		"type": fieldType,
		"text": text,
	})
	return b
}

func (b *SectionBlock) Accessory(element BlockElement) *SectionBlock {
	b.options["accessory"] = element.ToMap()
	return b
}

func (b *SectionBlock) ToMap() map[string]any {
	if len(b.fields) > 0 {
		b.options["fields"] = b.fields
	}
	return b.options
}

// DividerBlock represents a divider block.
type DividerBlock struct {
	options map[string]any
}

func NewDividerBlock() *DividerBlock {
	return &DividerBlock{
		options: map[string]any{"type": "divider"},
	}
}

func (b *DividerBlock) ToMap() map[string]any {
	return b.options
}

// ContextBlock represents a context block.
type ContextBlock struct {
	options map[string]any
}

func NewContextBlock() *ContextBlock {
	return &ContextBlock{
		options: map[string]any{"type": "context"},
	}
}

func (b *ContextBlock) Elements(elements ...BlockElement) *ContextBlock {
	e := make([]map[string]any, len(elements))
	for i, el := range elements {
		e[i] = el.ToMap()
	}
	b.options["elements"] = e
	return b
}

func (b *ContextBlock) ToMap() map[string]any {
	return b.options
}

// ImageBlock represents an image block.
type ImageBlock struct {
	options map[string]any
}

func NewImageBlock(imageUrl, altText string) *ImageBlock {
	return &ImageBlock{
		options: map[string]any{
			"type":      "image",
			"image_url": imageUrl,
			"alt_text":  altText,
		},
	}
}

func (b *ImageBlock) ToMap() map[string]any {
	return b.options
}

// HeaderBlock represents a header block.
type HeaderBlock struct {
	options map[string]any
}

func NewHeaderBlock(text string) *HeaderBlock {
	return &HeaderBlock{
		options: map[string]any{
			"type": "header",
			"text": map[string]any{
				"type": "plain_text",
				"text": text,
			},
		},
	}
}

func (b *HeaderBlock) ToMap() map[string]any {
	return b.options
}

// ActionsBlock represents an actions block.
type ActionsBlock struct {
	options map[string]any
}

func NewActionsBlock() *ActionsBlock {
	return &ActionsBlock{
		options: map[string]any{"type": "actions"},
	}
}

func (b *ActionsBlock) Elements(elements ...BlockElement) *ActionsBlock {
	e := make([]map[string]any, len(elements))
	for i, el := range elements {
		e[i] = el.ToMap()
	}
	b.options["elements"] = e
	return b
}

func (b *ActionsBlock) ToMap() map[string]any {
	return b.options
}

// BlockElement represents a block element.
type BlockElement interface {
	ToMap() map[string]any
}

// ButtonElement represents a button element.
type ButtonElement struct {
	options map[string]any
}

func NewButtonElement(text string) *ButtonElement {
	return &ButtonElement{
		options: map[string]any{
			"type": "button",
			"text": map[string]any{
				"type": "plain_text",
				"text": text,
			},
		},
	}
}

func (b *ButtonElement) ActionId(actionId string) *ButtonElement {
	b.options["action_id"] = actionId
	return b
}

func (b *ButtonElement) Url(url string) *ButtonElement {
	b.options["url"] = url
	return b
}

func (b *ButtonElement) Value(value string) *ButtonElement {
	b.options["value"] = value
	return b
}

func (b *ButtonElement) Style(style string) *ButtonElement {
	// "primary" or "danger"
	b.options["style"] = style
	return b
}

func (b *ButtonElement) ToMap() map[string]any {
	return b.options
}

// ImageElement represents an image element.
type ImageElement struct {
	options map[string]any
}

func NewImageElement(imageUrl, altText string) *ImageElement {
	return &ImageElement{
		options: map[string]any{
			"type":      "image",
			"image_url": imageUrl,
			"alt_text":  altText,
		},
	}
}

func (b *ImageElement) ToMap() map[string]any {
	return b.options
}

// UpdateMessageOptions for updating messages.
type UpdateMessageOptions struct {
	*Options
	channel   string
	messageId string
}

func NewUpdateMessageOptions(channel, messageId string) *UpdateMessageOptions {
	return &UpdateMessageOptions{
		Options:   NewOptions(),
		channel:   channel,
		messageId: messageId,
	}
}

func (o *UpdateMessageOptions) ToMap() map[string]any {
	m := o.Options.ToMap()
	m["channel"] = o.channel
	m["ts"] = o.messageId
	return m
}
