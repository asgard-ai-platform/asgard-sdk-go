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
	// Deprecated
	Description *string `json:"description,omitempty"`
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
