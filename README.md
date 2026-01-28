# asgard-sdk-go

A Go SDK for connecting to the Asgard EdgeServer, providing a client library for streaming bot provider events using Server-Sent Events (SSE).

## Features

- **Real-time Streaming**: Connect to Asgard EdgeServer using Server-Sent Events (SSE) for real-time bot interactions
- **Event-driven Architecture**: Handle various event types including run lifecycle, process execution, messages, and tool calls
- **Rich Message Templates**: Support for multiple message template types (text, images, videos, buttons, carousels, charts, tables)
- **Type-safe Models**: Comprehensive Go structs for all message and event types
- **Error Handling**: Detailed error information with location context
- **CLI Tool**: Command-line interface for testing and interacting with the EdgeServer

## Installation

```bash
go get go.asgard-ai.com/asgard-sdk-go
```

## Quick Start

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
    // Create client configuration
    config := &client.BotProviderConfig{
        EdgeServerHost:    "http://localhost:8080",
        Namespace:         "default",
        BotProviderName:   "my-bot",
        BotProviderApiKey: "your-api-key",
    }

    // Create message
    message := &models.GenericBotMessage{
        CustomChannelId: "channel-123",
        CustomMessageId: "message-456",
        Text:            "Hello, Asgard!",
        Action:          models.PostBackActionNone,
    }

    // Create SSE stream
    ctx := context.Background()
    stream, err := client.NewStreaming(ctx, config, message)
    if err != nil {
        log.Fatalf("Failed to create stream: %v", err)
    }
    defer stream.Close()

    // Process events
    for stream.Next() {
        event := stream.Current()

        switch event.EventType {
        case models.SseEventTypeMessageDelta:
            // Handle streaming message content
            if event.Fact.MessageDelta != nil {
                fmt.Print(event.Fact.MessageDelta.Message.Text)
            }

        case models.SseEventTypeMessageComplete:
            // Handle completed message
            if event.Fact.MessageComplete != nil {
                msg := event.Fact.MessageComplete.Message
                fmt.Printf("\nMessage completed: %s\n", msg.MessageId)
            }

        case models.SseEventTypeRunDone:
            fmt.Println("Run completed successfully!")
        }
    }

    // Check for errors
    if err := stream.Err(); err != nil {
        log.Fatalf("Stream error: %v", err)
    }
}
```

## Usage

### Creating a Client

The SDK uses a configuration-based approach:

```go
config := &client.BotProviderConfig{
    HTTPClient:        nil, // Optional: provide custom http.Client
    EdgeServerHost:    "http://localhost:8080",
    Namespace:         "default",
    BotProviderName:   "my-bot",
    BotProviderApiKey: "your-api-key",
}
```

### Sending Messages

```go
message := &models.GenericBotMessage{
    CustomChannelId: "unique-channel-id",
    CustomMessageId: "unique-message-id",
    Text:            "Your message text",
    Action:          models.PostBackActionNone, // or PostBackActionResetChanel
    BlobIds:         []string{},                // Optional: file attachments
    Payload:         map[string]interface{}{},  // Optional: custom payload
}
```

### Handling Events

The SDK provides various event types:

```go
for stream.Next() {
    event := stream.Current()

    switch event.EventType {
    case models.SseEventTypeRunInit:
        // Run initialization

    case models.SseEventTypeProcessStart:
        // Process started
        processId := event.Fact.ProcessStart.ProcessId

    case models.SseEventTypeProcessComplete:
        // Process completed
        result := event.Fact.ProcessComplete.TaskResult

    case models.SseEventTypeToolCallStart:
        // Tool call started
        toolCall := event.Fact.ToolCallStart.ToolCall

    case models.SseEventTypeToolCallComplete:
        // Tool call completed
        result := event.Fact.ToolCallComplete.ToolCallResult

    case models.SseEventTypeMessageStart:
        // Message streaming started

    case models.SseEventTypeMessageDelta:
        // Incremental message content
        text := event.Fact.MessageDelta.Message.Text

    case models.SseEventTypeMessageComplete:
        // Message completed
        msg := event.Fact.MessageComplete.Message

    case models.SseEventTypeRunDone:
        // Run completed successfully

    case models.SseEventTypeRunError:
        // Run encountered an error
        errDetail := event.Fact.RunError.Error
    }
}
```

### Working with Message Templates

The SDK supports various rich message templates:

```go
// Button template
template := &models.MessageTemplate{
    Type: models.MessageTemplateTypeButton,
    Text: stringPtr("Choose an option:"),
    Buttons: &[]models.MessageTemplateButton{
        {
            Label: "Option 1",
            Action: models.MessageTemplateAction{
                Type: models.MessageTemplateActionTypeMessage,
                Text: stringPtr("You selected Option 1"),
            },
        },
    },
}

// Carousel template
template := &models.MessageTemplate{
    Type: models.MessageTemplateTypeCarousel,
    Columns: &[]models.MessageTemplateColumn{
        {
            Title: "Item 1",
            Text:  "Description",
            ThumbnailImageUrl: stringPtr("https://example.com/image.jpg"),
            Buttons: []models.MessageTemplateButton{
                {
                    Label: "View",
                    Action: models.MessageTemplateAction{
                        Type: models.MessageTemplateActionTypeUri,
                        Uri:  stringPtr("https://example.com"),
                    },
                },
            },
        },
    },
}

