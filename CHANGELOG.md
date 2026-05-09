# Changelog

## [v1.3.0] - 2026-05-09

### Added

- `Client.GenerateSandboxEditorOpenUrl(ctx, sandboxName)` — calls `POST /ns/{namespace}/bot-provider/{name}/sandbox/{sandbox_name}/editor/open-url` and returns the one-time sandbox editor open URL string.

## [v1.2.0] - 2026-04-17

### Breaking Changes

- `BotAgent.SendMessage` and `Client.SendMessage` signatures changed: the positional `isDebug bool` parameter is replaced by `*client.MessageRequestOptions`. Pass `&client.MessageRequestOptions{IsDebug: true}` or `nil` for default behaviour.
- `BotAgent.NewStreamer` and `Client.NewStreamer` signatures changed: a `*client.MessageRequestOptions` parameter is added as the third argument. Pass `nil` for default behaviour.

### Added

- `client.MessageRequestOptions` struct for per-request configuration (`IsDebug`, `UserIdentityHint`).
- Support for `X-ASGARD-USER-IDENTITY-HINT` request header via `MessageRequestOptions.UserIdentityHint` on both REST and SSE transports.
- `models.SseEventTypeToolCallConsent` SSE event type (`asgard.tool_call.consent`).
- `models.GenericBotSseEventFactToolCallConsent` fact struct (emitted with the new consent event type).
- `models.PendingToolCall` — describes a single tool invocation awaiting user consent.
- `models.ToolCallConsentRequest` — included in `GenericBotReply` when bot requires consent before executing tool calls.
- `models.ToolCallConsentResponseItem` and `models.ToolCallConsentResult` — used when replying with `PostBackActionResponseToolCallConsent`.
- `models.PostBackActionResponseToolCallConsent` post-back action constant.
- `models.GenericBotMessage.ToolCallConsents` field for sending consent decisions back to the server.
- `models.GenericBotReply.ToolCallConsentRequest` field for receiving consent requests from the server.

### Fixed

- Typo in constant name: `PostBackActionResetChanel` → `PostBackActionResetChannel`.
