package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"asgard-ai.com/asgard-sdk-go/pkg/client"
	"asgard-ai.com/asgard-sdk-go/pkg/models"
	log "github.com/sirupsen/logrus"
)

var (
	// EdgeServer configuration
	edgeServerHost    = flag.String("host", getEnv("EDGE_SERVER_HOST", "http://localhost:8080"), "EdgeServer host URL")
	namespace         = flag.String("namespace", getEnv("NAMESPACE", "default"), "Namespace")
	botProviderName   = flag.String("bot", getEnv("BOT_PROVIDER_NAME", "default-bot"), "Bot provider name")
	botProviderApiKey = flag.String("apikey", getEnv("BOT_PROVIDER_API_KEY", ""), "Bot provider API key")

	// Message configuration
	customChannelId = flag.String("channel", "", "Custom channel ID (auto-generated if empty)")
	customMessageId = flag.String("message", "", "Custom message ID (auto-generated if empty)")
	text            = flag.String("text", "Hello, EdgeServer!", "Message text to send")
	action          = flag.String("action", "NONE", "PostBack action (NONE or RESET_CHANNEL)")

	// Logging configuration
	logLevel = flag.String("log-level", getEnv("LOG_LEVEL", "info"), "Log level (debug, info, warn, error)")
	verbose  = flag.Bool("verbose", false, "Enable verbose output (shows all event details)")
)

