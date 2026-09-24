package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// headerRecorder stands in for edgeserver and records the Attribution header of
// every request, answering whatever shape the calling method expects.
type headerRecorder struct {
	mu   sync.Mutex
	seen []string
	set  []bool
}

func (h *headerRecorder) server(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.mu.Lock()
		_, ok := r.Header[http.CanonicalHeaderKey(attributionHeader)]
		h.seen = append(h.seen, r.Header.Get(attributionHeader))
		h.set = append(h.set, ok)
		h.mu.Unlock()
		switch {
		case r.URL.Path == "/ns/ns/bot-provider/bp/message/sse":
			w.Header().Set("Content-Type", "text/event-stream")
			et, ev := doneEvent()
			writeFrame(w, "1", et, ev)
		case r.URL.Path == "/ns/ns/bot-provider/bp/message/dispatch":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"isSuccess":true,"data":{"requestId":"r","customChannelId":"c"}}`))
		default:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"isSuccess":true,"data":{"messages":[]}}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func (h *headerRecorder) last(t *testing.T) (string, bool) {
	t.Helper()
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.seen) == 0 {
		t.Fatal("no request reached the server")
	}
	return h.seen[len(h.seen)-1], h.set[len(h.set)-1]
}

// Every call that starts a run forwards the grant verbatim, and sends no header
// at all without one.
func TestAttributionForwardedOnEveryRunStartingCall(t *testing.T) {
	calls := map[string]func(c BotProviderClient, cfg *BotProviderConfig, opts *MessageRequestOptions) error{
		"SendMessage": func(c BotProviderClient, _ *BotProviderConfig, opts *MessageRequestOptions) error {
			_, err := c.SendMessage(context.Background(), testMessage(), opts)
			return err
		},
		"Dispatch": func(c BotProviderClient, _ *BotProviderConfig, opts *MessageRequestOptions) error {
			_, err := c.Dispatch(context.Background(), testMessage(), opts)
			return err
		},
		"NewStreaming": func(_ BotProviderClient, cfg *BotProviderConfig, opts *MessageRequestOptions) error {
			s, err := NewStreaming(context.Background(), cfg, testMessage(), opts)
			if err != nil {
				return err
			}
			defer s.Close()
			drain(s)
			return s.Err()
		},
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			rec := &headerRecorder{}
			cfg := testConfig(rec.server(t).URL)
			c := NewBotProviderClientWithConfig(cfg)

			if err := call(c, cfg, &MessageRequestOptions{Attribution: "grant.token.sig"}); err != nil {
				t.Fatalf("with grant: %v", err)
			}
			if got, ok := rec.last(t); !ok || got != "grant.token.sig" {
				t.Errorf("with grant: header = %q (sent=%v), want forwarded verbatim", got, ok)
			}

			if err := call(c, cfg, nil); err != nil {
				t.Fatalf("without grant: %v", err)
			}
			if got, ok := rec.last(t); ok {
				t.Errorf("without grant: header sent as %q, want absent", got)
			}
		})
	}
}
