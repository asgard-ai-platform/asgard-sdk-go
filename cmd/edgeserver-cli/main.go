package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"go.asgard-ai.com/asgard-sdk-go/pkg/client"
	"go.asgard-ai.com/asgard-sdk-go/pkg/models"
)

var (
	// Common
	edgeServerHost    = flag.String("host", getEnv("EDGE_SERVER_HOST", "http://localhost:8080"), "EdgeServer host URL")
	namespace         = flag.String("namespace", getEnv("NAMESPACE", "default"), "Namespace")
	botProviderName   = flag.String("bot", getEnv("BOT_PROVIDER_NAME", "default-bot"), "Bot provider name")
	botProviderApiKey = flag.String("apikey", getEnv("BOT_PROVIDER_API_KEY", ""), "Bot provider API key")
	agentType         = flag.String("agent", "bot", "Agent mode: bot or function")

	// Bot agent options
	channelID = flag.String("channel", "", "Conversation channel ID for bot agent (auto-generated if empty)")
	transport = flag.String("transport", "sse", "Initial bot transport: sse or rest")
	debug     = flag.Bool("debug", false, "Initial debug mode for bot REST /message")

	// Function agent options
	jsonTrigger        = flag.Bool("json-trigger", false, "Function agent: call /json trigger")
	formTrigger        = flag.Bool("form-trigger", false, "Function agent: call /form trigger")
	triggerPayload     = flag.String("trigger-payload", "", "Function agent: payload as JSON string")
	triggerPayloadFile = flag.String("trigger-payload-file", "", "Function agent: payload JSON file path")
	formFile           = flag.String("form-file", "", "Function agent: file path for /form trigger (optional)")
	formMime           = flag.String("form-mime", "", "Function agent: MIME type for /form file (optional)")

	// Logging
	logLevel = flag.String("log-level", getEnv("LOG_LEVEL", "info"), "Log level (debug, info, warn, error)")
	verbose  = flag.Bool("verbose", false, "Enable verbose output")
)

type botSession struct {
	channelID string
	transport string
	debug     bool
	blobIDs   []string
	seq       int
}

func main() {
	flag.Parse()
	setupLogging()

	if *botProviderApiKey == "" {
		log.Fatal("Bot provider API key is required (use -apikey or BOT_PROVIDER_API_KEY env var)")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Warn("Received interrupt signal, shutting down...")
		cancel()
	}()

	mode := strings.ToLower(strings.TrimSpace(*agentType))
	switch mode {
	case "bot":
		runBot(ctx)
	case "function":
		runFunction(ctx)
	default:
		log.Fatalf("Invalid -agent '%s' (supported: bot, function)", *agentType)
	}
}

func runBot(ctx context.Context) {
	a := client.NewBotAgent(*edgeServerHost, *namespace, *botProviderName, *botProviderApiKey)

	initialChannelID := strings.TrimSpace(*channelID)
	if initialChannelID == "" {
		initialChannelID = fmt.Sprintf("cli-channel-%d", time.Now().Unix())
	}

	initialTransport := strings.ToLower(strings.TrimSpace(*transport))
	if initialTransport != "sse" && initialTransport != "rest" {
		log.Fatalf("Invalid -transport '%s' (supported: sse, rest)", *transport)
	}

	session := &botSession{
		channelID: initialChannelID,
		transport: initialTransport,
		debug:     *debug,
		blobIDs:   []string{},
		seq:       0,
	}

	log.Info("BotAgent interactive mode")
	log.Infof("Host=%s Namespace=%s BotProvider=%s", *edgeServerHost, *namespace, *botProviderName)
	log.Infof("Channel=%s Transport=%s Debug=%v", session.channelID, session.transport, session.debug)
	printBotHelp()

	if err := runBotREPL(ctx, a, session); err != nil {
		log.Fatalf("Bot interactive mode failed: %v", err)
	}
}

func runFunction(ctx context.Context) {
	if *jsonTrigger == *formTrigger {
		log.Fatal("Function agent requires exactly one trigger mode: --json-trigger or --form-trigger")
	}

	a := client.NewFunctionAgent(*edgeServerHost, *namespace, *botProviderName, *botProviderApiKey)

	payload, err := parseTriggerPayload(*triggerPayload, *triggerPayloadFile)
	if err != nil {
		log.Fatalf("Invalid trigger payload: %v", err)
	}

	log.Info("FunctionAgent one-shot mode")
	log.Infof("Host=%s Namespace=%s BotProvider=%s", *edgeServerHost, *namespace, *botProviderName)

	if err := runFunctionOnce(ctx, a, payload); err != nil {
		log.Fatal(err)
	}
}

