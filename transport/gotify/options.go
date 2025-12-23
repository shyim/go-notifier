package gotify

import (
	"encoding/json"
)

// Options implements MessageOptionsInterface for Gotify.
type Options struct {
	options map[string]any
	extras  map[string]any
}

func NewOptions() *Options {
	return &Options{
		options: make(map[string]any),
		extras:  make(map[string]any),
	}
}

func (o *Options) ToMap() map[string]any {
	if len(o.extras) > 0 {
		o.options["extras"] = o.extras
	}
	return o.options
}

func (o *Options) GetRecipientId() string {
	if id, ok := o.options["recipient_id"].(string); ok {
		return id
	}
	return ""
}

// Recipient sets the recipient ID (priority or token).
func (o *Options) Recipient(id string) *Options {
	o.options["recipient_id"] = id
	return o
}

// Priority sets the message priority (0-10).
// 0 = lowest, 10 = highest
func (o *Options) Priority(priority int) *Options {
	if priority < 0 {
		priority = 0
	}
	if priority > 10 {
		priority = 10
	}
	o.options["priority"] = priority
	return o
}

// Title sets the message title.
func (o *Options) Title(title string) *Options {
	o.options["title"] = title
	return o
}

// Extras sets custom extras data.
func (o *Options) Extras(extras map[string]any) *Options {
	o.extras = extras
	return o
}

// AddExtra adds a single extra key-value pair.
func (o *Options) AddExtra(key string, value any) *Options {
	o.extras[key] = value
	return o
}

// MarshalJSON implements json.Marshaler.
func (o *Options) MarshalJSON() ([]byte, error) {
	return json.Marshal(o.options)
}
