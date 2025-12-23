package microsoftteams

import (
	"encoding/json"
)

// Options implements MessageOptionsInterface for Microsoft Teams.
type Options struct {
	options          map[string]any
	potentialActions []map[string]any
}

func NewOptions() *Options {
	return &Options{
		options:          make(map[string]any),
		potentialActions: make([]map[string]any, 0),
	}
}

func (o *Options) ToMap() map[string]any {
	if len(o.potentialActions) > 0 {
		o.options["potentialAction"] = o.potentialActions
	}
	return o.options
}

func (o *Options) GetRecipientId() string {
	if id, ok := o.options["recipient_id"].(string); ok {
		return id
	}
	return ""
}

// Recipient sets the recipient ID (webhook URL or channel).
func (o *Options) Recipient(id string) *Options {
	o.options["recipient_id"] = id
	return o
}

// Title sets the title of the message card.
func (o *Options) Title(title string) *Options {
	o.options["title"] = title
	return o
}

// Subtitle sets the subtitle of the message card.
func (o *Options) Subtitle(subtitle string) *Options {
	o.options["subtitle"] = subtitle
	return o
}

// Text sets the main text content of the message card.
func (o *Options) Text(text string) *Options {
	o.options["text"] = text
	return o
}

// ThemeColor sets the theme color for the message card (hex format, e.g., "FF0000").
func (o *Options) ThemeColor(color string) *Options {
	o.options["themeColor"] = color
	return o
}

// PotentialAction adds a potential action to the message card.
func (o *Options) PotentialAction(action map[string]any) *Options {
	o.potentialActions = append(o.potentialActions, action)
	return o
}

// AddOpenUriAction adds an "Open URI" action to the message card.
func (o *Options) AddOpenUriAction(name, uri string) *Options {
	action := map[string]any{
		"@type": "OpenUri",
		"name":  name,
		"targets": []map[string]any{
			{
				"os":  "default",
				"uri": uri,
			},
		},
	}
	return o.PotentialAction(action)
}

// AddHttpPostAction adds an HTTP POST action to the message card.
func (o *Options) AddHttpPostAction(name, target string, body map[string]any) *Options {
	action := map[string]any{
		"@type":  "HttpPOST",
		"name":   name,
		"target": target,
		"body":   body,
	}
	return o.PotentialAction(action)
}

// MarshalJSON implements json.Marshaler.
func (o *Options) MarshalJSON() ([]byte, error) {
	return json.Marshal(o.options)
}
