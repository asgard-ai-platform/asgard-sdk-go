package models

// SseEventType SSE Event Type
type SseEventType string

const (
	SseEventTypeRunInit         SseEventType = "asgard.run.init"
	SseEventTypeRunDone         SseEventType = "asgard.run.done"
	SseEventTypeRunError        SseEventType = "asgard.run.error"
	SseEventTypeProcessStart    SseEventType = "asgard.process.start"
	SseEventTypeProcessComplete SseEventType = "asgard.process.complete"
	SseEventTypeMessageStart    SseEventType = "asgard.message.start"
	SseEventTypeMessageDelta    SseEventType = "asgard.message.delta"
	SseEventTypeMessageComplete SseEventType = "asgard.message.complete"
	// Thinking events are additive (CLI-driver): extended-thinking blocks stream
	// and complete separately from the assistant message. Clients that don't
	// render thinking can ignore them.
	SseEventTypeMessageThinkingStart    SseEventType = "asgard.message.thinking.start"
	SseEventTypeMessageThinkingDelta    SseEventType = "asgard.message.thinking.delta"
	SseEventTypeMessageThinkingComplete SseEventType = "asgard.message.thinking.complete"
	SseEventTypeToolCallStart           SseEventType = "asgard.tool_call.start"
	SseEventTypeToolCallComplete        SseEventType = "asgard.tool_call.complete"
	SseEventTypeToolCallConsent         SseEventType = "asgard.tool_call.consent"
	SseEventTypeCompletionModelUsage    SseEventType = "asgard.completion_model.usage"
	SseEventTypeSandboxLaunch           SseEventType = "asgard.sandbox.launch"
	SseEventTypeSandboxReady            SseEventType = "asgard.sandbox.ready"
	// Subagent lifecycle events (additive): a subagent is a helper the agent spawns
	// to work on a sub-task. Started fires when it begins running, completed when it
	// finishes. Both carry agentId + parentToolUseId, so a client can maintain a live
	// list of running subagents. Clients that don't track subagents can ignore them.
	SseEventTypeSubagentStart    SseEventType = "asgard.subagent.start"
	SseEventTypeSubagentComplete SseEventType = "asgard.subagent.complete"
	// SseEventTypeMessageUser is the user's own turn, surfaced when replaying a
	// channel's history so a client can render the user side of the conversation.
	// It is delivered only on a history rejoin, not while a turn is streaming live.
	SseEventTypeMessageUser SseEventType = "asgard.message.user"
	// SseEventTypeChannelTitleUpdate is emitted when the conversation title
	// changes, so a client can update the channel's displayed title in place.
	SseEventTypeChannelTitleUpdate SseEventType = "asgard.channel.title.update"
)

// Message Template Type
type MessageTemplateType string

const (
	MessageTemplateTypeText       MessageTemplateType = "TEXT"
	MessageTemplateTypeImage      MessageTemplateType = "IMAGE"
	MessageTemplateTypeVideo      MessageTemplateType = "VIDEO"
	MessageTemplateTypeAudio      MessageTemplateType = "AUDIO"
	MessageTemplateTypeLocation   MessageTemplateType = "LOCATION"
	MessageTemplateTypeButton     MessageTemplateType = "BUTTON"
	MessageTemplateTypeCarousel   MessageTemplateType = "CAROUSEL"
	MessageTemplateTypeChart      MessageTemplateType = "CHART"
	MessageTemplateTypeTable      MessageTemplateType = "TABLE"
	MessageTemplateTypeAttachment MessageTemplateType = "ATTACHMENT"
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
