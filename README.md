# asgard-sdk-go

A Go SDK for Asgard EdgeServer.

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

### ListDirectory

```go
result, err := ss.ListDirectory(ctx, "/data", nil, nil)
if err != nil {
    log.Fatal(err)
}
for _, entry := range result.Entries {
    fmt.Printf("%s  dir=%v  size=%d\n", entry.Name, entry.IsDir, entry.SizeBytes)
}
```

### Stat

```go
info, err := ss.Stat(ctx, "/data/report.csv")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("exists=%v size=%d\n", info.Exists, info.SizeBytes)
```

### ReadFile

```go
data, err := ss.ReadFile(ctx, "/data/report.csv", nil, nil)
if err != nil {
    log.Fatal(err)
}
fmt.Println(string(data))
```

Read a slice with optional offset and limit (bytes):

```go
offset := int64(1024)
limit  := int64(4096)
data, err := ss.ReadFile(ctx, "/data/report.csv", &offset, &limit)
```

### WriteFile

```go
f, _ := os.Open("report.csv")
defer f.Close()

result, err := ss.WriteFile(ctx, "/data/report.csv", f, "report.csv")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("wrote %d bytes\n", result.BytesWritten)
```

### MakeDirectory

```go
if err := ss.MakeDirectory(ctx, "/data/2026/reports"); err != nil {
    log.Fatal(err)
}
```

### Remove / RemoveAll

```go
// Remove a single file or empty directory
if err := ss.Remove(ctx, "/data/old.csv"); err != nil {
    log.Fatal(err)
}

// Recursively delete a directory and all its contents
if err := ss.RemoveAll(ctx, "/data/archive"); err != nil {
    log.Fatal(err)
}
```

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
    EdgeServerHost:  "http://localhost:8080",
    Namespace:       "default",
    SourceSetName:   "my-sourceset",
    SourceSetApiKey: "your-api-key",
})
```
