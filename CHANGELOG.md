# Changelog

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
