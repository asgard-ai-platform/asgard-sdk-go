package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/tmaxmax/go-sse"
	"go.asgard-ai.com/asgard-sdk-go/pkg/models"
)

// BotProviderStreamer defines the interface for streaming bot provider events
type BotProviderStreamer interface {
	Next() bool
	Current() *models.GenericBotSseEvent
	Err() error
	Close() error
}

// botProviderStream implements BotProviderStreamer
type botProviderStream struct {
	ctx          context.Context
	config       *BotProviderConfig
	message      *models.GenericBotMessage
	sseClient    *sse.Client
	connection   *sse.Connection
	eventChan    chan models.GenericBotSseEventWrapper
	currentEvent *models.GenericBotSseEvent
	err          error
	closed       bool
	mu           sync.Mutex
}

// NewStreaming creates a new bot provider stream and establishes the SSE connection
func NewStreaming(ctx context.Context, config *BotProviderConfig, message *models.GenericBotMessage) (BotProviderStreamer, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if message == nil {
		return nil, fmt.Errorf("message cannot be nil")
	}

	sseClient := &sse.Client{
		Backoff: sse.Backoff{
			MaxRetries: -1,
		},
	}

	if config.HTTPClient != nil {
		sseClient.HTTPClient = config.HTTPClient
	}

	stream := &botProviderStream{
		ctx:       ctx,
		config:    config,
		message:   message,
		eventChan: make(chan models.GenericBotSseEventWrapper, 100),
		sseClient: sseClient,
	}

	if err := stream.connect(); err != nil {
		return nil, fmt.Errorf("failed to establish SSE connection: %w", err)
	}

	return stream, nil
}

// connect establishes the SSE connection
func (s *botProviderStream) connect() error {
	// Marshal the message
	messageBytes, err := json.Marshal(s.message)
	if err != nil {
		return fmt.Errorf("failed to marshal bot message: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/ns/%s/bot-provider/%s/message/sse",
		s.config.EdgeServerHost, s.config.Namespace, s.config.BotProviderName)

	// Log request details for debugging
	log.WithFields(log.Fields{
		"url":  url,
		"body": string(messageBytes),
	}).Info("[EdgeServer] Sending SSE request")

	req, err := http.NewRequestWithContext(s.ctx, http.MethodPost, url, bytes.NewBuffer(messageBytes))
	if err != nil {
		return fmt.Errorf("failed to create SSE request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", s.config.BotProviderApiKey)

	// Create SSE connection
	buf := make([]byte, 0, 1024*1024) // Buffer starting at 1MB
	maxToken := 1024 * 1024 * 10      // Buffer max token size at 10MB
	s.connection = s.sseClient.
		NewConnection(req)
	s.connection.Buffer(buf, maxToken) // Set buffer size to 1MB and max token to 10MB to prevent token too long error

	// Subscribe to events
	s.connection.SubscribeToAll(func(event sse.Event) {
		// Log raw SSE event for debugging
		log.WithFields(log.Fields{
			"event_type": event.Type,
			"event_data": event.Data,
		}).Debug("[EdgeServer] Received SSE event")

		var edgeEvent models.GenericBotSseEvent
		if err := json.Unmarshal([]byte(event.Data), &edgeEvent); err != nil {
			log.WithError(err).WithField("raw_data", event.Data).Error("[EdgeServer] Failed to unmarshal SSE event")
			s.eventChan <- models.GenericBotSseEventWrapper{
				Event:           nil,
				ConnectionError: fmt.Errorf("failed to unmarshal event: %w", err),
			}
		} else {
			log.WithFields(log.Fields{
				"event_type": edgeEvent.EventType,
				"request_id": edgeEvent.RequestId,
				"event_id":   edgeEvent.EventId,
			}).Debug("[EdgeServer] Parsed SSE event")

			s.eventChan <- models.GenericBotSseEventWrapper{
				Event:           &edgeEvent,
				ConnectionError: nil,
			}
		}
	})

	// Start connection in a goroutine
	go func() {
		defer close(s.eventChan)
		if err := s.connection.Connect(); !errors.Is(err, io.EOF) {
			log.WithError(err).Error("[EdgeServer] SSE connection failed")
			s.eventChan <- models.GenericBotSseEventWrapper{
				Event:           nil,
				ConnectionError: fmt.Errorf("SSE connection failed: %w", err),
			}
		} else {
			log.Debug("[EdgeServer] SSE connection closed normally")
		}
	}()

	return nil
}

// Next advances to the next event. Returns false if there are no more events or an error occurred.
func (s *botProviderStream) Next() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed || s.err != nil {
		return false
	}

	select {
	case ev, ok := <-s.eventChan:
		if !ok {
			// Channel closed, no more events
			return false
		}

		if ev.ConnectionError != nil {
			s.err = ev.ConnectionError
			return false
		}

		// Check for run error events
		if ev.Event.EventType == models.SseEventTypeRunError {
			s.err = fmt.Errorf("SSE stream error: %s", ev.Event.Fact.RunError.Error)
			return false
		}

		s.currentEvent = ev.Event
		return true

	case <-s.ctx.Done():
		s.err = s.ctx.Err()
		return false
	}
}

// Current returns the current event. Should only be called after Next() returns true.
func (s *botProviderStream) Current() *models.GenericBotSseEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.currentEvent
}

// Err returns any error that occurred during streaming
func (s *botProviderStream) Err() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.err
}

// Close closes the stream and cleans up resources
func (s *botProviderStream) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true

	// Clear current event reference to help GC
	s.currentEvent = nil

	return nil
}
