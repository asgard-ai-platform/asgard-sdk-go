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
	// Canvas events (additive) carry a visual the agent draws: one HTML/SVG
	// fragment, delivered as it is written so a client can show it taking shape.
	// `start` opens the block, each `delta` carries the markup that became
	// available (in the message's `text`, appended by the client exactly like a
	// text delta), and `complete` carries the whole fragment as a CANVAS template.
	//
	// The complete is authoritative and self-sufficient: a client that ignored or
	// missed every delta renders the same document from it alone. Rejoining a
	// channel's history replays only the complete, never the deltas.
	//
	// A `complete` with NO template means the canvas could not be rendered. It
	// exists to close the block that `start` opened — discard it rather than
	// keeping whatever partial markup arrived.
	//
	// Clients that don't render canvases can ignore all three.
	SseEventTypeMessageCanvasStart    SseEventType = "asgard.message.canvas.start"
	SseEventTypeMessageCanvasDelta    SseEventType = "asgard.message.canvas.delta"
	SseEventTypeMessageCanvasComplete SseEventType = "asgard.message.canvas.complete"
	SseEventTypeToolCallStart         SseEventType = "asgard.tool_call.start"
	SseEventTypeToolCallComplete      SseEventType = "asgard.tool_call.complete"
	SseEventTypeToolCallConsent       SseEventType = "asgard.tool_call.consent"
	SseEventTypeCompletionModelUsage  SseEventType = "asgard.completion_model.usage"
	SseEventTypeSandboxLaunch         SseEventType = "asgard.sandbox.launch"
	SseEventTypeSandboxReady          SseEventType = "asgard.sandbox.ready"
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
	// SseEventTypeChannelStatusUpdate is emitted when the agent declares where the
	// work stands (NEEDS_INPUT / COMPLETED), so a client can update the badge beside
	// the conversation in place.
	//
	// Live-only: never replayed when rejoining a channel's history. A client that
	// (re)opens a conversation reads the current value from
	// ChannelMetadata.ConversationStatus, then keeps it up to date from this event.
	SseEventTypeChannelStatusUpdate SseEventType = "asgard.channel.status.update"
	// SseEventTypePromptSuggestion carries a prediction of what the user is
	// likely to send next, for a client to offer as accept-able placeholder text
	// in its input box. It arrives after the reply, before the run's terminal
	// event.
	//
	// Live-only: it is never replayed when rejoining a channel's history, since a
	// prediction from an earlier turn is stale. Expect at most one per reply, and
	// often none — a prediction is only offered when the next step is clear.
	SseEventTypePromptSuggestion SseEventType = "asgard.prompt_suggestion"
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
	// MessageTemplateTypeQuestion renders a multiple-choice question form the
	// user can answer, skip, or ignore entirely. Answering it is NOT a protocol
	// step: the client turns the selections into ordinary message text and
	// posts it as the next user message.
	MessageTemplateTypeQuestion MessageTemplateType = "QUESTION"
	// MessageTemplateTypeCanvas carries a visual the agent drew as an HTML/SVG
	// fragment. See MessageTemplateCanvas for the isolation a renderer MUST apply.
	MessageTemplateTypeCanvas MessageTemplateType = "CANVAS"
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
