package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/tmaxmax/go-sse"
	"go.asgard-ai.com/asgard-sdk-go/pkg/models"
)

// BotProviderStreamer defines the interface for streaming bot provider events.
type BotProviderStreamer interface {
	// Next advances to the next event. Returns false when the turn ends
	// (asgard.run.done / asgard.run.error), the stream is closed, or an error
	// occurs. Transient connection drops are resumed transparently in between and
	// are invisible here.
	Next() bool
	// Current returns the event Next() last surfaced.
	Current() *models.GenericBotSseEvent
	// LastEventID returns the SSE id: of the event Current() last returned — the
	// durable resume cursor for that event. Persist it (or, when relaying, forward
	// it downstream as the SSE id:) so a later reconnect can resume via
	// MessageRequestOptions.LastEventID / ChannelStreamOptions.LastEventID. Empty
	// before the first event.
	LastEventID() string
	// Err returns the terminal error, if any (a run.error, a non-2xx response, or
	// a fatal connection failure). nil after a clean run.done end.
	Err() error
	// Close stops the stream and releases its resources. Idempotent. It only
	// detaches this client: the run executes in the background on the server, so
	// for a POST send (NewStreaming) the message was already dispatched and the
	// turn keeps running to completion server-side — Close does not cancel it.
	Close() error
}

type streamMode int

const (
	modeSend    streamMode = iota // POST /message/sse — dispatch a message + stream one turn
	modeChannel                   // GET  /message/sse — rejoin (replay history + stream the turn)
)

// Reconnect backoff. The SDK owns reconnection (go-sse is configured for a single
// attempt per Connect), so these govern the transparent-resume backoff.
const (
	reconnectInitialBackoff = 500 * time.Millisecond
	reconnectMaxBackoff     = 5 * time.Second
	reconnectBackoffFactor  = 2
)

// streamItem carries one parsed event (or a fatal error) from the producer
// goroutine to Next(), tagged with the SSE id: it arrived with.
type streamItem struct {
	event *models.GenericBotSseEvent
	id    string
	err   error
}

// botProviderStream implements BotProviderStreamer. A single producer goroutine
// (run) owns the connection lifecycle, reconnection, and the resume cursor;
// Next()/Current()/Err()/LastEventID() are the consumer side; Close() may be
// called from any goroutine.
type botProviderStream struct {
	// immutable after construction
	userCtx   context.Context
	streamCtx context.Context
	cancel    context.CancelFunc
	config    *BotProviderConfig
	sseClient *sse.Client
	mode      streamMode

	// request inputs (immutable after construction)
	message               *models.GenericBotMessage // modeSend
	customChannelID       string                    // modeChannel
	isDebug               bool                      // modeSend
	bypassToolCallConsent bool                      // modeSend
	userIdentityHint      string

	eventChan chan streamItem

	// producer-goroutine-only state (validator + onEvent + run all run on the
	// single producer goroutine, synchronously inside Connect) — no locking needed.
	reconnectCursor string // last SSE id: seen; written as Last-Event-ID on reconnect
	httpStatus      int    // status of the current attempt's response (0 = none)
	validatorOK     bool   // current attempt established a 200 text/event-stream

	// consumer-side state
	mu           sync.Mutex
	currentEvent *models.GenericBotSseEvent
	currentID    string
	err          error
	closed       bool
}

// NewStreaming opens a POST /message/sse send-and-stream: it dispatches message
// and streams that turn until its terminal (run.done / run.error). Transient
// drops mid-turn are resumed transparently via Last-Event-ID. If
// opts.LastEventID is set, the message is NOT re-dispatched — the server
// resubscribes from that cursor (use this to relay a downstream client's
// reconnect without duplicating the turn).
func NewStreaming(ctx context.Context, config *BotProviderConfig, message *models.GenericBotMessage, opts *MessageRequestOptions) (BotProviderStreamer, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if message == nil {
		return nil, fmt.Errorf("message cannot be nil")
	}
	if opts == nil {
		opts = &MessageRequestOptions{}
	}
	s := &botProviderStream{
		config:                config,
		mode:                  modeSend,
		message:               message,
		isDebug:               opts.IsDebug,
		bypassToolCallConsent: opts.BypassToolCallConsent,
		userIdentityHint:      opts.UserIdentityHint,
		reconnectCursor:       opts.LastEventID,
	}
	return startStream(ctx, s), nil
}

