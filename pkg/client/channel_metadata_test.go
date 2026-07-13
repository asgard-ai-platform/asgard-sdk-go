package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.asgard-ai.com/asgard-sdk-go/pkg/models"
)

// A history rejoin surfaces the additive messageUser + channelTitleUpdate facts
// through Current() unchanged (pure unmarshal, no per-type dispatch).
func TestChannelStreamerSurfacesUserAndTitleFacts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		writeFrame(w, "1", string(models.SseEventTypeMessageUser), models.GenericBotSseEvent{
			EventType: models.SseEventTypeMessageUser,
			Fact: models.GenericBotSseEventFact{MessageUser: &models.GenericBotSseEventFactMessageUser{
				MessageId: "u1", Text: "hello", IdentityHint: "primary", CustomMessageId: "cm1", BlobIds: []string{"b1"},
			}},
		})
		writeFrame(w, "1", string(models.SseEventTypeChannelTitleUpdate), models.GenericBotSseEvent{
			EventType: models.SseEventTypeChannelTitleUpdate,
			Fact:      models.GenericBotSseEventFact{ChannelTitleUpdate: &models.GenericBotSseEventFactChannelTitleUpdate{Title: "Trip Planning"}},
		})
		et, ev := doneEvent()
		writeFrame(w, "1", et, ev)
	}))
	defer srv.Close()

	s, err := NewChannelStreaming(context.Background(), testConfig(srv.URL), "chan-1", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	var sawUser, sawTitle bool
	for s.Next() {
		cur := s.Current()
		switch cur.EventType {
		case models.SseEventTypeMessageUser:
			mu := cur.Fact.MessageUser
			if mu == nil || mu.Text != "hello" || mu.IdentityHint != "primary" || mu.MessageId != "u1" || mu.CustomMessageId != "cm1" || len(mu.BlobIds) != 1 || mu.BlobIds[0] != "b1" {
				t.Fatalf("messageUser fact wrong: %+v", mu)
			}
			sawUser = true
		case models.SseEventTypeChannelTitleUpdate:
			ct := cur.Fact.ChannelTitleUpdate
			if ct == nil || ct.Title != "Trip Planning" {
				t.Fatalf("channelTitleUpdate fact wrong: %+v", ct)
			}
			sawTitle = true
		}
	}
	if err := s.Err(); err != nil {
		t.Fatalf("stream err: %v", err)
	}
	if !sawUser || !sawTitle {
		t.Fatalf("missing facts: user=%v title=%v", sawUser, sawTitle)
	}
}

// ChannelMetadata GETs the metadata endpoint and decodes the envelope.
func TestChannelMetadata(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/ns/ns/bot-provider/bp/channel/metadata" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if q := r.URL.Query().Get("custom_channel_id"); q != "chan-1" {
			t.Errorf("custom_channel_id = %q", q)
		}
		if k := r.Header.Get("X-API-KEY"); k != "key" {
			t.Errorf("api key = %q", k)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"isSuccess": true,
			"data":      map[string]any{"customChannelId": "chan-1", "title": "Trip Planning", "runState": "IDLE", "lastActivityAt": 1720000000000},
		})
	}))
	defer srv.Close()

	c := NewBotProviderClientWithConfig(testConfig(srv.URL))
	md, err := c.ChannelMetadata(context.Background(), "chan-1")
	if err != nil {
		t.Fatal(err)
	}
	if md.CustomChannelId != "chan-1" || md.Title == nil || *md.Title != "Trip Planning" || md.RunState != "IDLE" || md.LastActivityAt != 1720000000000 {
		t.Fatalf("metadata wrong: %+v (title=%v)", md, md.Title)
	}
}

// An unknown channel is a 404 surfaced as an *APIError that IsNotFound matches.
func TestChannelMetadataNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"isSuccess": false, "data": nil, "error": "channel not found", "errorCode": "CHANNEL_NOT_FOUND",
		})
	}))
	defer srv.Close()

	c := NewBotProviderClientWithConfig(testConfig(srv.URL))
	if _, err := c.ChannelMetadata(context.Background(), "nope"); err == nil {
		t.Fatal("expected error for unknown channel")
	} else if !IsNotFound(err) {
		t.Fatalf("expected IsNotFound, got %v", err)
	}
}
