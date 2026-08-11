# Changelog

## [Unreleased]

### Added — QUESTION message template

One new message-template type (`pkg/models/template.go`, `pkg/models/constants.go`):

- `QUESTION` — a multiple-choice question card the agent sends when it needs a
  decision from the user rather than a guess. `MessageTemplate.Questions` carries
  1–4 `MessageTemplateQuestion` entries, each with a `Question`, a short `Header`,
  a `MultiSelect` flag, and 2–4 `Options` of `Label` + `Description`.

Two behaviours worth designing around:

- **Answering is not a protocol step.** The card is pure UI. A client turns the
  user's selections into ordinary message text and posts it as the next user
  message — there is no accept/reject wire call, no run to resume, and no state
  to reconcile. The run that produced the card is already finished when it
  renders, so a user is equally free to ignore the card and type something else.
- **Free text always wins.** The options are a shortcut, not a closed set. A
  client should offer a free-text escape hatch and send whatever the user wrote;
  the agent reads the next message as plain prose either way.

Purely additive: existing template types are unchanged, and the card also carries
a plain-text rendering in `Text`, so a client that does not know `QUESTION` falls
back to showing the questions as text rather than an empty bubble. Requires a
backend carrying the matching template.

## [v1.7.1] - 2026-08-11

### Added — predicted next user message

One new SSE event (`pkg/models/sse_event.go`, `pkg/models/constants.go`):

- `asgard.prompt_suggestion` (`fact.promptSuggestion`) — a prediction of what the
  user is likely to send next, carrying a short `suggestion` a client can offer as
  accept-able placeholder text in its input box. Arrives after the reply and
  before the run's terminal event.

Two behaviours worth designing around:

- **Live-only.** It is never replayed when rejoining a channel's history — a
  prediction from an earlier turn is stale, so a rejoining client should keep
  whatever placeholder it already shows rather than expect one.
- **Optional and at most one per reply.** A prediction is only offered when the
  next step is clear, so most replies carry none. Treat its absence as normal,
  not as a missed event, and never block on it.

Purely additive: existing events and facts are unchanged; clients that ignore the
new event type / fact field are unaffected. Requires a backend carrying the
matching event.

## [v1.7.0] - 2026-08-10

### Changed — BREAKING: SourceSet volume client gains copy/move and file metadata

The SourceSet volume API is now the backend for a file explorer, so this client
covers the same ground the sandbox filesystem client already did. Two existing
methods change shape; the SourceSet and Sandbox clients are now symmetric.
Requires an edge server carrying the matching volume API.

**Breaking — two signatures:**