// NewChannelStreaming opens a GET /message/sse rejoin: it replays the channel's
// collapsed history (from opts.LastEventID, or the full history when empty) and
// then streams the in-flight turn until its terminal. It does NOT dispatch a
// message. Transient drops are resumed transparently.
func NewChannelStreaming(ctx context.Context, config *BotProviderConfig, customChannelID string, opts *ChannelStreamOptions) (BotProviderStreamer, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if customChannelID == "" {
		return nil, fmt.Errorf("customChannelID cannot be empty")
	}
	if opts == nil {
		opts = &ChannelStreamOptions{}
	}
	s := &botProviderStream{
		config:           config,
		mode:             modeChannel,
		customChannelID:  customChannelID,
		userIdentityHint: opts.UserIdentityHint,
		reconnectCursor:  opts.LastEventID,
	}
	return startStream(ctx, s), nil
}

// startStream wires the shared context/sse-client/goroutine for a constructed stream.
func startStream(ctx context.Context, s *botProviderStream) *botProviderStream {
	s.userCtx = ctx
	s.streamCtx, s.cancel = context.WithCancel(ctx)
	s.eventChan = make(chan streamItem, 100)
	s.sseClient = &sse.Client{
		Backoff:           sse.Backoff{MaxRetries: -1}, // -1 = single attempt; the SDK owns reconnection
		ResponseValidator: s.validate,
	}
	if s.config.HTTPClient != nil {
		s.sseClient.HTTPClient = s.config.HTTPClient
	}
	go s.run()
	return s
}

// validate records the response status (for the reconnect decision) and accepts
// only a 200 text/event-stream. Runs on the producer goroutine inside Connect.
func (s *botProviderStream) validate(res *http.Response) error {
	s.httpStatus = res.StatusCode
	s.validatorOK = false
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
		return fmt.Errorf("unexpected content-type %q", ct)
	}
	s.validatorOK = true
	return nil
}

// run is the producer goroutine: connect → (on drop) classify → backoff → resume,
// until a clean end (terminal / Close / user-ctx cancel) or a surfaced error.
// It is the sole closer of eventChan.
func (s *botProviderStream) run() {
	defer close(s.eventChan)

	backoff := reconnectInitialBackoff
	everEstablished := false
	for {
		if s.streamCtx.Err() != nil {
			return
		}
		established, connErr := s.connectOnce()
		if established {
			everEstablished = true
			backoff = reconnectInitialBackoff // reset after a good stream
		}
		// A terminal event cancels streamCtx from onEvent; Close() cancels it too.
		// Either way this is a clean stop — no reconnect. (User-ctx cancellation is
		// surfaced by Next via userCtx.)
		if s.streamCtx.Err() != nil {
			return
		}
		if !s.shouldReconnect(established, everEstablished) {
			s.send(streamItem{err: s.connError(connErr)})
			return
		}
		log.WithFields(log.Fields{
			"mode":     s.mode,
			"sinceSeq": s.reconnectCursor,
			"status":   s.httpStatus,
		}).Debug("[EdgeServer] SSE dropped, resuming from Last-Event-ID")
		select {
		case <-time.After(backoff):
		case <-s.streamCtx.Done():
			return
		}
		backoff = nextBackoff(backoff)
	}
}

// connectOnce runs one connection attempt. Returns whether it established a 200
// event-stream (i.e. read actually started) and the connection error.
func (s *botProviderStream) connectOnce() (established bool, err error) {
	s.httpStatus = 0
	s.validatorOK = false

	req, err := s.buildRequest()
	if err != nil {
		return false, err
	}
	conn := s.sseClient.NewConnection(req)
	buf := make([]byte, 0, 1024*1024) // 1MB start
	conn.Buffer(buf, 1024*1024*10)    // 10MB max token, to survive large events
	conn.SubscribeToAll(s.onEvent)
	err = conn.Connect()
	return s.validatorOK, err
}

