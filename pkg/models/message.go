package models

// GenericBotMessage represents a message sent from client to the Edge Server
type GenericBotMessage struct {
	CustomChannelId string                 `json:"customChannelId"`
	CustomMessageId string                 `json:"customMessageId"`
	Text            string                 `json:"text,omitempty"`
	Action          PostBackAction         `json:"action"`
	BlobIds         []string               `json:"blobIds,omitempty"`
	Payload         map[string]interface{} `json:"payload,omitempty"`
}

// PostBackAction defines the action type for a message
type PostBackAction string

const (
	PostBackActionNone        PostBackAction = "NONE"
	PostBackActionResetChanel PostBackAction = "RESET_CHANNEL"
)

// BufferedMessage represents a message returned from the Edge Server
type BufferedMessage struct {
	MessageId              string           `json:"messageId"`
	ReplyToCustomMessageId string           `json:"replyToCustomMessageId"`
	Text                   string           `json:"text"`
	Payload                interface{}      `json:"payload"`
	IsDebug                bool             `json:"isDebug"`
	Idx                    *int             `json:"idx"`
	Template               *MessageTemplate `json:"template"`
}