- `SourceSetClient.ReadFile(...)` returns `([]byte, *models.SourceSetReadMeta,
  error)` instead of `([]byte, error)`. The metadata carries `TotalBytes` (the
  file's full size, independent of any requested range) and `Truncated`, so a
  caller can tell a bounded read from a whole file. Mirrors `SandboxFsRead`.
  Migration: `b, err := c.ReadFile(...)` → `b, _, err := c.ReadFile(...)`.
- `SourceSetClient.WriteFile(...)` takes two more arguments, `mode *uint32` and
  `createOnly bool`. Mirrors `SandboxFsWrite`.
  Migration: `c.WriteFile(ctx, p, r, name)` → `c.WriteFile(ctx, p, r, name, nil,
  false)`.

**Breaking — the interface gained methods.** Anything implementing
`SourceSetClient` (a test fake, a mock) must add `Copy` and `Move`.

**Added:**

- `Copy(ctx, src, dst string, overwrite bool) (*models.SourceSetCopyResult,
  error)`: duplicates `src` to `dst`, recursing when `src` is a directory, and
  reports `BytesCopied`. Without `overwrite`, an existing destination is a 409
  `*APIError` rather than a silent replacement.
- `Move(ctx, src, dst string, overwrite bool) error`: relocates `src` to `dst`.
  A rename is a move within the same parent directory. Same 409 semantics.
- `mode` on write, and `createOnly` — which fails with 409 instead of truncating
  a file that already exists, so "create a new file" and "save over one" are no
  longer the same request.
- `SourceSetDirEntry.Mode` and `SourceSetStatResult.Mode` (Unix permission bits).
- `models.SourceSetReadMeta` and `models.SourceSetCopyResult`.

`ReadFile` now sends `offset_bytes` / `limit_bytes` to match the sandbox
filesystem API. The server still accepts the older `offset` / `limit`, so an
older client keeps working against a newer server.

There is deliberately **no** `Watch`: the volume is served by several replicas,
so a filesystem watch registered on one would not observe writes served by
another. Callers refresh explicitly.

## [v1.6.10] - 2026-07-28

### Added — stop an in-flight run

Additive, backward-compatible.

- New `BotProviderClient.SuspendChannel(ctx, customChannelID, *SuspendOptions)
  error` (`pkg/client/bot_provider.go`): stops the channel's in-flight run.
  Until now there was no way to do this — a run continues in the background
  after a streamer is closed, since closing one only stops watching, so a user
  pressing "stop" could not actually stop the work.
- New `SuspendOptions` (`pkg/client/options.go`): `RequestID` pins the suspend
  to one specific run so a caller cannot accidentally stop a newer one that
  replaced it; `Force` abandons the run immediately instead of letting it stop
  gracefully (reach for it only when a graceful suspend has not taken effect,
  since in-flight work is dropped rather than allowed to wind down);
  `UserIdentityHint` as elsewhere.
- The call returns as soon as the request is registered, **not** when the run
  has stopped. The run stops asynchronously and announces it the same way it
  announces finishing — with a terminal event on the channel's stream — so a
  caller that needs to know the agent is idle should keep reading its streamer
  and wait for that terminal. There is no new event type to handle.
- The conversation is preserved: the transcript survives and the next message
  continues it. Because a stopped run never reaches its end, the turn is rolled
  back — the channel's context is left as the turn found it.
- Idempotent: stopping a run that already finished, or that a newer turn
  superseded, succeeds and does nothing.

## [v1.6.9] - 2026-07-21

### Added — typed sandbox `fs/watch` SSE client

Additive, backward-compatible.

- New `BotProviderClient.SandboxFsWatch(ctx, sandboxName, path, recursive)
  (io.ReadCloser, error)` (`pkg/client/bot_provider.go`): opens the sandbox
  `fs/watch` SSE endpoint (`GET .../sandbox/{name}/fs/watch?path=&recursive=`)
  and returns the raw `text/event-stream` body for the caller to relay or parse.
  Each frame is an `event: change` carrying a JSON `SandboxFsWatchEvent`
  (`op` ∈ CREATE/WRITE/REMOVE/RENAME/CHMOD). A non-2xx response — e.g. an
  `*APIError` with 404 for a missing path — is returned before any stream
  begins, so a relay can surface the right status ahead of the event-stream.
  This completes the fuller sandbox file API introduced in v1.6.8, where the
  `SandboxFsWatchEvent` model shipped but the Go client was deferred. It is a
  deliberately thin passthrough (a watch has no durable resume cursor, unlike
  the message streamer); use an HTTP client without a read timeout since the
  stream is long-lived.

## [v1.6.8] - 2026-07-21

### Added — launched-sandbox discovery, browser handoff, and a fuller sandbox file API

Additive, backward-compatible. Support for surfacing a channel's live Sandboxes,
handing the user into a sandbox browser, and a complete sandbox file-explorer API.

- `models.ChannelMetadata` gains `LaunchedSandboxes []LaunchedSandbox`
  (`pkg/models/edge_api.go`): the channel's currently-live Sandboxes — Ready and
  within their shutdown lease — each with `SandboxName`, `SandboxBlueprintName`,
  `WorkingDirectory`, `EditorServerEnabled`, `BrowserEnabled`. A channel may back
  more than one sandbox, so treat it as a set. Returned by `ChannelMetadata(...)`.
- New `BotProviderClient.GenerateSandboxBrowserOpenUrl(ctx, sandboxName)` mints a
  one-time URL to take over the sandbox's browser (Neko) — for 2FA / sign-in /
  captcha handoff (`POST .../sandbox/{name}/browser/open-url`), mirroring the
  existing editor open-url method.
- Sandbox file API rounded out to a full explorer surface (`pkg/client/bot_provider.go`,
  `pkg/models/sandbox.go`): `SandboxFsStat`, `SandboxFsMkdir`, `SandboxFsRemove`
  (file / empty dir), `SandboxFsRemoveAll` (recursive), `SandboxFsCopy`,
  `SandboxFsMove`, alongside the existing `SandboxFsList` / `SandboxFsRead` /
  `SandboxFsWrite`. New result models `SandboxFsStatResult` / `SandboxFsCopyResult`.
- New edge SSE endpoint `GET .../sandbox/{name}/fs/watch?path=&recursive=` streams
  `SandboxFsWatchEvent` (`op` ∈ CREATE/WRITE/REMOVE/RENAME/CHMOD) for watch-and-reload.
  The event model ships now; a typed Go streaming client is deferred (no Go consumer
  yet — consume the SSE stream directly if needed).
- Agents can now push two card types a client resolves client-side: a browser
  handoff card (URI `sandbox://<name>/open-browser`) and an open-file card (URI
  `sandbox://<name>/open-file?absolute_path=<abs>`), analogous to the existing
  `channel-home://` download card.

## [v1.6.7] - 2026-07-19

### Added — `IsPreset` on completion-model usage

Additive, backward-compatible.

- `models.GenericBotSseEventFactCompletionModelUsage` gains `IsPreset bool`
  (`json:"isPreset"`, `pkg/models/sse_event.go`): the `completion_model.usage` SSE
  event now reports whether the usage came from a platform-provided preset
  completion model, so a client can meter that usage apart from usage on its own
  model key. Existing clients are unaffected.

## [v1.6.6] - 2026-07-15

### Changed — BREAKING: `cwd` download surface renamed to `channel home`

The per-channel file-download surface was named after "cwd" (current working
directory), but it is really the channel's durable **Channel Home** — the
by-channel file plane the server can read even when no live session is running,
independent of any working directory. Renamed across the API; **not
backward-compatible** and released in lockstep with the matching server version.

- Client method `BotProviderClient.DownloadCwdFile(...)` →
  **`DownloadChannelHomeFile(...)`** (`pkg/client/bot_provider.go`); signature
  unchanged.
- Model `models.CwdDownloadMeta` → **`models.ChannelHomeDownloadMeta`**
  (`pkg/models/cwd.go` → `pkg/models/channel_home.go`); fields unchanged.
- Endpoint `GET .../cwd/download` → **`GET .../channel-home/download`**.
- Download-card link URI scheme `cwd://<relative_path>` →
  **`channel-home://<relative_path>`** (update any client-side scheme handling
  that resolves these links).

### Documentation

- Document how to leave an SSE stream early (`README.md`): `stream.Close()` or a
  cancelled `context` detaches the client and stops the transparent-resume loop,
  but a `POST` turn already dispatched keeps running to completion server-side —
  `Close()` does not cancel it. No code behavior changed.
- Clarify the transparent-resume stop conditions and their limitation
  (`README.md`, `pkg/client/streamer.go` doc comments): the resume loop stops on a
  turn terminal (`asgard.run.done` / `asgard.run.error`), `Close()`/context
  cancel, or a non-2xx response; it does not consult `ChannelMetadata.RunState`,
  so a server-side cancellation is recognized only when delivered as one of those
  terminals.

## [v1.6.5] - 2026-07-13

Additive, backward-compatible.

### Added — user-turn + channel-title events, and a channel-metadata client

Two new SSE events (`pkg/models/sse_event.go`, `pkg/models/constants.go`):

- `asgard.message.user` (`fact.messageUser`) — the user's own turn, surfaced when
  replaying a channel's history so a client can render the user side of the
  conversation. Carries `text`, `identityHint` (which end user sent it),
  `customMessageId`, and `blobIds`. Delivered only on a history rejoin, not while a
  turn streams live.
- `asgard.channel.title.update` (`fact.channelTitleUpdate`) — the conversation
  title changed; carries the new `title` so a client can update it in place.

One new client method (`pkg/client/bot_provider.go`):

- `BotProviderClient.ChannelMetadata(ctx, customChannelID)` → `*models.ChannelMetadata`
  (`{customChannelId, title (nullable), runState, lastActivityAt}`), a GET on the
  channel-metadata endpoint so a client entering a chat room can restore its UI
  (e.g. the room title) without opening the stream. Returns an `*APIError`
  (`client.IsNotFound`) when the channel does not exist yet.

Purely additive: existing events, facts, and methods are unchanged; clients that
ignore the new event types / fact fields / method are unaffected.

## [v1.6.4] - 2026-07-11

Naming fix for the subagent lifecycle events added in v1.6.3. If you integrated
against v1.6.3, rename the fields/types; the JSON shape is otherwise identical.

### Changed - subagent lifecycle event/field names

To match every other lifecycle event (`process.start`/`complete`,
`tool_call.start`/`complete`, `message.start`/`complete`), the subagent events
drop the past-tense form:

- `asgard.subagent.started` -> `asgard.subagent.start` (`fact.subagentStarted` -> `fact.subagentStart`)
- `asgard.subagent.completed` -> `asgard.subagent.complete` (`fact.subagentCompleted` -> `fact.subagentComplete`)

Fact shapes are unchanged (agentId, parentToolUseId, subagentType, description /
status, summary). The complete event's `status` is one of `completed`, `failed`,
or `cancelled`.

## [v1.6.3] - 2026-07-11

Additive, backward-compatible.

### Added — subagent lifecycle events

Two new SSE events let a client track subagents the agent spawns to work on
sub-tasks:

- `asgard.subagent.started` (`fact.subagentStarted`) — a subagent began running.
- `asgard.subagent.completed` (`fact.subagentCompleted`) — a subagent finished,
  carrying `status` (e.g. `"completed"`) and `summary` (its final report).

Both facts carry `agentId` (the subagent's id) and `parentToolUseId` (the id of
the tool call that spawned it — the same value that tags the subagent's own
`message.*` / `tool_call.*` events). Maintain a live "running subagents" list keyed
by `agentId`; group a subagent's nested activity by `parentToolUseId`.

Purely additive: existing events are unchanged, and clients that ignore the new
event types / fact fields are unaffected.

## [v1.6.2] - 2026-07-10

Documentation-only. The `toolUseResultSidecar` field added in v1.6.1 is unchanged
(same JSON shape and behavior); its Go doc comment and the changelog below were
reworded to describe it in implementation-agnostic terms.

## [v1.6.1] - 2026-07-10

Additive, backward-compatible.

### Added — `toolUseResultSidecar` on `GenericBotSseEventFactToolCallComplete`

An optional **structured companion** to `toolCallResult`: arbitrary JSON that a
tool may return in addition to the human-readable result string, forwarded
verbatim. It lets clients consume a tool's structured output — for example a
stable id (and status) to correlate and track a multi-step tool's results, e.g.
build a list keyed by id — without parsing the result string. Absent/`nil` when
the tool returned no structured data.

Purely additive: `toolCallResult` is unchanged, and clients that ignore the new
field are unaffected.

## [v1.6.0] - 2026-07-09

Aligns the SDK with asgard-core's CLI-driver migration (edgeserver PR #104): the
BotProvider SSE endpoints moved to a **background-livestream + Last-Event-ID
resume** model. The data-model additions are backward-compatible, but **SSE
streaming behavior changes substantively** — read the two warnings first.

### ⚠️ Behavior change — transparent SSE resume

`NewStreamer` (POST) and the new `NewChannelStreamer` (GET) now **transparently
resume** a dropped stream: when an already-established `200` SSE connection is
interrupted (network blip, gateway/proxy idle-timeout, brief outage), the SDK
reconnects and resubscribes from the last `Last-Event-ID` cursor with backoff,
and `Next()` keeps delivering events as if nothing happened. The run executes in
the background on asgard-core independently of the connection, so no output is
lost. Previously (v1.5.x) any drop ended the stream with an error.

Reconnection is deliberately scoped to a stream that reached `200`:

- A **non-2xx response** (including transient `5xx`/`429`) is **surfaced as
  `*client.APIError` and never silently retried** — mirrors native `EventSource`
  semantics; the caller (or, in a relay, the downstream browser) decides whether
  to re-open. (This also gives `NewStreamer`/`NewChannelStreamer` the same
  `*APIError` you already get from the REST methods on the initial connect.)
- An **initial** connection failure (one that never reached `200`) is surfaced
  immediately with no auto-reconnect (fail fast; also prevents a POST
  double-dispatch).
- A turn's terminal (`asgard.run.done` / `asgard.run.error`) ends the stream
  cleanly — `for s.Next() {}` returns after `run.done` exactly once; it does not
  reconnect.

### ⚠️ Behavior change — relay topologies must forward `Last-Event-ID`

In a `browser → backend relay (this SDK) → asgard-core` topology, the relay
**must** propagate a downstream client's inbound `Last-Event-ID` into the SDK via
`MessageRequestOptions.LastEventID` / `ChannelStreamOptions.LastEventID`.
Otherwise the resume cursor is lost at the SDK boundary, asgard-core treats the
reconnect as a fresh send, and **the turn is dispatched twice (a duplicate
run)**. On the way out, forward `streamer.LastEventID()` as the SSE `id:` of each
event you relay downstream so the browser's next reconnect carries the right
cursor.

### Added

- `MessageRequestOptions.LastEventID` (`pkg/client/options.go`) — when set,
  `NewStreamer` sends it as the `Last-Event-ID` request header and asgard-core
  **resubscribes from that cursor without re-dispatching** the message (the body's
  `CustomChannelId` is still used). Empty = a fresh send. Primarily for relays
  forwarding a downstream reconnect.
- `BotProviderClient.NewChannelStreamer(ctx, customChannelID, opts) (BotProviderStreamer, error)`
  (`pkg/client/bot_provider.go`) — opens the new `GET /ns/{ns}/bot-provider/{name}/message/sse`
  rejoin: replays the channel's collapsed history then streams the in-flight turn
  until its terminal, **without dispatching a message**. Use it to (re)attach a
  viewer to a channel.
- `client.ChannelStreamOptions{ LastEventID, UserIdentityHint }`
  (`pkg/client/options.go`) — options for `NewChannelStreamer`.
- `BotProviderStreamer.LastEventID() string` — the SSE `id:` (durable transcript
  seq) of the event `Current()` last returned; the resume cursor for that event.
  Persist it, or forward it downstream when relaying.
- `models.SseEventTypeMessageThinkingStart` / `...ThinkingDelta` /
  `...ThinkingComplete` (`asgard.message.thinking.{start,delta,complete}`) and the
  matching `GenericBotSseEventFact.MessageThinkingStart` / `...Delta` /
  `...Complete` fields (`pkg/models`) — extended-thinking blocks now stream
  separately from the assistant message (reusing the message fact shape).
- `GenericBotSseEventFactToolCallStart.ToolUseId` / `.ParentToolUseId` and
  `GenericBotSseEventFactToolCallComplete.ToolUseId` / `.ParentToolUseId` /
  `.IsError` (`pkg/models/sse_event.go`) — CLI-driver correlation id, subagent
  nesting, and tool-result error flag. Additive, `omitempty`.
- `BufferedMessage.ParentToolUseId` (`pkg/models/message.go`) — nests subagent
  (Task) output under its parent tool_use. Additive, `omitempty`.

### Changed

- `BotProviderStreamer` interface gains `LastEventID() string`. Consumers are
  unaffected; only a custom mock implementation of the interface needs the extra
  method.
- SSE connect/response errors from `NewStreamer` / `NewChannelStreamer` are now
  `*client.APIError` for non-2xx responses (previously an opaque connection
  error).

### Notes

- Wire shapes mirror asgard-core's `internal/models/edgeserver.go` (the
  edgeserver serializes those structs directly). Everything here is additive on
  the wire — no fields were removed or renamed.