func main() {
	flag.Parse()

	// Setup logging
	setupLogging()

	// Validate configuration
	if *botProviderApiKey == "" {
		log.Fatal("Bot provider API key is required (use -apikey or BOT_PROVIDER_API_KEY env var)")
	}

	// Print configuration
	log.Info("EdgeServer CLI Test Tool")
	log.Info("========================")
	log.Infof("Host:         %s", *edgeServerHost)
	log.Infof("Namespace:    %s", *namespace)
	log.Infof("Bot Provider: %s", *botProviderName)
	log.Infof("Message Text: %s", *text)
	log.Infof("Action:       %s", *action)
	log.Info("========================")

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupts
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Warn("Received interrupt signal, shutting down...")
		cancel()
	}()

	// Create client configuration
	config := &client.BotProviderConfig{
		EdgeServerHost:    *edgeServerHost,
		Namespace:         *namespace,
		BotProviderName:   *botProviderName,
		BotProviderApiKey: *botProviderApiKey,
	}

	// Generate IDs if not provided
	channelId := *customChannelId
	if channelId == "" {
		channelId = fmt.Sprintf("cli-channel-%d", time.Now().Unix())
	}

	messageId := *customMessageId
	if messageId == "" {
		messageId = fmt.Sprintf("cli-message-%d", time.Now().Unix())
	}

	// Create message
	var postBackAction models.PostBackAction
	switch *action {
	case "RESET_CHANNEL":
		postBackAction = models.PostBackActionResetChanel
	default:
		postBackAction = models.PostBackActionNone
	}

	message := &models.GenericBotMessage{
		CustomChannelId: channelId,
		CustomMessageId: messageId,
		Text:            *text,
		Action:          postBackAction,
	}

	log.Infof("Channel ID: %s", channelId)
	log.Infof("Message ID: %s", messageId)
	log.Info("")

	// Create SSE stream
	log.Info("Connecting to EdgeServer...")
	stream, err := client.NewStreaming(ctx, config, message)
	if err != nil {
		log.Fatalf("Failed to create SSE stream: %v", err)
	}
	defer stream.Close()

	log.Info("✓ Connected! Waiting for events...")
	log.Info("")

	// Process events
	eventCount := 0
	messageCount := 0
	startTime := time.Now()

	for stream.Next() {
		event := stream.Current()
		eventCount++

		// Always show event type
		log.WithFields(log.Fields{
			"event_type": event.EventType,
			"event_id":   event.EventId,
			"request_id": event.RequestId,
		}).Info("Received SSE event")

		// Handle different event types
		switch event.EventType {
		case models.SseEventTypeRunInit:
			log.Info("🚀 Run initialized")

		case models.SseEventTypeProcessStart:
			if event.Fact.ProcessStart != nil {
				log.Infof("⚙️  Process started: %s", event.Fact.ProcessStart.ProcessId)
				if *verbose && event.Fact.ProcessStart.Task != nil {
					log.Debugf("   Task: %+v", event.Fact.ProcessStart.Task)
				}
			}

		case models.SseEventTypeProcessComplete:
			if event.Fact.ProcessComplete != nil {
				log.Infof("✓ Process completed: %s", event.Fact.ProcessComplete.ProcessId)
				if *verbose && event.Fact.ProcessComplete.TaskResult != nil {
					log.Debugf("   Result: %+v", event.Fact.ProcessComplete.TaskResult)
				}
			}

		case models.SseEventTypeToolCallStart:
			if event.Fact.ToolCallStart != nil {
				tc := event.Fact.ToolCallStart.ToolCall
				log.Infof("🔧 Tool call started: %s.%s (seq: %d)",
					tc.ToolsetName, tc.ToolName, event.Fact.ToolCallStart.CallSeq)
				if *verbose {
					log.Debugf("   Parameters: %+v", tc.Parameter)
				}
			}

		case models.SseEventTypeToolCallComplete:
			if event.Fact.ToolCallComplete != nil {
				tc := event.Fact.ToolCallComplete.ToolCall
				log.Infof("✓ Tool call completed: %s.%s (seq: %d)",
					tc.ToolsetName, tc.ToolName, event.Fact.ToolCallComplete.CallSeq)
				if *verbose {
					log.Debugf("   Result: %+v", event.Fact.ToolCallComplete.ToolCallResult)
				}
			}

		case models.SseEventTypeMessageStart:
			messageCount++
			if event.Fact.MessageStart != nil {
				msg := event.Fact.MessageStart.Message
				log.Infof("💬 Message started: %s", msg.MessageId)
				if msg.Text != "" {
					log.Infof("   Text: %s", msg.Text)
				}
			}

		case models.SseEventTypeMessageDelta:
			if event.Fact.MessageDelta != nil {
				msg := event.Fact.MessageDelta.Message
				if msg.Text != "" {
					fmt.Print(msg.Text)
				}
			}

		case models.SseEventTypeMessageComplete:
			messageCount++
			if event.Fact.MessageComplete != nil {
				msg := event.Fact.MessageComplete.Message
				fmt.Println() // Newline after delta streaming
				log.Infof("✓ Message completed: %s", msg.MessageId)
				if msg.Text != "" && !*verbose {
					log.Infof("   Full text: %s", msg.Text)
				}
				if msg.Template != nil {
					log.Infof("   Template type: %s", msg.Template.Type)
					if *verbose {
						log.Debugf("   Template: %+v", msg.Template)
					}
				}
				if msg.IsDebug {
					log.Info("   [DEBUG MESSAGE]")
				}
			}

		case models.SseEventTypeRunDone:
			elapsed := time.Since(startTime)
			log.Info("🎉 Run completed successfully!")
			log.Infof("   Duration: %v", elapsed.Round(time.Millisecond))
			log.Infof("   Total events: %d", eventCount)
			log.Infof("   Messages received: %d", messageCount)

		case models.SseEventTypeRunError:
			if event.Fact.RunError != nil {
				log.Errorf("❌ Run error: %s", event.Fact.RunError.Error.Message)
				log.Errorf("   Code: %s", event.Fact.RunError.Error.Code)
				if event.Fact.RunError.Error.Inner != "" {
					log.Errorf("   Inner error: %s", event.Fact.RunError.Error.Inner)
				}
				log.Errorf("   Location: %s / %s / %s",
					event.Fact.RunError.Error.Location.Namespace,
					event.Fact.RunError.Error.Location.WorkflowName,
					event.Fact.RunError.Error.Location.ProcessorName)
			}

		default:
			log.Warnf("Unknown event type: %s", event.EventType)
		}

		// Show full event in verbose mode
		if *verbose {
			log.Debugf("Full event: %+v", event)
			log.Debug("")
		}
	}

	// Check for errors
	if err := stream.Err(); err != nil {
		elapsed := time.Since(startTime)
		log.Errorf("Stream error after %v: %v", elapsed.Round(time.Millisecond), err)
		log.Infof("Events received before error: %d", eventCount)
		os.Exit(1)
	}

	log.Info("")
	log.Info("Stream closed normally")
}

func setupLogging() {
	// Set log format
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "15:04:05.000",
		ForceColors:     true,
	})

	// Set log level
	level, err := log.ParseLevel(*logLevel)
	if err != nil {
		log.Warnf("Invalid log level '%s', using 'info'", *logLevel)
		level = log.InfoLevel
	}
	log.SetLevel(level)

	// Enable verbose if flag is set
	if *verbose {
		log.SetLevel(log.DebugLevel)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