// shouldReconnect implements the "only resume an established 200 stream" policy:
//   - a definitive non-200 HTTP response (incl. 5xx) → surface, never resume;
//   - a 200 stream that dropped (this attempt), or a network error (no response)
//     after we've had a 200 before → resume (subject to canResume);
//   - a first-connect failure that never reached 200 → surface (prevents a POST
//     double-dispatch and fails fast on startup errors).
func (s *botProviderStream) shouldReconnect(established, everEstablished bool) bool {
	if st := s.httpStatus; st != 0 && st != http.StatusOK {
		return false
	}
	if established || everEstablished {
		return s.canResume()
	}
	return false
}

// canResume guards the POST empty-cursor window: reconnecting a POST without a
// Last-Event-ID would make the server re-dispatch the message (a duplicate turn),
// so we only resume once a cursor exists. GET carries no dispatch, so resuming
// from the full history (empty cursor) is always safe.
func (s *botProviderStream) canResume() bool {
	if s.mode == modeSend && s.reconnectCursor == "" {
		return false
	}
	return true
}

// onEvent parses one SSE frame, advances the resume cursor, hands the event to
// the consumer, and cancels (clean stop, no reconnect) on a turn terminal. Runs
// on the producer goroutine.
func (s *botProviderStream) onEvent(ev sse.Event) {
	var edgeEvent models.GenericBotSseEvent
	if err := json.Unmarshal([]byte(ev.Data), &edgeEvent); err != nil {
		log.WithError(err).WithField("raw_data", ev.Data).Error("[EdgeServer] Failed to unmarshal SSE event")
		s.send(streamItem{err: fmt.Errorf("failed to unmarshal event: %w", err)})
		return
	}
	// ev.LastEventID is the SSE id: carried forward per spec (the durable
	// transcript seq); ephemeral events without their own id inherit the last one.
	if ev.LastEventID != "" {
		s.reconnectCursor = ev.LastEventID
	}
	s.send(streamItem{event: &edgeEvent, id: ev.LastEventID})

	// A per-request terminal ends this stream cleanly for BOTH modes: cancel so
	// the producer stops without reconnecting into a synthesized run.init/run.done.
	if edgeEvent.EventType == models.SseEventTypeRunDone || edgeEvent.EventType == models.SseEventTypeRunError {
		s.cancel()
	}
}

// send hands an item to the consumer, abandoning it if the stream is being torn
// down (guards against a producer goroutine leak when the buffer is full and the
// consumer has stopped).
func (s *botProviderStream) send(it streamItem) {
	select {
	case s.eventChan <- it:
	case <-s.streamCtx.Done():
	}
}

// buildRequest constructs a fresh request for the current attempt, stamping the
// resume cursor as the Last-Event-ID header when present.
func (s *botProviderStream) buildRequest() (*http.Request, error) {
	if s.mode == modeChannel {
		return s.buildChannelRequest()
	}
	return s.buildSendRequest()
}

