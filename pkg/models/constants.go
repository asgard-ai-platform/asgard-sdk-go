package models

// SseEventType SSE Event Type
type SseEventType string

const (
	SseEventTypeRunInit          SseEventType = "asgard.run.init"
	SseEventTypeRunDone          SseEventType = "asgard.run.done"
	SseEventTypeRunError         SseEventType = "asgard.run.error"
	SseEventTypeProcessStart     SseEventType = "asgard.process.start"
	SseEventTypeProcessComplete  SseEventType = "asgard.process.complete"
	SseEventTypeMessageStart     SseEventType = "asgard.message.start"
	SseEventTypeMessageDelta     SseEventType = "asgard.message.delta"
	SseEventTypeMessageComplete  SseEventType = "asgard.message.complete"
	SseEventTypeToolCallStart    SseEventType = "asgard.tool_call.start"
	SseEventTypeToolCallComplete SseEventType = "asgard.tool_call.complete"
)

// Message Template Type
type MessageTemplateType string

const (
	MessageTemplateTypeText     MessageTemplateType = "TEXT"
	MessageTemplateTypeImage    MessageTemplateType = "IMAGE"
	MessageTemplateTypeVideo    MessageTemplateType = "VIDEO"
	MessageTemplateTypeAudio    MessageTemplateType = "AUDIO"
	MessageTemplateTypeLocation MessageTemplateType = "LOCATION"
	MessageTemplateTypeButton   MessageTemplateType = "BUTTON"
	MessageTemplateTypeCarousel MessageTemplateType = "CAROUSEL"
	MessageTemplateTypeChart    MessageTemplateType = "CHART"
	MessageTemplateTypeTable    MessageTemplateType = "TABLE"
)

// Message Template Action Type
type MessageTemplateActionType string

const (
	MessageTemplateActionTypeMessage MessageTemplateActionType = "MESSAGE"
	MessageTemplateActionTypeUri     MessageTemplateActionType = "URI"
	MessageTemplateActionTypeEmit    MessageTemplateActionType = "EMIT"
)

// Image Aspect Ratio
type ImageAspectRatio string

const (
	ImageAspectRatioRectangle ImageAspectRatio = "rectangle"
	ImageAspectRatioSquare    ImageAspectRatio = "square"
)

// Image Size
type ImageSize string

const (
	ImageSizeCover   ImageSize = "cover"
	ImageSizeContain ImageSize = "contain"
)

// MessageTemplateRowType defines the row type for table templates
type MessageTemplateRowType string

const (
	MessageTemplateRowTypeObject MessageTemplateRowType = "OBJECT"
	MessageTemplateRowTypeArray  MessageTemplateRowType = "ARRAY"
)

// MessageTemplateTableColumnFormat defines the format for table columns
type MessageTemplateTableColumnFormat string

const (
	MessageTemplateTableColumnFormatDate     MessageTemplateTableColumnFormat = "DATE"
	MessageTemplateTableColumnFormatDateTime MessageTemplateTableColumnFormat = "DATE_TIME"
	MessageTemplateTableColumnFormatCurrency MessageTemplateTableColumnFormat = "CURRENCY"
)
