# Changelog

## [v1.5.5] - 2026-05-28

### Added

- `MessageTemplateTable.Sql` (`pkg/models/template.go`) — original SQL query whose result populates the table. Set by the new `show_result_set_table` builtin tool in asgard-core's stream-llm-completion-message processor. Additive optional field — safe for existing consumers.
- `MessageTemplateTable.SqlExplanation` (`pkg/models/template.go`) — short, plain-language summary of what the SQL query does, in the user's conversation language. Set alongside `Sql`. Additive.

### Notes

- Pairs with the asgard-core `semanticLayers.dataVisualization` opt-in, which adds two builtin tools (`show_result_set_table`, `show_vega_visualization`) that push UI-rendered query results to the front end. Templates emitted by `show_result_set_table` carry the SQL context on the `table` sub-object so the UI can display it alongside the rendered rows.

## [v1.5.4] - 2026-05-26

### Added

- `ToolCall.Reason` (`pkg/models/sse_event.go`) — short, user-facing rationale the agent supplied for the call. Sourced from the `_reason` meta property the agent attaches to every builtin tool input in asgard-core. Empty when the agent didn't supply one. Additive JSON field — safe for existing consumers.
- `PendingToolCall.Reason` (`pkg/models/consent.go`) — same field surfaced on consent-pending tool calls so the consent UI can display *why* the agent wants to run each call. Additive.

### Notes

- Pairs with the asgard-core change that adds `_reason` to every builtin tool's JSON-Schema and propagates it through `corepb.ToolCall.reason` → SSE events. Tools that don't carry `_reason` simply emit `reason == ""`.

## [v1.5.3] - 2026-05-20

### Added

- `client.APIError` — typed error returned by every HTTP method when the server responds with a non-2xx status or an `isSuccess=false` envelope. Fields: `StatusCode` (int), `ErrorCode` (server category, e.g. `"FAILED_PRECONDITION"`, `"INVALID_ARGUMENT"`), `Message`, `Op`. Inspect via `errors.As(err, &apiErr)`.
- `client.IsBadRequest`, `client.IsUnauthorized`, `client.IsForbidden`, `client.IsNotFound`, `client.IsConflict`, `client.IsPreconditionFailed` — predicate helpers over `*APIError`.
- `client.StatusCode(err) int` — extracts the HTTP status from any error that wraps `*APIError` (returns 0 otherwise).
- README: new **Error handling** section showing `errors.As(&apiErr)` and the `Is<Status>` helpers, with a worked example for `SandboxHeartbeat`'s 412 / 400 responses.

### Changed

- All `BotProviderClient` and `SourceSetClient` methods now return `*client.APIError` (via the `error` interface) on non-2xx responses, replacing the previous opaque `fmt.Errorf("... failed (%d): %s", ...)` strings. Callers that only do `if err != nil { ... }` are unaffected. The error string format changes slightly to include `errorCode` after the status code — the old format was never part of the documented contract.
- Centralized response decoding into internal `decodeAPIResponse[T]` (JSON-body methods) and `decodeAPIError` (binary-body methods `SandboxFsRead` / `SourceSetClient.ReadFile`). No behavior change beyond the typed error.

### Notes

- SSE connect errors from `NewStreaming` / `NewStreamer` are still surfaced as the previous `ConnectionError` shape — the underlying SSE library does not expose the initial HTTP response, so APIError parity is not yet possible there.
- Pairs with asgard-core change that tags sandbox 412 responses with `errorCode = "FAILED_PRECONDITION"` (previously `"INTERNAL"`), so callers can branch on `apiErr.ErrorCode` as well as status.

## [v1.5.2] - 2026-05-20

### Added

- `models.MessageTemplateTypeAttachment` (`"ATTACHMENT"`) — new template type for rendering a list of attachment chips.
- `models.MessageTemplateAttachment` struct (fields: `Title`, `Text`, `DefaultAction`, optional `DownloadAction`). `DefaultAction` fires when the chip body is tapped; `DownloadAction`, when set, renders an additional download button on the right. Both actions reuse the existing `MessageTemplateAction` (`URI` / `MESSAGE` / `EMIT`).
- `MessageTemplate.Attachments` field (`*[]MessageTemplateAttachment`, `json:"attachments,omitempty"`) used when `Type == MessageTemplateTypeAttachment`.

## [v1.5.1] - 2026-05-19

### Added

- `MessageRequestOptions.BypassToolCallConsent` field. When `true`, the SSE endpoint (`NewStreaming` / `NewStreamer`) auto-approves every tool call in the request without writing to the persistent `tool_call_allow_list`. Mirrors the new asgard-core `bypass_tool_call_consent` query param. The REST endpoint (`SendMessage`) does not yet accept this server-side and will ignore it.

### Fixed

- `MessageRequestOptions.IsDebug` is now honored by `NewStreaming` (SSE). Previously the SSE code path built the URL without query params and silently ignored this option, contradicting the docstring. REST behavior is unchanged.
- CHANGELOG: the prior release header was mislabeled `v2.0.0`; corrected to `v1.5.0` to match the actual published git tag.

## [v1.5.0] - 2026-05-11

### Breaking Changes

