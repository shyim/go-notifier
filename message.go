package notifier

// MessageInterface represents a message that can be sent via a transport.
type MessageInterface interface {
	// GetRecipientId returns the recipient identifier.
	GetRecipientId() string
	// GetSubject returns the message subject/content.
	GetSubject() string
	// GetOptions returns the message options for a specific transport.
	// The key should match the transport's scheme (e.g., "telegram", "slack").
	GetOptions(transportKey string) MessageOptionsInterface
	// GetTransport returns the transport name, if specified.
	GetTransport() string
}

// MessageOptionsInterface represents options for a message.
type MessageOptionsInterface interface {
	// ToMap converts options to a map.
	ToMap() map[string]any
	// GetRecipientId returns the recipient ID from options, if any.
	GetRecipientId() string
}

// ChatMessage represents a chat message (e.g., Telegram, Slack).
type ChatMessage struct {
	subject   string
	options   map[string]MessageOptionsInterface
	transport string
}

func NewChatMessage(subject string) *ChatMessage {
	return &ChatMessage{
		subject: subject,
		options: make(map[string]MessageOptionsInterface),
	}
}

func (m *ChatMessage) GetRecipientId() string {
	// Check all options for a recipient ID
	for _, opt := range m.options {
		if id := opt.GetRecipientId(); id != "" {
			return id
		}
	}
	return ""
}

func (m *ChatMessage) GetSubject() string {
	return m.subject
}

// GetOptions returns options for a specific transport key.
func (m *ChatMessage) GetOptions(transportKey string) MessageOptionsInterface {
	return m.options[transportKey]
}

func (m *ChatMessage) GetTransport() string {
	return m.transport
}

// WithOptions adds transport-specific options.
// The key should be the transport scheme (e.g., "telegram", "slack").
func (m *ChatMessage) WithOptions(transportKey string, options MessageOptionsInterface) *ChatMessage {
	m.options[transportKey] = options
	return m
}

// Transport sets the specific transport to use.
func (m *ChatMessage) Transport(transport string) *ChatMessage {
	m.transport = transport
	return m
}

// Subject sets the message subject.
func (m *ChatMessage) Subject(subject string) *ChatMessage {
	m.subject = subject
	return m
}

// SentMessage represents a message that has been sent.
type SentMessage struct {
	original  MessageInterface
	transport string
	messageID string
	info      map[string]any
}

func NewSentMessage(original MessageInterface, transport string, info ...map[string]any) *SentMessage {
	var i map[string]any
	if len(info) > 0 {
		i = info[0]
	} else {
		i = make(map[string]any)
	}
	return &SentMessage{
		original:  original,
		transport: transport,
		info:      i,
	}
}

func (s *SentMessage) GetOriginalMessage() MessageInterface {
	return s.original
}

func (s *SentMessage) GetTransport() string {
	return s.transport
}

func (s *SentMessage) SetMessageID(id string) {
	s.messageID = id
}

func (s *SentMessage) GetMessageID() string {
	return s.messageID
}

func (s *SentMessage) GetInfo(key ...string) any {
	if len(key) == 0 {
		return s.info
	}
	return s.info[key[0]]
}

func (s *SentMessage) SetInfo(key string, value any) {
	s.info[key] = value
}