- The resume cursor is asgard-core's durable Postgres transcript `seq`. Ephemeral
  events (run.*, deltas, tool_call, sandbox, consent, usage) carry no `seq` of
  their own; per the SSE spec the SDK anchors the cursor on the last persisted
  message via `Event.LastEventID` carry-forward — you don't need every event to
  have an id.
- Empty-cursor POST edge case: if a POST drops after the send was accepted but
  before the first frame (`id: 0`) is read, the SDK does not auto-reconnect (that
  would risk a duplicate dispatch) — it surfaces the error; re-send with the same
  idempotent `CustomMessageId`.

## [v1.5.6] - 2026-06-04

### Added

- `BotProviderClient.DownloadCwdFile(ctx, customChannelID, relativePath) ([]byte, *models.CwdDownloadMeta, error)` (`pkg/client/bot_provider.go`) — downloads a file from a channel's `/work` working directory via the edgeserver `GET /ns/{namespace}/bot-provider/{name}/cwd/download` endpoint. Returns the file bytes plus `CwdDownloadMeta` (filename from `Content-Disposition`, mime from `Content-Type`).
- `models.CwdDownloadMeta` (`pkg/models/cwd.go`) — filename + mime metadata for the download response.

### Notes

- Pairs with the asgard-core endpoint that resolves the `cwd://<relative_path>` links the agent pushes via the `show_cwd_download_link` builtin tool. `relative_path` is relative to `/work` (no leading `/`, no `.`/`..`); the server enforces this and returns 400 on violation, 404 on a missing file.

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
