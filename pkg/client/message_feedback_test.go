package client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.asgard-ai.com/asgard-sdk-go/pkg/models"
)

// captureFeedback stands in for edgeserver's POST /message/feedback: it records
// the request (headers + body) and answers with the given status and body.
func captureFeedback(t *testing.T, status int, body string) (*httptest.Server, **http.Request, *string) {
	t.Helper()
	var got *http.Request
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Clone(r.Context())
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv, &got, &gotBody
}

// SendMessageFeedback is a POST on the message/feedback sibling of /message,
// authenticated like every other call, carrying the verdict JSON and the
// optional identity hint header; the reply is the persisted entry's id + seq.
func TestSendMessageFeedbackSendsVerdict(t *testing.T) {
	srv, got, gotBody := captureFeedback(t, http.StatusOK,
		`{"isSuccess":true,"data":{"messageId":"fb1","seq":42}}`)
	c := NewBotProviderClientWithConfig(testConfig(srv.URL))

	reply, err := c.SendMessageFeedback(context.Background(), &models.MessageFeedback{
		CustomChannelId: "room-1",
		MessageId:       "m1",
		Verdict:         models.FeedbackVerdictGood,
		Comment:         "nice",
	}, &FeedbackOptions{UserIdentityHint: "usr-1"})
	if err != nil {
		t.Fatalf("SendMessageFeedback: %v", err)
	}
	r := *got
	if r == nil {
		t.Fatal("no request reached the server")
	}
	if r.Method != http.MethodPost || r.URL.Path != "/ns/ns/bot-provider/bp/message/feedback" {
		t.Fatalf("request: %s %s", r.Method, r.URL)
	}
	if r.Header.Get("X-API-KEY") != "key" {
		t.Fatalf("api key header: %q", r.Header.Get("X-API-KEY"))
	}
	if r.Header.Get("X-ASGARD-USER-IDENTITY-HINT") != "usr-1" {
		t.Fatalf("identity hint header: %q", r.Header.Get("X-ASGARD-USER-IDENTITY-HINT"))
	}
	for _, want := range []string{`"customChannelId":"room-1"`, `"messageId":"m1"`, `"verdict":"GOOD"`, `"comment":"nice"`} {
		if !strings.Contains(*gotBody, want) {
			t.Fatalf("body lacks %s: %s", want, *gotBody)
		}
	}
	if reply.MessageId != "fb1" || reply.Seq != 42 {
		t.Fatalf("reply: %+v", reply)
	}
}

// The server's status survives to the typed error helpers — a 404 (unknown or
// non-ratable message) must be recognizable as IsNotFound so a relay can map it
// back rather than reporting a server fault.
func TestSendMessageFeedbackMapsNotFound(t *testing.T) {
	srv, _, _ := captureFeedback(t, http.StatusNotFound,
		`{"isSuccess":false,"error":"message m1 not found","errorCode":"MESSAGE_NOT_FOUND"}`)
	c := NewBotProviderClientWithConfig(testConfig(srv.URL))

	_, err := c.SendMessageFeedback(context.Background(), &models.MessageFeedback{
		CustomChannelId: "room-1", MessageId: "m1", Verdict: models.FeedbackVerdictBad,
	}, nil)
	if !IsNotFound(err) {
		t.Fatalf("want IsNotFound, got %v", err)
	}
}

// Required fields and the verdict enum are guarded client-side, before any
// round trip.
func TestSendMessageFeedbackGuardsBeforeRoundTrip(t *testing.T) {
	srv, got, _ := captureFeedback(t, http.StatusOK, `{"isSuccess":true,"data":{}}`)
	c := NewBotProviderClientWithConfig(testConfig(srv.URL))

	cases := []*models.MessageFeedback{
		nil,
		{MessageId: "m1", Verdict: models.FeedbackVerdictGood},                              // no channel
		{CustomChannelId: "room-1", Verdict: models.FeedbackVerdictGood},                    // no message
		{CustomChannelId: "room-1", MessageId: "m1"},                                        // no verdict
		{CustomChannelId: "room-1", MessageId: "m1", Verdict: models.FeedbackVerdict("ok")}, // bad verdict
	}
	for i, fb := range cases {
		if _, err := c.SendMessageFeedback(context.Background(), fb, nil); err == nil {
			t.Errorf("case %d: want client-side error, got nil", i)
		}
	}
	if *got != nil {
		t.Fatalf("a guarded request still reached the server: %v", (*got).URL)
	}
}
