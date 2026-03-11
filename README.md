# asgard-sdk-go

A lightweight Go SDK for Asgard EdgeServer, built around two agent roles.

## 1) Agent Concept

The SDK splits capabilities into two agent types:

- **BotAgent**
  - For conversational workflows
  - Includes: `SSE streaming`, `message`, `blob upload`
- **FunctionAgent**
  - For trigger workflows
  - Includes: `json trigger`, `form trigger`

This split matches product behavior:
- Bot = multi-turn conversation
- Function = one-shot execution

## 2) Start from Agent

Install:

```bash
go get go.asgard-ai.com/asgard-sdk-go
```

Create agents:

```go
import "go.asgard-ai.com/asgard-sdk-go/pkg/client"

botAgent := client.NewBotAgent(
    "http://localhost:8080",
    "default",
    "my-bot",
    "your-api-key",
)

functionAgent := client.NewFunctionAgent(
    "http://localhost:8080",
    "default",
    "my-bot",
    "your-api-key",
)
```

## 3) Streamer Example (BotAgent)

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
    agent := client.NewBotAgent(
        "http://localhost:8080",
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

    stream, err := agent.NewStreamer(context.Background(), msg)
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
        case models.SseEventTypeRunDone:
            fmt.Println("\nDone")
        }
    }

    if err := stream.Err(); err != nil {
        log.Fatal(err)
    }
}
```

## 4) CLI Usage

Build CLI:

```bash
cd cmd/edgeserver-cli
go build -o edgeserver-cli
```

### Bot agent (interactive)

```bash
./edgeserver-cli \
  -host http://localhost:8080 \
  -namespace default \
  -bot my-bot \
  -apikey your-api-key \
  -agent bot
```

Bot REPL commands:

- `/help`
- `/transport sse|rest`
- `/debug on|off`
- `/blob <path> [mime]`
- `/blobs`
- `/clear-blobs`
- `/channel [id]`
- `/reset [text]`
- `/exit`

### Function agent (one-shot)

JSON trigger:

```bash
./edgeserver-cli \
  -host http://localhost:8080 \
  -namespace default \
  -bot my-bot \
  -apikey your-api-key \
  -agent function \
  -json-trigger \
  -trigger-payload '{"event":"ping"}'
```

Form trigger:

```bash
./edgeserver-cli \
  -host http://localhost:8080 \
  -namespace default \
  -bot my-bot \
  -apikey your-api-key \
  -agent function \
  -form-trigger \
  -trigger-payload-file ./payload.json \
  -form-file ./invoice.pdf \
  -form-mime application/pdf
```