- `Client` interface renamed to `BotProviderClient`. Update all type references and variable declarations accordingly.
- `BotProviderClient` concrete struct is now unexported (`botProviderClient`). Callers must use the `BotProviderClient` interface; direct struct instantiation is no longer possible. Constructors `NewBotProviderClient` and `NewBotProviderClientWithConfig` now return `BotProviderClient` (interface) instead of `*BotProviderClient`.
- `BotAgent` interface removed. Replace all `client.BotAgent` references with `client.BotProviderClient`.
- `FunctionAgent` interface removed. Replace all `client.FunctionAgent` references with `client.BotProviderClient`.
- `NewBotAgent`, `NewBotAgentWithConfig`, `NewFunctionAgent`, `NewFunctionAgentWithConfig` removed. Use `NewBotProviderClient` or `NewBotProviderClientWithConfig` instead.
- `SourceSetClient` is now an interface (was a concrete struct). `NewSourceSetClient` and `NewSourceSetClientWithConfig` now return `SourceSetClient` (interface) instead of `*SourceSetClient`. Direct struct field access is no longer possible.

### Removed

- `cmd/edgeserver-cli` — the reference CLI is no longer part of this module.

## [v1.4.0] - 2026-05-11

### Breaking Changes

- `Client` interface gains four new methods: `SandboxFsList`, `SandboxFsRead`, `SandboxFsWrite`, `SandboxHeartbeat`. Any external implementation of `Client` must add these methods.

### Added

- `Client.SandboxFsList(ctx, sandboxName, path)` — calls `GET /sandbox/{name}/fs/list` and returns `*models.SandboxFsListResult`.
- `Client.SandboxFsRead(ctx, sandboxName, path, offsetBytes, limitBytes)` — calls `GET /sandbox/{name}/fs/file` and returns the raw file bytes plus `*models.SandboxFsReadMeta` (total size and truncation flag from response headers). `offsetBytes` and `limitBytes` may be `nil` to use server defaults.
- `Client.SandboxFsWrite(ctx, sandboxName, path, reader, filename, mode, createOnly)` — calls `PUT /sandbox/{name}/fs/file` via `multipart/form-data` (field `"file"`). `mode` may be `nil` to use the server default (0644). Returns `*models.SandboxFsWriteResult`.
- `Client.SandboxHeartbeat(ctx, sandboxName)` — calls `POST /sandbox/{name}/heartbeat` to extend the sandbox lease and returns `*models.SandboxHeartbeatResult` with the new `ShutdownAt` timestamp.
- `models.SandboxFsDirEntry` — entry struct returned by `SandboxFsList` (fields: `Name`, `IsDir`, `SizeBytes`, `MtimeUnix`, `Mode`).
- `models.SandboxFsListResult` — list response (fields: `Entries`, `Truncated`).
- `models.SandboxFsReadMeta` — metadata returned alongside file bytes by `SandboxFsRead` (fields: `TotalBytes`, `Truncated`).
- `models.SandboxFsWriteResult` — write response (field: `BytesWritten`).
- `models.SandboxHeartbeatResult` — heartbeat response (field: `ShutdownAt` RFC3339 string).
- `client.SourceSetClient` — new client for SourceSet volume endpoints. Construct with `NewSourceSetClient(host, namespace, name, apiKey)` or `NewSourceSetClientWithConfig(*SourceSetConfig)`.
- `client.SourceSetConfig` — configuration struct for `SourceSetClient` (fields: `HTTPClient`, `EdgeServerHost`, `Namespace`, `SourceSetName`, `SourceSetApiKey`, `Headers`).
- `SourceSetClient.ListDirectory(ctx, path, page, pageSize)` — `GET /volume/list`; `page` and `pageSize` may be `nil`.
- `SourceSetClient.Stat(ctx, path)` — `GET /volume/stat`.
- `SourceSetClient.ReadFile(ctx, path, offsetBytes, limitBytes)` — `GET /volume/file`; returns raw `[]byte`.
- `SourceSetClient.WriteFile(ctx, path, reader, filename)` — `PUT /volume/file` via `multipart/form-data` (field `"file"`).
- `SourceSetClient.MakeDirectory(ctx, path)` — `POST /volume/mkdir`.
- `SourceSetClient.Remove(ctx, path)` — `DELETE /volume/item`.
- `SourceSetClient.RemoveAll(ctx, path)` — `DELETE /volume/all`.
- `models.SourceSetDirEntry` — entry struct (fields: `Name`, `IsDir`, `SizeBytes`, `MtimeUnix`).
- `models.SourceSetListDirectoryResult` — list response (fields: `Entries`, `Paging`).
- `models.SourceSetPaging` — pagination info (fields: `Index`, `Size`, `Total`).
- `models.SourceSetStatResult` — stat response (fields: `Exists`, `IsDir`, `SizeBytes`, `MtimeUnix`, `Etag`).
- `models.SourceSetWriteFileResult` — write response (field: `BytesWritten`).

## [v1.3.1] - 2026-05-10

### Added

- `models.SseEventTypeSandboxLaunch` (`asgard.sandbox.launch`) and `models.SseEventTypeSandboxReady` (`asgard.sandbox.ready`) SSE event type constants.
- `models.GenericBotSseEventFactSandboxLaunch` and `models.GenericBotSseEventFactSandboxReady` fact structs (fields: `SandboxName`, `BlueprintName`).
- `GenericBotSseEventFact.SandboxLaunch` and `GenericBotSseEventFact.SandboxReady` fields.

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
