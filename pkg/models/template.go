package models

// MessageTemplate represents a structured message template
type MessageTemplate struct {
	Type                 MessageTemplateType           `json:"type"`
	Text                 *string                       `json:"text,omitempty"`
	QuickReplies         []QuickReply                  `json:"quickReplies,omitempty"`
	OriginalContentUrl   *string                       `json:"originalContentUrl,omitempty"`
	PreviewImageUrl      *string                       `json:"previewImageUrl,omitempty"`
	Duration             *int64                        `json:"duration,omitempty"`
	Title                *string                       `json:"title,omitempty"`
	Latitude             *float64                      `json:"latitude,omitempty"`
	Longitude            *float64                      `json:"longitude,omitempty"`
	ThumbnailImageUrl    *string                       `json:"thumbnailImageUrl,omitempty"`
	ImageAspectRatio     *ImageAspectRatio             `json:"imageAspectRatio,omitempty"`
	ImageSize            *ImageSize                    `json:"imageSize,omitempty"`
	ImageBackgroundColor *string                       `json:"imageBackgroundColor,omitempty"`
	Buttons              *[]MessageTemplateButton      `json:"buttons,omitempty"`
	DefaultAction        *MessageTemplateAction        `json:"defaultAction,omitempty"`
	Columns              *[]MessageTemplateColumn      `json:"columns,omitempty"`
	Data                 *interface{}                  `json:"data,omitempty"`
	ChartOptions         *[]MessageTemplateChartOption `json:"chartOptions,omitempty"`
	DefaultChart         *string                       `json:"defaultChart,omitempty"`
	Table                *MessageTemplateTable         `json:"table,omitempty"`
	References           []MessageTemplateReference    `json:"references,omitempty"`
	Attachments          *[]MessageTemplateAttachment  `json:"attachments,omitempty"`
	Questions            []MessageTemplateQuestion     `json:"questions,omitempty"`
	Canvas               *MessageTemplateCanvas        `json:"canvas,omitempty"`
	// Deprecated
	Description *string `json:"description,omitempty"`
}

// MessageTemplateCanvas is the payload of a CANVAS template: one HTML/SVG
// fragment the agent drew, to be rendered as a visual card.
//
// The markup is UNTRUSTED. It is generated per-conversation and may contain
// <style> and <script>, so a client that injects it into its own page hands that
// page's origin — its cookies, its storage, its DOM — to content it did not
// author. It MUST be rendered in an isolated browsing context with scripting
// allowed but same-origin access denied (in a browser: an iframe sandboxed
// WITHOUT same-origin, fed by srcdoc). That isolation also keeps the fragment
// from reaching the network, which is expected: a canvas is authored to be
// self-contained, with no external stylesheets, fonts, images or scripts.
//
// Html is the COMPLETE fragment and is authoritative. The same markup also
// arrives beforehand as incremental canvas deltas, purely so the card can be
// shown taking shape; a client that ignored them renders correctly from this
// field alone.
type MessageTemplateCanvas struct {
	Html string `json:"html"`
}

// MessageTemplateQuestion is one multiple-choice question in a QUESTION
// template. The agent supplies 1-4 of them per card, each with 2-4 options.
//
// Answering is deliberately NOT a protocol handshake: the client composes the
// selections into plain text and posts them as an ordinary next user message,
// so a user is free to ignore the card and type something else instead. The
// run that produced the card is already finished by the time it renders.
type MessageTemplateQuestion struct {
	// Question is the full question text to display.
	Question string `json:"question"`
	// Header is a short label for the question (the agent keeps it under ~12
	// characters), suitable for a chip or column heading.
	Header string `json:"header"`
	// MultiSelect allows more than one option to be chosen.
	MultiSelect bool `json:"multiSelect"`
	// Options are the offered choices. A client SHOULD also offer a free-text
	// escape hatch — the answer text is unconstrained, the options are only a
	// shortcut.
	Options []MessageTemplateQuestionOption `json:"options"`
}

// MessageTemplateQuestionOption is one choice of a MessageTemplateQuestion.
type MessageTemplateQuestionOption struct {
	// Label is the short display text and the value a client puts in the
	// composed answer.
	Label string `json:"label"`
	// Description explains what picking this option means. May be empty.
	Description string `json:"description,omitempty"`
}

// QuickReply represents a quick reply option
type QuickReply struct {
	Text string `json:"text"`
}

// MessageTemplateButton represents a button in a message template
type MessageTemplateButton struct {
	Label  string                `json:"label"`
	Action MessageTemplateAction `json:"action"`
}

// MessageTemplateColumn represents a column in a carousel template
type MessageTemplateColumn struct {
	Title                string                  `json:"title"`
	Text                 string                  `json:"text"`
	ThumbnailImageUrl    *string                 `json:"thumbnailImageUrl,omitempty"`
	ImageAspectRatio     *ImageAspectRatio       `json:"imageAspectRatio,omitempty"`
	ImageSize            *ImageSize              `json:"imageSize,omitempty"`
	ImageBackgroundColor *string                 `json:"imageBackgroundColor,omitempty"`
	Buttons              []MessageTemplateButton `json:"buttons"`
	DefaultAction        *MessageTemplateAction  `json:"defaultAction,omitempty"`
}

// MessageTemplateAction represents an action associated with a button or default action
type MessageTemplateAction struct {
	Type      MessageTemplateActionType `json:"type"`
	Text      *string                   `json:"text"`
	Uri       *string                   `json:"uri"`
	EventName *string                   `json:"eventName,omitempty"` // Superset: from data-insight-api
	Payload   *interface{}              `json:"payload"`
}

// MessageTemplateChartOption represents a chart option
type MessageTemplateChartOption struct {
	Type  string                 `json:"type"`
	Title string                 `json:"title"`
	Spec  map[string]interface{} `json:"spec"`
}

// MessageTemplateTable represents a table template
type MessageTemplateTable struct {
	RowType    MessageTemplateRowType          `json:"rowType"`
	Columns    []MessageTemplateTableColumn    `json:"columns"`
	Pagination *MessageTemplateTablePagination `json:"pagination,omitempty"`
	Data       []interface{}                   `json:"data"`
	// Sql is the original SQL query whose result populates this table.
	// Set by the show_result_set_table builtin tool in stream-llm-completion-message.
	Sql *string `json:"sql,omitempty"`
	// SqlExplanation is a short, plain-language summary of what the SQL query
	// does, in the user's conversation language. Set alongside Sql.
	SqlExplanation *string `json:"sqlExplanation,omitempty"`
}

// MessageTemplateTableColumn represents a column in a table template
type MessageTemplateTableColumn struct {
	Header string                            `json:"header"`
	Key    string                            `json:"key"`
	Format *MessageTemplateTableColumnFormat `json:"format,omitempty"`
}

// MessageTemplateTablePagination represents pagination settings for a table
type MessageTemplateTablePagination struct {
	Size int `json:"size"`
}

// MessageTemplateReference represents a reference/citation in a message
type MessageTemplateReference struct {
	Title string `json:"title"`
	Uri   string `json:"uri"`
}

// MessageTemplateAttachment represents a single attachment chip in an ATTACHMENT template.
// DefaultAction fires when the chip body is tapped; DownloadAction, when set, renders an
// additional download button on the right.
type MessageTemplateAttachment struct {
	Title          string                 `json:"title"`
	Text           string                 `json:"text"`
	DefaultAction  MessageTemplateAction  `json:"defaultAction"`
	DownloadAction *MessageTemplateAction `json:"downloadAction,omitempty"`
}