// Table template
template := &models.MessageTemplate{
    Type: models.MessageTemplateTypeTable,
    Table: &models.MessageTemplateTable{
        RowType: models.MessageTemplateRowTypeObject,
        Columns: []models.MessageTemplateTableColumn{
            {Header: "Name", Key: "name"},
            {Header: "Age", Key: "age"},
        },
        Data: []interface{}{
            map[string]interface{}{"name": "Alice", "age": 30},
            map[string]interface{}{"name": "Bob", "age": 25},
        },
    },
}
```

### Error Handling

```go
stream, err := client.NewStreaming(ctx, config, message)
if err != nil {
    log.Fatalf("Failed to create stream: %v", err)
}
defer stream.Close()

for stream.Next() {
    event := stream.Current()

    // Check for run errors
    if event.EventType == models.SseEventTypeRunError {
        errDetail := event.Fact.RunError.Error
        log.Printf("Error: %s (code: %s)", errDetail.Message, errDetail.Code)
        log.Printf("Location: %s/%s/%s",
            errDetail.Location.Namespace,
            errDetail.Location.WorkflowName,
            errDetail.Location.ProcessorName)
    }
}

// Check for stream errors
if err := stream.Err(); err != nil {
    log.Fatalf("Stream error: %v", err)
}
```

## CLI Tool

The SDK includes a command-line tool for testing EdgeServer connections.

### Installation

```bash
cd cmd/edgeserver-cli
go build -o edgeserver-cli
```

### Usage

```bash
# Basic usage
./edgeserver-cli \
  -host http://localhost:8080 \
  -namespace default \
  -bot my-bot \
  -apikey your-api-key \
  -text "Hello, EdgeServer!"

# With environment variables
export EDGE_SERVER_HOST=http://localhost:8080
export NAMESPACE=default
export BOT_PROVIDER_NAME=my-bot
export BOT_PROVIDER_API_KEY=your-api-key

./edgeserver-cli -text "Hello!"

# Verbose mode
./edgeserver-cli -verbose -text "Debug this"

# Custom log level
./edgeserver-cli -log-level debug -text "Testing"

# Reset channel
./edgeserver-cli -action RESET_CHANNEL -text "Start fresh"
```

### CLI Options

- `-host`: EdgeServer host URL (env: `EDGE_SERVER_HOST`)
- `-namespace`: Namespace (env: `NAMESPACE`)
- `-bot`: Bot provider name (env: `BOT_PROVIDER_NAME`)
- `-apikey`: Bot provider API key (env: `BOT_PROVIDER_API_KEY`)
- `-channel`: Custom channel ID (auto-generated if empty)
- `-message`: Custom message ID (auto-generated if empty)
- `-text`: Message text to send
- `-action`: PostBack action (`NONE` or `RESET_CHANNEL`)
- `-log-level`: Log level (`debug`, `info`, `warn`, `error`)
- `-verbose`: Enable verbose output

## API Reference

### Client Package

#### `BotProviderConfig`
Configuration for connecting to the EdgeServer.

```go
type BotProviderConfig struct {
    HTTPClient        *http.Client
    EdgeServerHost    string
    Namespace         string
    BotProviderName   string
    BotProviderApiKey string
}
```

#### `NewStreaming`
Creates a new bot provider stream and establishes the SSE connection.

```go
func NewStreaming(
    ctx context.Context,
    config *BotProviderConfig,
    message *models.GenericBotMessage,
) (BotProviderStreamer, error)
```

#### `BotProviderStreamer` Interface
Interface for streaming bot provider events.

```go
type BotProviderStreamer interface {
    Next() bool                           // Advance to next event
    Current() *models.GenericBotSseEvent  // Get current event
    Err() error                           // Get stream error
    Close() error                         // Close stream
}
```

### Models Package

#### Message Types
- `GenericBotMessage`: Message sent from client to EdgeServer
- `BufferedMessage`: Message returned from EdgeServer
- `MessageTemplate`: Rich message template structures

#### Event Types
- `GenericBotSseEvent`: Server-Sent Event from EdgeServer
- Event type constants: `SseEventTypeRunInit`, `SseEventTypeRunDone`, `SseEventTypeRunError`, etc.

#### Template Types
- `MessageTemplateTypeText`
- `MessageTemplateTypeImage`
- `MessageTemplateTypeVideo`
- `MessageTemplateTypeAudio`
- `MessageTemplateTypeLocation`
- `MessageTemplateTypeButton`
- `MessageTemplateTypeCarousel`
- `MessageTemplateTypeChart`
- `MessageTemplateTypeTable`

## Dependencies

- [github.com/sirupsen/logrus](https://github.com/sirupsen/logrus) - Structured logging
- [github.com/tmaxmax/go-sse](https://github.com/tmaxmax/go-sse) - Server-Sent Events client

## Requirements

- Go 1.22.2 or higher
