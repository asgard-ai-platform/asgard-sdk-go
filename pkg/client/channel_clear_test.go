package client

import (
	"context"
	"net/http"
	"testing"
)

// ClearChannel is a POST on the channel's clear action, addressed by the natural
// key (path = ns/bot, query = custom_channel_id) and authenticated like every
// other call; a 204 is success with nothing to decode.
func TestClearChannelSendsPostWithNaturalKey(t *testing.T) {
	srv, got := captureDelete(t, http.StatusNoContent, "")
	c := NewBotProviderClientWithConfig(testConfig(srv.URL))

	if err := c.ClearChannel(context.Background(), "room 1"); err != nil {
		t.Fatalf("ClearChannel: %v", err)
	}
	r := *got
	if r == nil {
		t.Fatal("no request reached the server")
	}
	if r.Method != http.MethodPost {
		t.Errorf("method = %s, want POST", r.Method)
	}
	if r.URL.Path != "/ns/ns/bot-provider/bp/channel/clear" {
		t.Errorf("path = %s, want /ns/ns/bot-provider/bp/channel/clear", r.URL.Path)
	}
	if r.URL.Query().Get("custom_channel_id") != "room 1" {
		t.Errorf("custom_channel_id = %q, want \"room 1\" (must be query-encoded)", r.URL.Query().Get("custom_channel_id"))
	}
	if r.Header.Get("X-API-KEY") != "key" {
		t.Errorf("X-API-KEY = %q, want key", r.Header.Get("X-API-KEY"))
	}
}

// A non-2xx is surfaced as an *APIError so callers can branch on the status.
func TestClearChannelSurfacesAPIError(t *testing.T) {
	srv, _ := captureDelete(t, http.StatusInternalServerError,
		`{"isSuccess":false,"errorCode":"INTERNAL","message":"clear failed"}`)
	c := NewBotProviderClientWithConfig(testConfig(srv.URL))

	err := c.ClearChannel(context.Background(), "room-1")
	if err == nil {
		t.Fatal("expected an error on 500")
	}
	if StatusCode(err) != http.StatusInternalServerError {
		t.Errorf("StatusCode = %d, want 500 (%v)", StatusCode(err), err)
	}
}

// An empty id is refused client-side, before the round trip.
func TestClearChannelRejectsEmptyID(t *testing.T) {
	srv, got := captureDelete(t, http.StatusNoContent, "")
	c := NewBotProviderClientWithConfig(testConfig(srv.URL))

	if err := c.ClearChannel(context.Background(), ""); err == nil {
		t.Fatal("expected an error for an empty customChannelID")
	}
	if *got != nil {
		t.Error("an empty id must not reach the server")
	}
}
