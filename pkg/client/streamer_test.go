package client

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"go.asgard-ai.com/asgard-sdk-go/pkg/models"
)

// --- helpers ---------------------------------------------------------------

func sseFrame(id, eventType string, ev models.GenericBotSseEvent) string {
	data, _ := json.Marshal(ev)
	out := ""
	if id != "" {
		out += "id: " + id + "\n"
	}
	out += "event: " + eventType + "\n"
	out += "data: " + string(data) + "\n\n"
	return out
}

func deltaEvent(text string) (string, models.GenericBotSseEvent) {
	return string(models.SseEventTypeMessageDelta), models.GenericBotSseEvent{
		EventType: models.SseEventTypeMessageDelta,
		Fact:      models.GenericBotSseEventFact{MessageDelta: &models.GenericBotSseEventFactMessage{Message: models.BufferedMessage{Text: text}}},
	}
}

func doneEvent() (string, models.GenericBotSseEvent) {
	return string(models.SseEventTypeRunDone), models.GenericBotSseEvent{
		EventType: models.SseEventTypeRunDone,
		Fact:      models.GenericBotSseEventFact{RunDone: &models.GenericBotSseEventFactRunDone{}},
	}
}

func errorEvent(msg string) (string, models.GenericBotSseEvent) {
	return string(models.SseEventTypeRunError), models.GenericBotSseEvent{
		EventType: models.SseEventTypeRunError,
		Fact:      models.GenericBotSseEventFact{RunError: &models.GenericBotSseEventFactRunError{Error: models.ErrorDetail{Message: msg, Code: "INTERNAL"}}},
	}
}

func writeFrame(w http.ResponseWriter, id string, eventType string, ev models.GenericBotSseEvent) {
	io.WriteString(w, sseFrame(id, eventType, ev))
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

func testConfig(host string) *BotProviderConfig {
	return &BotProviderConfig{
		HTTPClient:        &http.Client{},
		EdgeServerHost:    host,
		Namespace:         "ns",
		BotProviderName:   "bp",
		BotProviderApiKey: "key",
	}
}

func testMessage() *models.GenericBotMessage {
	return &models.GenericBotMessage{CustomChannelId: "chan-1", CustomMessageId: "m1", Text: "hi", Action: models.PostBackActionNone}
}

// drain runs the stream to completion, returning the event types seen via
// Current() (run.done is included; run.error is not — it lands on Err()).
func drain(s BotProviderStreamer) []models.SseEventType {
	var got []models.SseEventType
	for s.Next() {
		got = append(got, s.Current().EventType)
	}
	return got
}

// --- tests -----------------------------------------------------------------

// P1: a POST turn ends exactly once at run.done — no reconnect loop.
func TestPostStopsOnceAtRunDone(t *testing.T) {
	var mu sync.Mutex
	var reqs int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		reqs++
		mu.Unlock()
		w.Header().Set("Content-Type", "text/event-stream")
		et, ev := deltaEvent("a")
		writeFrame(w, "1", et, ev)
		et, ev = doneEvent()
		writeFrame(w, "1", et, ev)
	}))
	defer srv.Close()

	s, err := NewStreaming(context.Background(), testConfig(srv.URL), testMessage(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	got := drain(s)
	if len(got) != 2 || got[0] != models.SseEventTypeMessageDelta || got[1] != models.SseEventTypeRunDone {
		t.Fatalf("unexpected events: %v", got)
	}
	if s.Err() != nil {
		t.Fatalf("expected nil Err after clean run.done, got %v", s.Err())
	}
	// Give any erroneous reconnect a chance to fire.
	time.Sleep(200 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	if reqs != 1 {
		t.Fatalf("expected exactly 1 request (no reconnect after run.done), got %d", reqs)
	}
}

// run.error ends the stream via Err(), not Current().
func TestPostRunErrorSurfacesViaErr(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		et, ev := errorEvent("boom")
		writeFrame(w, "0", et, ev)
	}))
	defer srv.Close()

	s, _ := NewStreaming(context.Background(), testConfig(srv.URL), testMessage(), nil)
	defer s.Close()

	got := drain(s)
	if len(got) != 0 {
		t.Fatalf("run.error must not be delivered via Current(), got %v", got)
	}
	if s.Err() == nil {
		t.Fatal("expected Err() to be set on run.error")
	}
	var detail *models.ErrorDetail
	if !errors.As(s.Err(), &detail) || detail.Message != "boom" {
		t.Fatalf("expected wrapped ErrorDetail{boom}, got %v", s.Err())
	}
}

