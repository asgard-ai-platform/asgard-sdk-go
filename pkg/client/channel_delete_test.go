package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// captureDelete stands in for edgeserver's DELETE /channel: it records the
// request and answers with the given status.
func captureDelete(t *testing.T, status int, body string) (*httptest.Server, **http.Request) {
	t.Helper()
	var got *http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Clone(r.Context())
		if body != "" {
			w.Header().Set("Content-Type", "application/json")
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv, &got
}

// DeleteChannel is a DELETE on the channel resource, addressed by the natural
// key (path = ns/bot, query = custom_channel_id) and authenticated like every
// other call; a 204 is success with nothing to decode.
func TestDeleteChannelSendsDeleteWithNaturalKey(t *testing.T) {
	srv, got := captureDelete(t, http.StatusNoContent, "")
	c := NewBotProviderClientWithConfig(testConfig(srv.URL))

	if err := c.DeleteChannel(context.Background(), "room 1"); err != nil {
		t.Fatalf("DeleteChannel: %v", err)
	}
	r := *got
	if r == nil {
		t.Fatal("no request reached the server")
	}
	if r.Method != http.MethodDelete {
		t.Errorf("method = %s, want DELETE", r.Method)
	}
	if r.URL.Path != "/ns/ns/bot-provider/bp/channel" {
		t.Errorf("path = %s, want /ns/ns/bot-provider/bp/channel", r.URL.Path)
	}
	if r.URL.Query().Get("custom_channel_id") != "room 1" {
		t.Errorf("custom_channel_id = %q, want \"room 1\" (must be query-encoded)", r.URL.Query().Get("custom_channel_id"))
	}
	if r.Header.Get("X-API-KEY") != "key" {
		t.Errorf("X-API-KEY = %q, want key", r.Header.Get("X-API-KEY"))
	}
}

// A non-2xx is surfaced as an *APIError so callers can branch on the status.
func TestDeleteChannelSurfacesAPIError(t *testing.T) {
	srv, _ := captureDelete(t, http.StatusInternalServerError,
		`{"isSuccess":false,"errorCode":"INTERNAL","message":"teardown failed"}`)
	c := NewBotProviderClientWithConfig(testConfig(srv.URL))

	err := c.DeleteChannel(context.Background(), "room-1")
	if err == nil {
		t.Fatal("expected an error on 500")
	}
	if StatusCode(err) != http.StatusInternalServerError {
		t.Errorf("StatusCode = %d, want 500 (%v)", StatusCode(err), err)
	}
}

// An empty id is refused client-side — the server would 400 it anyway, but a
// blank id is a programming error worth catching before the round trip.
func TestDeleteChannelRejectsEmptyID(t *testing.T) {
	srv, got := captureDelete(t, http.StatusNoContent, "")
	c := NewBotProviderClientWithConfig(testConfig(srv.URL))

	if err := c.DeleteChannel(context.Background(), ""); err == nil {
		t.Fatal("expected an error for an empty customChannelID")
	}
	if *got != nil {
		t.Error("an empty id must not reach the server")
	}
}