func runBotREPL(ctx context.Context, a client.BotAgent, session *botSession) error {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		fmt.Print("bot> ")
		if !scanner.Scan() {
			if scanErr := scanner.Err(); scanErr != nil {
				return fmt.Errorf("failed to read input: %w", scanErr)
			}
			log.Info("Input closed, exiting")
			return nil
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		if strings.HasPrefix(input, "/") {
			keepGoing, err := handleBotCommand(ctx, a, session, input)
			if err != nil {
				log.Errorf("Command failed: %v", err)
			}
			if !keepGoing {
				return nil
			}
			continue
		}

		if err := sendBotMessage(ctx, a, session, input, models.PostBackActionNone); err != nil {
			log.Errorf("Send failed: %v", err)
		}
	}
}

func handleBotCommand(ctx context.Context, a client.BotAgent, session *botSession, input string) (bool, error) {
	parts := strings.Fields(input)
	cmd := parts[0]

	switch cmd {
	case "/help":
		printBotHelp()
		return true, nil
	case "/exit", "/quit":
		log.Info("Bye")
		return false, nil
	case "/transport":
		if len(parts) != 2 || (parts[1] != "sse" && parts[1] != "rest") {
			return true, fmt.Errorf("usage: /transport sse|rest")
		}
		session.transport = parts[1]
		log.Infof("Transport -> %s", session.transport)
		return true, nil
	case "/debug":
		if len(parts) != 2 || (parts[1] != "on" && parts[1] != "off") {
			return true, fmt.Errorf("usage: /debug on|off")
		}
		session.debug = parts[1] == "on"
		log.Infof("Debug -> %v", session.debug)
		return true, nil
	case "/blob":
		if len(parts) < 2 {
			return true, fmt.Errorf("usage: /blob <path> [mime]")
		}
		mimeType := ""
		if len(parts) >= 3 {
			mimeType = parts[2]
		}
		blob, err := uploadBlob(ctx, a, session.channelID, parts[1], mimeType)
		if err != nil {
			return true, err
		}
		session.blobIDs = append(session.blobIDs, blob.BlobId)
		log.Infof("Blob attached: %s", blob.BlobId)
		return true, nil
	case "/blobs":
		if len(session.blobIDs) == 0 {
			log.Info("No attached blobs")
			return true, nil
		}
		log.Infof("Attached blobs: %s", strings.Join(session.blobIDs, ", "))
		return true, nil
	case "/clear-blobs":
		session.blobIDs = nil
		log.Info("Attached blobs cleared")
		return true, nil
	case "/channel":
		if len(parts) == 1 {
			log.Infof("Current channel: %s", session.channelID)
			return true, nil
		}
		session.channelID = parts[1]
		log.Infof("Channel -> %s", session.channelID)
		return true, nil
	case "/reset":
		msg := "reset"
		if len(parts) > 1 {
			msg = strings.TrimSpace(strings.TrimPrefix(input, "/reset"))
		}
		return true, sendBotMessage(ctx, a, session, msg, models.PostBackActionResetChanel)
	default:
		return true, fmt.Errorf("unknown command: %s (use /help)", cmd)
	}
}

func sendBotMessage(ctx context.Context, a client.BotAgent, session *botSession, text string, action models.PostBackAction) error {
	session.seq++
	messageID := fmt.Sprintf("cli-message-%d-%d", time.Now().Unix(), session.seq)

	msg := &models.GenericBotMessage{
		CustomChannelId: session.channelID,
		CustomMessageId: messageID,
		Text:            text,
		Action:          action,
		BlobIds:         append([]string{}, session.blobIDs...),
	}

	log.Debugf("[send] channel=%s message=%s transport=%s action=%s blobs=%d",
		session.channelID,
		messageID,
		session.transport,
		action,
		len(msg.BlobIds),
	)

	switch session.transport {
	case "rest":
		return sendByREST(ctx, a, msg, session.debug)
	case "sse":
		return sendBySSE(ctx, a, msg)
	default:
		return fmt.Errorf("unsupported transport: %s", session.transport)
	}
}

func sendByREST(ctx context.Context, a client.BotAgent, msg *models.GenericBotMessage, debug bool) error {
	start := time.Now()
	reply, err := a.SendMessage(ctx, msg, debug)
	if err != nil {
		return err
	}

	log.Debugf("[rest] done in %v, requestId=%s, messages=%d",
		time.Since(start).Round(time.Millisecond),
		reply.RequestId,
		len(reply.Messages),
	)

	for _, m := range reply.Messages {
		if m.Text != "" {
			fmt.Println(m.Text)
		}
		if m.Template != nil {
			log.Debugf("template=%s", m.Template.Type)
		}
		if *verbose {
			log.Debugf("message=%+v", m)
		}
	}

	if reply.ErrorDetail != nil {
		log.Warnf("error detail: %+v", *reply.ErrorDetail)
	}

	return nil
}