// P2: an established 200 stream that drops mid-turn resumes transparently with
// Last-Event-ID; the consumer sees a continuous event sequence.
func TestResumesEstablishedStreamOnDrop(t *testing.T) {
	var mu sync.Mutex
	var reqs int
	var lastEventIDs []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		reqs++
		n := reqs
		lastEventIDs = append(lastEventIDs, r.Header.Get("Last-Event-ID"))
		mu.Unlock()
		w.Header().Set("Content-Type", "text/event-stream")
		if n == 1 {
			et, ev := deltaEvent("a")
			writeFrame(w, "1", et, ev)
			return // drop mid-turn
		}
		et, ev := deltaEvent("b")
		writeFrame(w, "2", et, ev)
		et, ev = doneEvent()
		writeFrame(w, "2", et, ev)
	}))
	defer srv.Close()

	s, _ := NewStreaming(context.Background(), testConfig(srv.URL), testMessage(), nil)
	defer s.Close()

	var texts []string
	for s.Next() {
		if d := s.Current().Fact.MessageDelta; d != nil {
			texts = append(texts, d.Message.Text)
		}
	}
	if s.Err() != nil {
		t.Fatalf("expected clean end after resume, got %v", s.Err())
	}
	if len(texts) != 2 || texts[0] != "a" || texts[1] != "b" {
		t.Fatalf("expected [a b] across resume, got %v", texts)
	}
	mu.Lock()
	defer mu.Unlock()
	if reqs != 2 {
		t.Fatalf("expected 2 requests (1 drop + 1 resume), got %d", reqs)
	}
	if lastEventIDs[0] != "" {
		t.Fatalf("first request must not carry Last-Event-ID, got %q", lastEventIDs[0])
	}
	if lastEventIDs[1] != "1" {
		t.Fatalf("resume must carry Last-Event-ID from last frame, want 1 got %q", lastEventIDs[1])
	}
}

// A non-200 response on reconnect is surfaced as *APIError with no further
// reconnect ("only reconnect a 200 stream").
func TestNon200OnReconnectSurfacesAndStops(t *testing.T) {
	var mu sync.Mutex
	var reqs int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		reqs++
		n := reqs
		mu.Unlock()
		if n == 1 {
			w.Header().Set("Content-Type", "text/event-stream")
			et, ev := deltaEvent("a")
			writeFrame(w, "1", et, ev)
			return // drop
		}
		http.Error(w, "deploying", http.StatusServiceUnavailable) // 503 on reconnect
	}))
	defer srv.Close()

	s, _ := NewStreaming(context.Background(), testConfig(srv.URL), testMessage(), nil)
	defer s.Close()

	got := drain(s)
	if len(got) != 1 || got[0] != models.SseEventTypeMessageDelta {
		t.Fatalf("expected the pre-drop delta, got %v", got)
	}
	var apiErr *APIError
	if !errors.As(s.Err(), &apiErr) || apiErr.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected *APIError 503, got %v", s.Err())
	}
	// Give any (erroneous) extra reconnect a chance to fire.
	time.Sleep(200 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	if reqs != 2 {
		t.Fatalf("expected exactly 2 requests (no retry past 503), got %d", reqs)
	}
}

// An initial non-200 is surfaced immediately with no reconnect (fail fast; also
// prevents a POST double-dispatch).
func TestInitialNon200SurfacesNoReconnect(t *testing.T) {
	var mu sync.Mutex
	var reqs int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		reqs++
		mu.Unlock()
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	defer srv.Close()

	s, _ := NewStreaming(context.Background(), testConfig(srv.URL), testMessage(), nil)
	defer s.Close()

	if drain(s) != nil {
		t.Fatal("expected no events")
	}
	var apiErr *APIError
	if !errors.As(s.Err(), &apiErr) || apiErr.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected *APIError 500, got %v", s.Err())
	}
	time.Sleep(200 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	if reqs != 1 {
		t.Fatalf("expected exactly 1 request (no reconnect on initial failure), got %d", reqs)
	}
}

// Relay inbound: an explicit LastEventID is sent on the very first request (POST
// resubscribe), so a relayed downstream reconnect does not lose the cursor.
func TestExplicitLastEventIDOnFirstRequest(t *testing.T) {
	got := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case got <- r.Header.Get("Last-Event-ID"):
		default:
		}
		w.Header().Set("Content-Type", "text/event-stream")
		et, ev := doneEvent()
		writeFrame(w, "42", et, ev)
	}))
	defer srv.Close()

	s, _ := NewStreaming(context.Background(), testConfig(srv.URL), testMessage(), &MessageRequestOptions{LastEventID: "42"})
	defer s.Close()
	drain(s)

	select {
	case v := <-got:
		if v != "42" {
			t.Fatalf("first request Last-Event-ID: want 42 got %q", v)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server never received a request")
	}
}

