# asgard-sdk-go

A Go SDK for Asgard EdgeServer.

## Table of Contents

- [Installation](#installation)
- [BotProviderClient](#botproviderclient)
- [Streaming (SSE)](#streaming-sse)
  - [Transparent resume](#transparent-resume)
  - [Leaving a stream early](#leaving-a-stream-early)
  - [Rejoining a channel](#rejoining-a-channel)
  - [Relaying SSE to a browser](#relaying-sse-to-a-browser)
- [SendMessage (REST)](#sendmessage-rest)
- [UploadBlob](#uploadblob)
- [DeleteChannel](#deletechannel)
- [TriggerJSON](#triggerjson)
- [TriggerForm](#triggerform)
- [SourceSetClient](#sourcesetclient)
  - [Copy / Move](#copy--move)
  - [ListDirectory](#listdirectory)
  - [Stat](#stat)
  - [ReadFile](#readfile)
  - [WriteFile](#writefile)
  - [MakeDirectory](#makedirectory)
  - [Remove / RemoveAll](#remove--removeall)
- [Custom HTTP client and headers](#custom-http-client-and-headers)
- [Error handling](#error-handling)

## Installation

```bash
go get go.asgard-ai.com/asgard-sdk-go
```

## BotProviderClient

`BotProviderClient` is the single interface for all bot-provider APIs: streaming, messaging, blob upload, function triggers, and sandbox operations.

```go
import "go.asgard-ai.com/asgard-sdk-go/pkg/client"

c := client.NewBotProviderClient(
    "https://api.asgard-ai.com",
    "default",       // namespace
    "my-bot",        // bot provider name
    "your-api-key",
)
```

## Streaming (SSE)

The most common usage — stream bot responses event by event:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "go.asgard-ai.com/asgard-sdk-go/pkg/client"
    "go.asgard-ai.com/asgard-sdk-go/pkg/models"
)

func main() {
    c := client.NewBotProviderClient(
        "https://api.asgard-ai.com",
        "default",
        "my-bot",
        "your-api-key",
    )

    msg := &models.GenericBotMessage{
        CustomChannelId: "channel-1",
        CustomMessageId: "msg-1",
        Text:            "Hello",
        Action:          models.PostBackActionNone,
    }

    stream, err := c.NewStreamer(context.Background(), msg, nil)
    if err != nil {
        log.Fatal(err)
    }
    defer stream.Close()

    for stream.Next() {
        event := stream.Current()
        switch event.EventType {
        case models.SseEventTypeMessageDelta:
            if event.Fact.MessageDelta != nil {
                fmt.Print(event.Fact.MessageDelta.Message.Text)
            }
        case models.SseEventTypeMessageComplete:
            fmt.Println()
        case models.SseEventTypeRunError:
            if event.Fact.RunError != nil {
                log.Printf("run error: %s", event.Fact.RunError.Error.Message)
            }
        }
    }

    if err := stream.Err(); err != nil {
        log.Fatal(err)
    }
}
```

Pass `MessageRequestOptions` to enable debug mode, set a user identity hint,
or bypass the tool-call consent gate for this single request:

```go
opts := &client.MessageRequestOptions{
    IsDebug:               true,
    UserIdentityHint:      "user-123",
    BypassToolCallConsent: true,
}
stream, err := c.NewStreamer(ctx, msg, opts)
```

`BypassToolCallConsent: true` makes the SSE run auto-approve every tool call
without emitting an `asgard.tool_call.consent` event. The server's persistent
`tool_call_allow_list` is not modified.

### Transparent resume

The run executes in the background on the server, independent of your SSE
connection. If an **established** stream drops (network blip, gateway idle
timeout, brief outage), the streamer reconnects and resubscribes from the last
`Last-Event-ID` cursor automatically — `Next()` just keeps returning events, so
your `for stream.Next()` loop is unaffected. You don't have to do anything.

Reconnection only ever resumes a stream that reached `200`. A non-2xx response
(including a transient `5xx`/`429` during a deploy) is surfaced as a
`*client.APIError` and **not** retried — inspect it via `errors.As` and re-open
if you want to. A turn's terminal (`asgard.run.done` / `asgard.run.error`) ends
the stream cleanly.

The resume loop stops — no further reconnect — on any of: a turn terminal
(`asgard.run.done` / `asgard.run.error`), `Close()` or a cancelled `context`
(see [Leaving a stream early](#leaving-a-stream-early)), or a non-2xx response.
It does **not** inspect the channel's server-side run state
(`ChannelMetadata.RunState`): a turn cancelled on the server is recognized only
when the server delivers that cancellation as one of the terminals above.
Cancellation signalled any other way is currently treated as a transient drop
and keeps resuming — so if you no longer care about the turn, stop the stream
yourself with `Close()`.

### Leaving a stream early

You can stop consuming a stream at any time — you are never obligated to drain it
to its terminal. Call `stream.Close()` (or cancel the `context` you passed in).
Both are idempotent and safe from any goroutine: they stop the transparent-resume
loop, release the connection, and immediately unblock a blocked `Next()`, so your
`for stream.Next()` loop exits at once. After a plain `Close()`, `stream.Err()` is
`nil`; after a `context` cancel, it reflects the `context` error.

```go
stream, err := c.NewStreamer(ctx, msg, nil)
if err != nil {
    log.Fatal(err)
}
defer stream.Close() // idempotent — safe even if you also Close() below

for stream.Next() {
    ev := stream.Current()
    if noLongerInterested(ev) {
        stream.Close() // stop receiving; the loop exits on the next Next()
        break
    }
    // ... handle ev ...
}
```

**Leaving only detaches *your client*.** The run executes in the background on
asgard-core independently of your SSE connection, so for `POST /message/sse`
(`NewStreamer`) the message has already been dispatched and **the turn keeps
running to completion on the server** — `Close()` does not cancel it. You can
re-attach later with [`NewChannelStreamer`](#rejoining-a-channel) and replay
whatever you missed. To actually stop the work, use
[`SuspendChannel`](#stopping-a-run).

### Stopping a run

`SuspendChannel` stops a channel's in-flight run. This is what a "stop" button
should call — closing a streamer only stops watching, while the agent keeps
working in the background.

```go
if err := c.SuspendChannel(ctx, "channel-1", nil); err != nil {
    log.Fatal(err)
}
// Keep reading the stream you already have: the run stops asynchronously and
// announces it with the same terminal event it would emit on finishing.
```

The call returns as soon as the request is registered, **not** when the run has
stopped. Do not treat its return as "the agent is idle" — wait for your
stream's terminal event instead. There is no new event type to handle.

The conversation is preserved: the transcript survives and the next message
continues it. Because a stopped run never reaches its end, the turn is rolled
back — the channel's context is left as the turn found it.

Stopping a run that already finished, or that a newer turn superseded, succeeds
and does nothing. Pass `&client.SuspendOptions{RequestID: id}` to pin the
suspend to one specific run so a caller cannot accidentally stop a newer one.
`Force: true` abandons the run immediately instead of letting it stop
gracefully — reach for it only when a graceful suspend has not taken effect,
since in-flight work is dropped rather than allowed to wind down.

### Rejoining a channel

`NewChannelStreamer` opens the `GET /message/sse` rejoin: it replays a channel's
collapsed history and then streams the in-flight turn until its terminal —
**without** dispatching a new message. Use it to (re)attach a viewer to a channel
after a page reload or a process restart.

```go
// Resume from a cursor you persisted earlier (empty = replay full history).
stream, err := c.NewChannelStreamer(ctx, "channel-1", &client.ChannelStreamOptions{
    LastEventID: savedCursor, // e.g. the last stream.LastEventID() you stored
})
if err != nil {
    log.Fatal(err)
}
defer stream.Close()

for stream.Next() {
    ev := stream.Current()
    // ... render ev ...
    savedCursor = stream.LastEventID() // persist the resume cursor as you go
}
if err := stream.Err(); err != nil {
    log.Fatal(err)
}
```

`stream.LastEventID()` returns the SSE `id:` (a durable transcript seq) of the
event `Current()` last returned — persist it to resume later.

### Relaying SSE to a browser

A common topology is `browser → your backend relay (this SDK) → asgard-core`.
For resume to work end-to-end across the relay, the cursor must flow **both
ways**:

- **Inbound**: forward the browser's `Last-Event-ID` request header into the SDK
  (`MessageRequestOptions.LastEventID` for `NewStreamer`, or
  `ChannelStreamOptions.LastEventID` for `NewChannelStreamer`). If you drop it,
  asgard-core sees a fresh send and **dispatches the turn twice**.
- **Outbound**: stamp each SSE event you write downstream with
  `stream.LastEventID()` as its `id:`, so the browser's next reconnect carries the
  right cursor.

```go
func handleBrowserSSE(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/event-stream")
    flusher := w.(http.Flusher)

    // Inbound: propagate the browser's resume cursor (empty on first connect).
    stream, err := c.NewChannelStreamer(r.Context(), r.URL.Query().Get("channelId"),
        &client.ChannelStreamOptions{LastEventID: r.Header.Get("Last-Event-ID")})
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadGateway)
        return
    }
    defer stream.Close()

    for stream.Next() {
        ev := stream.Current()
        data, _ := json.Marshal(ev)
        // Outbound: re-stamp the same id: the browser will echo back on reconnect.
        fmt.Fprintf(w, "id: %s\nevent: %s\ndata: %s\n\n", stream.LastEventID(), ev.EventType, data)
        flusher.Flush()
    }
}
```

This composes cleanly because the SDK's reconnect policy mirrors the browser's
native `EventSource` (resume a dropped `200`, stop on a non-2xx). Note that the
browser can't tell a clean end from a drop at the transport layer, so the front
end should treat `asgard.run.done` / `asgard.run.error` as the signal to
`eventSource.close()` rather than letting it reconnect.

## SendMessage (REST)

Synchronous message — waits for the full reply:

```go
reply, err := c.SendMessage(ctx, msg, nil)
if err != nil {
    log.Fatal(err)
}
for _, m := range reply.Messages {
    fmt.Println(m.Text)
}
```

Pass `MessageRequestOptions` to enable debug mode or set a user identity hint:

```go
opts := &client.MessageRequestOptions{
    IsDebug:          true,
    UserIdentityHint: "user-123",
}
reply, err := c.SendMessage(ctx, msg, opts)
```

The `BypassToolCallConsent` option is currently only honored by the SSE
endpoint (`NewStreamer`); the REST `SendMessage` endpoint ignores it
server-side.

## UploadBlob

Upload a file to attach to subsequent messages via `BlobIds`:

```go
f, _ := os.Open("invoice.pdf")
defer f.Close()

mime := "application/pdf"
blob, err := c.UploadBlob(ctx, "channel-1", f, "invoice.pdf", &mime)
if err != nil {
    log.Fatal(err)
}

msg := &models.GenericBotMessage{
    CustomChannelId: "channel-1",
    CustomMessageId: "msg-2",
    Text:            "Please process this invoice",
    BlobIds:         []string{blob.BlobId},
}
```

## DeleteChannel

`DeleteChannel` ends a conversation. It releases everything the channel holds —
the in-flight run, the transcript, every uploaded blob, the tool-call
allow-list, the Sandbox and its Channel Home — and removes the channel itself,
so the same `customChannelId` is free for a fresh conversation:

```go
if err := c.DeleteChannel(ctx, "channel-1"); err != nil {
    log.Fatal(err)
}
// "channel-1" is now unknown to the platform; the next UploadBlob or message
// on it starts a brand-new conversation with a new session.
```

Deleting a channel that does not exist (never created, or already deleted)
succeeds and does nothing. Unlike `SuspendChannel`, the call returns once the
teardown has completed — a channel backing a live Sandbox waits for the pod to
terminate — so you may start a new turn on the same id straight away.

This is also how to start over **with attachments**. `Action: RESET_CHANNEL`
wipes the channel *before* dispatching its message, which destroys any blob the
message names, so the server rejects a reset that carries `BlobIds` (400).
Delete first, then upload, then send with `Action: NONE`:

```go
_ = c.DeleteChannel(ctx, "channel-1")
blob, _ := c.UploadBlob(ctx, "channel-1", f, "invoice.pdf", &mime)
_, err := c.SendMessage(ctx, &models.GenericBotMessage{
    CustomChannelId: "channel-1",
    CustomMessageId: "msg-1",
    Text:            "Please process this invoice",
    Action:          models.PostBackActionNone,
    BlobIds:         []string{blob.BlobId},
}, nil)
```

## TriggerJSON

One-shot JSON trigger — no conversation state:

```go
result, err := c.TriggerJSON(ctx, map[string]interface{}{
    "event": "order.created",
    "orderId": "ORD-001",
})
if err != nil {
    log.Fatal(err)
}
fmt.Printf("%+v\n", result)
```

## TriggerForm

Form trigger with an optional file attachment:

```go
payload := map[string]interface{}{"type": "invoice"}

f, _ := os.Open("invoice.pdf")
defer f.Close()

mime := "application/pdf"
result, err := c.TriggerForm(ctx, payload, f, "invoice.pdf", &mime)
```

To trigger without a file, pass `nil` for reader, filename, and mime:

```go
result, err := c.TriggerForm(ctx, payload, nil, "", nil)
```

## SourceSetClient

`SourceSetClient` is the interface for SourceSet volume operations.

```go
ss := client.NewSourceSetClient(
    "https://api.asgard-ai.com",
    "default",        // namespace
    "my-sourceset",   // source set name
    "your-api-key",
)
```

Every path is **relative to the volume root**: no leading `/`, no `.` or `..`
component, no consecutive or trailing slashes. The empty string means the root
(accepted by `ListDirectory` only). The server enforces this, so a path outside
the contract comes back as an `*APIError` with status 400.

### ListDirectory

Results are paginated; `Paging.Total` is the full entry count of the directory,
so a caller can page through a large one.

```go
result, err := ss.ListDirectory(ctx, "data", nil, nil)
if err != nil {
    log.Fatal(err)
}
for _, entry := range result.Entries {
    fmt.Printf("%s  dir=%v  size=%d  mode=%o\n",
        entry.Name, entry.IsDir, entry.SizeBytes, entry.Mode)
}
```

### Stat

```go
info, err := ss.Stat(ctx, "data/report.csv")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("exists=%v size=%d mode=%o\n", info.Exists, info.SizeBytes, info.Mode)
```

A missing path is not an error — it returns `Exists: false`.

### ReadFile

`meta.TotalBytes` is the file's full size regardless of any range, and
`meta.Truncated` reports that content remains past what was returned.

```go
data, meta, err := ss.ReadFile(ctx, "data/report.csv", nil, nil)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("%d of %d bytes (truncated=%v)\n", len(data), meta.TotalBytes, meta.Truncated)
```

Read a slice with optional offset and limit (bytes):

```go
offset := int64(1024)
limit := int64(4096)
data, meta, err := ss.ReadFile(ctx, "data/report.csv", &offset, &limit)
```

### WriteFile

`mode` is the Unix permission bits (`nil` for the server default 0644).
`createOnly` fails with a 409 `*APIError` instead of truncating a file that is
already there — use it for "create new", and `false` for "save".

```go
f, _ := os.Open("report.csv")
defer f.Close()

result, err := ss.WriteFile(ctx, "data/report.csv", f, "report.csv", nil, false)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("wrote %d bytes\n", result.BytesWritten)
```

### MakeDirectory

```go
if err := ss.MakeDirectory(ctx, "data/2026/reports"); err != nil {
    log.Fatal(err)
}
```

### Copy / Move

`Copy` recurses when the source is a directory. `Move` doubles as rename — a move
within the same parent directory. Without `overwrite`, an existing destination is
a 409 `*APIError` rather than a silent replacement; with it, the destination is
replaced wholesale rather than merged.

```go
res, err := ss.Copy(ctx, "data/report.csv", "archive/report.csv", false)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("copied %d bytes\n", res.BytesCopied)

// Rename in place.
if err := ss.Move(ctx, "data/report.csv", "data/report-2026.csv", false); err != nil {
    log.Fatal(err)
}
```

### Remove / RemoveAll

```go
// Remove a single file or empty directory
if err := ss.Remove(ctx, "data/old.csv"); err != nil {
    log.Fatal(err)
}

// Recursively delete a directory and all its contents
if err := ss.RemoveAll(ctx, "data/archive"); err != nil {
    log.Fatal(err)
}
```

`RemoveAll` refuses the volume root, so it can delete a subtree but never the
SourceSet's whole contents.

## Custom HTTP client and headers

Use `BotProviderConfig` or `SourceSetConfig` to provide a custom HTTP client or extra headers:

```go
import "net/http"
import "time"

c := client.NewBotProviderClientWithConfig(&client.BotProviderConfig{
    HTTPClient:        &http.Client{Timeout: 60 * time.Second},
    EdgeServerHost:    "https://api.asgard-ai.com",
    Namespace:         "default",
    BotProviderName:   "my-bot",
    BotProviderApiKey: "your-api-key",
    Headers: map[string]string{
        "X-Request-Source": "my-service",
    },
})

ss := client.NewSourceSetClientWithConfig(&client.SourceSetConfig{
    HTTPClient:      &http.Client{Timeout: 120 * time.Second},
    EdgeServerHost:  "https://api.asgard-ai.com",
    Namespace:       "default",
    SourceSetName:   "my-sourceset",
    SourceSetApiKey: "your-api-key",
})
```

## Error handling

Every SDK HTTP method returns `*client.APIError` (via `error`) when the server
responds with a non-2xx status or an `isSuccess=false` envelope. Inspect it
with `errors.As`, or use the `Is<Status>` helpers:

```go
import (
    "errors"
    "log"

    "go.asgard-ai.com/asgard-sdk-go/pkg/client"
)

result, err := bp.SandboxHeartbeat(ctx, "my-sandbox")
if err != nil {
    // Sandbox CR not found — re-launch via the message API before retrying.
    if client.IsPreconditionFailed(err) {
        log.Println("sandbox gone, relaunching")
        return relaunch()
    }
    // Sandbox does not belong to this bot provider — config bug.
    if client.IsBadRequest(err) {
        return err
    }
    // Generic: pull out status + server-supplied errorCode/message.
    var apiErr *client.APIError
    if errors.As(err, &apiErr) {
        log.Printf("api error: status=%d code=%s msg=%s",
            apiErr.StatusCode, apiErr.ErrorCode, apiErr.Message)
    }
    return err
}
_ = result
```

Network-level errors (DNS, dial, timeout) are still returned as plain wrapped
errors and will not satisfy `errors.As(&apiErr)`.