func (s *botProviderStream) buildSendRequest() (*http.Request, error) {
	body, err := json.Marshal(s.message)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal bot message: %w", err)
	}
	u, err := url.Parse(fmt.Sprintf("%s/ns/%s/bot-provider/%s/message/sse",
		s.config.EdgeServerHost, s.config.Namespace, s.config.BotProviderName))
	if err != nil {
		return nil, fmt.Errorf("failed to parse SSE URL: %w", err)
	}
	q := u.Query()
	if s.isDebug {
		q.Set("is_debug", "true")
	}
	if s.bypassToolCallConsent {
		q.Set("bypass_tool_call_consent", "true")
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(s.streamCtx, http.MethodPost, u.String(), bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create SSE request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	s.applyCommonHeaders(req)
	return req, nil
}

func (s *botProviderStream) buildChannelRequest() (*http.Request, error) {
	u, err := url.Parse(fmt.Sprintf("%s/ns/%s/bot-provider/%s/message/sse",
		s.config.EdgeServerHost, s.config.Namespace, s.config.BotProviderName))
	if err != nil {
		return nil, fmt.Errorf("failed to parse SSE URL: %w", err)
	}
	q := u.Query()
	q.Set("custom_channel_id", s.customChannelID)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(s.streamCtx, http.MethodGet, u.String(), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSE request: %w", err)
	}
	s.applyCommonHeaders(req)
	return req, nil
}

// applyCommonHeaders sets auth, caller headers, identity hint, and the resume
// cursor shared by both request shapes.
func (s *botProviderStream) applyCommonHeaders(req *http.Request) {
	req.Header.Set("x-api-key", s.config.BotProviderApiKey)
	for k, v := range s.config.Headers {
		req.Header.Set(k, v)
	}
	if s.userIdentityHint != "" {
		req.Header.Set("X-ASGARD-USER-IDENTITY-HINT", s.userIdentityHint)
	}
	if s.reconnectCursor != "" {
		req.Header.Set("Last-Event-ID", s.reconnectCursor)
	}
}

// connError renders the terminal error surfaced when the SDK stops without a
// clean end: an *APIError for a non-200 response (parity with the REST methods),
// otherwise the wrapped connection failure.
func (s *botProviderStream) connError(connErr error) error {
	if st := s.httpStatus; st != 0 && st != http.StatusOK {
		return &APIError{StatusCode: st, Op: s.op()}
	}
	if connErr != nil {
		return fmt.Errorf("SSE connection failed: %w", connErr)
	}
	return errors.New("SSE connection failed")
}

func (s *botProviderStream) op() string {
	if s.mode == modeChannel {
		return "rejoin channel sse"
	}
	return "stream message sse"
}

// Next advances to the next event. See BotProviderStreamer.Next.
func (s *botProviderStream) Next() bool {
	s.mu.Lock()
	if s.closed || s.err != nil {
		s.mu.Unlock()
		return false
	}
	s.mu.Unlock()

	// Block on the channel WITHOUT holding s.mu so Close() can cancel promptly.
	// Close()/user-ctx cancel unblocks this by making the producer close eventChan.
	it, ok := <-s.eventChan
	if !ok {
		s.mu.Lock()
		if s.err == nil && s.userCtx.Err() != nil {
			s.err = s.userCtx.Err()
		}
		s.mu.Unlock()
		return false
	}
	if it.err != nil {
		s.mu.Lock()
		s.err = it.err
		s.mu.Unlock()
		return false
	}
	// run.error ends the stream and is surfaced via Err() (not Current()), matching
	// prior behavior. run.done is delivered via Current(), then the next Next()
	// returns false (clean end).
	if it.event != nil && it.event.EventType == models.SseEventTypeRunError {
		s.mu.Lock()
		s.err = runError(it.event)
		s.mu.Unlock()
		return false
	}

	s.mu.Lock()
	s.currentEvent = it.event
	s.currentID = it.id
	s.mu.Unlock()
	return true
}

// Current returns the current event. Only valid after Next() returns true.
func (s *botProviderStream) Current() *models.GenericBotSseEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.currentEvent
}

// LastEventID returns the resume cursor of the current event. See the interface.
func (s *botProviderStream) LastEventID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.currentID
}

// Err returns any terminal error.
func (s *botProviderStream) Err() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.err
}

// Close stops the stream and releases resources. Idempotent; safe from any
// goroutine. It cancels streamCtx, which stops the producer's resume loop and
// unblocks Next(). Note this only detaches the client — a POST turn already
// dispatched keeps running to completion server-side; Close does not cancel it.
func (s *botProviderStream) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	s.currentEvent = nil
	s.mu.Unlock()

	s.cancel() // stops the producer (unblocks Connect / send / backoff), which closes eventChan
	return nil
}

func runError(ev *models.GenericBotSseEvent) error {
	if ev != nil && ev.Fact.RunError != nil {
		detail := ev.Fact.RunError.Error
		return fmt.Errorf("SSE stream error: %w", &detail)
	}
	return errors.New("SSE stream error")
}

func nextBackoff(cur time.Duration) time.Duration {
	next := cur * reconnectBackoffFactor
	if next > reconnectMaxBackoff {
		return reconnectMaxBackoff
	}
	return next
}