func sendBySSE(ctx context.Context, a client.BotAgent, msg *models.GenericBotMessage) error {
	stream, err := a.NewStreamer(ctx, msg)
	if err != nil {
		return err
	}
	defer stream.Close()

	for stream.Next() {
		e := stream.Current()
		if *verbose {
			log.Debugf("event=%+v", e)
		}

		switch e.EventType {
		case models.SseEventTypeMessageDelta:
			if e.Fact.MessageDelta != nil && e.Fact.MessageDelta.Message.Text != "" {
				fmt.Print(e.Fact.MessageDelta.Message.Text)
			}
		case models.SseEventTypeMessageComplete:
			fmt.Println()
			if e.Fact.MessageComplete != nil && e.Fact.MessageComplete.Message.Template != nil {
				log.Debugf("template=%s", e.Fact.MessageComplete.Message.Template.Type)
			}
		case models.SseEventTypeRunError:
			if e.Fact.RunError != nil {
				return fmt.Errorf("run error: %s", e.Fact.RunError.Error.Message)
			}
		}
	}

	if err := stream.Err(); err != nil {
		return err
	}
	return nil
}

func uploadBlob(ctx context.Context, a client.BotAgent, channelID, filePath, mimeType string) (*models.Blob, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	filename := filepath.Base(filePath)
	var mime *string
	if mimeType != "" {
		mime = &mimeType
	}

	return a.UploadBlob(ctx, channelID, file, filename, mime)
}

func runFunctionOnce(ctx context.Context, a client.FunctionAgent, payload map[string]interface{}) error {
	start := time.Now()

	var (
		result interface{}
		err    error
	)

	if *jsonTrigger {
		result, err = a.TriggerJSON(ctx, payload)
	} else {
		var (
			reader   io.Reader
			filename string
			mime     *string
			file     *os.File
		)
		if *formFile != "" {
			file, err = os.Open(*formFile)
			if err != nil {
				return fmt.Errorf("failed to open form file: %w", err)
			}
			defer file.Close()
			reader = file
			filename = filepath.Base(*formFile)
			if *formMime != "" {
				mime = formMime
			}
		}
		result, err = a.TriggerForm(ctx, payload, reader, filename, mime)
	}
	if err != nil {
		return fmt.Errorf("function api failed: %w", err)
	}

	log.Infof("Done in %v", time.Since(start).Round(time.Millisecond))
	if result == nil {
		fmt.Println("null")
		return nil
	}

	pretty, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Printf("%+v\n", result)
		return nil
	}
	fmt.Println(string(pretty))
	return nil
}

func parseTriggerPayload(payloadString, payloadFile string) (map[string]interface{}, error) {
	if payloadString != "" && payloadFile != "" {
		return nil, fmt.Errorf("use either --trigger-payload or --trigger-payload-file, not both")
	}

	var payloadBytes []byte
	if payloadFile != "" {
		data, err := os.ReadFile(payloadFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read payload file: %w", err)
		}
		payloadBytes = data
	} else if payloadString != "" {
		payloadBytes = []byte(payloadString)
	} else {
		return map[string]interface{}{}, nil
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, fmt.Errorf("payload must be valid JSON object: %w", err)
	}
	return payload, nil
}

func printBotHelp() {
	fmt.Println("BotAgent commands:")
	fmt.Println("  /help                      Show help")
	fmt.Println("  /exit                      Exit")
	fmt.Println("  /transport sse|rest        Switch message transport")
	fmt.Println("  /debug on|off              Toggle debug for REST /message")
	fmt.Println("  /blob <path> [mime]        Upload blob and attach to conversation")
	fmt.Println("  /blobs                     Show attached blob IDs")
	fmt.Println("  /clear-blobs               Clear attached blob IDs")
	fmt.Println("  /channel [id]              Show or switch channel")
	fmt.Println("  /reset [text]              Send RESET_CHANNEL message")
	fmt.Println("  <any text>                 Send normal message")
}

func setupLogging() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "15:04:05.000",
		ForceColors:     true,
	})

	level, err := log.ParseLevel(*logLevel)
	if err != nil {
		log.Warnf("Invalid log level '%s', using 'info'", *logLevel)
		level = log.InfoLevel
	}
	log.SetLevel(level)
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
