package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.asgard-ai.com/asgard-sdk-go/pkg/models"
)

// captureDispatch stands in for edgeserver, recording what the client sent and
// answering with the 202 receipt the real endpoint returns.
func captureDispatch(t *testing.T, status int, body string) (*httptest.Server, *http.Request, *[]byte) {
	t.Helper()
	var gotReq *http.Request
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotReq = r.Clone(r.Context())
		gotBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv, gotReq, &gotBody
}

// The whole point of Dispatch is the receipt: a caller that walks away still learns
// which run it started and where it landed.
func TestDispatchReturnsTheReceipt(t *testing.T) {
	srv, _, gotBody := captureDispatch(t, http.StatusAccepted,
		`{"isSuccess":true,"data":{"requestId":"req-9","customChannelId":"inv-42"}}`)

	c := NewBotProviderClientWithConfig(testConfig(srv.URL))
	invocationID := "inv-42"
	reply, err := c.Dispatch(context.Background(), &models.GenericBotMessage{
		CustomChannelId: "inv-42",
		CustomMessageId: "inv-42",
		Text:            "check yesterday's failed orders",
		Action:          models.PostBackActionNone,
		InvocationId:    &invocationID,
	}, nil)
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if reply.RequestId != "req-9" || reply.CustomChannelId != "inv-42" {
		t.Errorf("receipt = %+v, want req-9 / inv-42", reply)
	}

	// invocationId has to survive marshalling, or the run's terminal has nothing to
	// settle and the trigger side is back to watching its own run to the end.
	var sent map[string]any
	if err := json.Unmarshal(*gotBody, &sent); err != nil {
		t.Fatalf("unmarshal sent body: %v", err)
	}
	if sent["invocationId"] != "inv-42" {
		t.Errorf("invocationId = %v, want inv-42 (body: %s)", sent["invocationId"], *gotBody)
	}
	if sent["text"] != "check yesterday's failed orders" {
		t.Errorf("text = %v, want the message (body: %s)", sent["text"], *gotBody)
	}
}

// 202 is the success status here, not 200 — a decoder that only accepts 200 would
// turn every successful dispatch into an error.
func TestDispatchAcceptsStatus202(t *testing.T) {
	srv, _, _ := captureDispatch(t, http.StatusAccepted,
		`{"isSuccess":true,"data":{"requestId":"r","customChannelId":"c"}}`)

	c := NewBotProviderClientWithConfig(testConfig(srv.URL))
	if _, err := c.Dispatch(context.Background(), testMessage(), nil); err != nil {
		t.Fatalf("202 was rejected: %v", err)
	}
}

// A dispatch that was not accepted must be an error, because the caller uses exactly
// this to decide whether any run started at all.
func TestDispatchSurfacesRejection(t *testing.T) {
	srv, _, _ := captureDispatch(t, http.StatusBadRequest,
		`{"isSuccess":false,"error":"channel is awaiting a tool-call consent response"}`)

	c := NewBotProviderClientWithConfig(testConfig(srv.URL))
	if _, err := c.Dispatch(context.Background(), testMessage(), nil); err == nil {
		t.Fatal("a rejected dispatch reported success")
	}
}

func TestDispatchRejectsNilMessage(t *testing.T) {
	c := NewBotProviderClient("http://unused", "ns", "bp", "key")
	if _, err := c.Dispatch(context.Background(), nil, nil); err == nil {
		t.Fatal("nil message was accepted")
	}
}