// LastEventID() is bound to the event Current() last returned (not "the latest
// the producer has seen"), so a relay stamps the right id per event.
func TestLastEventIDBoundToCurrent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		et, ev := deltaEvent("a")
		writeFrame(w, "5", et, ev)
		et, ev = deltaEvent("b")
		writeFrame(w, "6", et, ev)
		et, ev = doneEvent()
		writeFrame(w, "6", et, ev)
	}))
	defer srv.Close()

	s, _ := NewStreaming(context.Background(), testConfig(srv.URL), testMessage(), nil)
	defer s.Close()

	var ids []string
	for s.Next() {
		if s.Current().EventType == models.SseEventTypeMessageDelta {
			ids = append(ids, s.LastEventID())
		}
	}
	if len(ids) != 2 || ids[0] != "5" || ids[1] != "6" {
		t.Fatalf("expected per-event ids [5 6], got %v", ids)
	}
}

// GET rejoin uses GET + custom_channel_id and stops at the terminal like POST.
func TestChannelStreamerRejoinStopsAtTerminal(t *testing.T) {
	gotMethod := make(chan string, 1)
	gotChannel := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case gotMethod <- r.Method:
		default:
		}
		select {
		case gotChannel <- r.URL.Query().Get("custom_channel_id"):
		default:
		}
		w.Header().Set("Content-Type", "text/event-stream")
		et, ev := deltaEvent("history")
		writeFrame(w, "7", et, ev)
		et, ev = doneEvent()
		writeFrame(w, "7", et, ev)
	}))
	defer srv.Close()

	s, err := NewChannelStreaming(context.Background(), testConfig(srv.URL), "chan-9", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	got := drain(s)
	if len(got) != 2 || got[1] != models.SseEventTypeRunDone {
		t.Fatalf("expected [delta done], got %v", got)
	}
	if m := <-gotMethod; m != http.MethodGet {
		t.Fatalf("expected GET, got %s", m)
	}
	if c := <-gotChannel; c != "chan-9" {
		t.Fatalf("expected custom_channel_id=chan-9, got %q", c)
	}
}

// Close() mid-stream returns promptly and unblocks a blocked Next().
func TestCloseUnblocksNext(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		et, ev := deltaEvent("a")
		writeFrame(w, "1", et, ev)
		<-release // hold the connection open (no terminal)
	}))
	defer srv.Close()
	defer close(release)

	s, _ := NewStreaming(context.Background(), testConfig(srv.URL), testMessage(), nil)
	if !s.Next() { // the first delta
		t.Fatalf("expected first event, err=%v", s.Err())
	}

	done := make(chan bool, 1)
	go func() {
		done <- s.Next() // blocks until Close()
	}()
	time.Sleep(100 * time.Millisecond)
	s.Close()

	select {
	case v := <-done:
		if v {
			t.Fatal("Next() should return false after Close()")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Close() did not unblock Next()")
	}
}

// A poison frame — one SSE event bigger than the scanner buffer — must fail
// fast, not retry: the resume cursor points AT the oversized event, so every
// reconnect would replay it first and die identically (the 2026-08-11 prod
// incident: a 26 MB replayed tool_call.complete turned one open rejoin into an
// unbounded ~3s retry storm, with the channel's history permanently unloadable).
func TestPoisonOversizedEventFailsFastNoRetryStorm(t *testing.T) {
	giant := strings.Repeat("A", 65*1024*1024) // over the 64 MB scanner fuse
	var mu sync.Mutex
	var reqs int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		reqs++
		mu.Unlock()
		w.Header().Set("Content-Type", "text/event-stream")
		et, ev := deltaEvent("history before the poison frame")
		writeFrame(w, "80941", et, ev)
		// One event whose data: line exceeds the scanner buffer.
		io.WriteString(w, "id: 80951\nevent: asgard.tool_call.complete\ndata: "+giant+"\n\n")
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}))
	defer srv.Close()

	s, err := NewChannelStreaming(context.Background(), testConfig(srv.URL), "chan-poison", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	got := drain(s)
	if len(got) != 1 || got[0] != models.SseEventTypeMessageDelta {
		t.Fatalf("expected only the pre-poison event, got %v", got)
	}
	if s.Err() == nil || !errors.Is(s.Err(), bufio.ErrTooLong) {
		t.Fatalf("expected a surfaced bufio.ErrTooLong, got %v", s.Err())
	}
	// Give any erroneous reconnect a chance to fire before counting.
	time.Sleep(300 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	if reqs != 1 {
		t.Fatalf("poison frame must not be retried: got %d requests", reqs)
	}
}
