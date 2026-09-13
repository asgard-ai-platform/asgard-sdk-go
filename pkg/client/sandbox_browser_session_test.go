package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// What belongs to the SDK here is narrow but easy to get wrong silently: hit the right
// path with the right method and key, and refuse a response that is shaped like success
// but carries nothing usable. A half-filled session surfaces much later, as a WebSocket
// that will not connect.
func TestCreateSandboxBrowserSession(t *testing.T) {
	var gotPath, gotMethod, gotKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod, gotKey = r.URL.EscapedPath(), r.Method, r.Header.Get("X-API-KEY")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"isSuccess":true,"data":{"wsUrl":"wss://lease.example.com/api/ws","token":"tok-123"}}`))
	}))
	defer srv.Close()

	c := NewBotProviderClientWithConfig(testConfig(srv.URL))
	got, err := c.CreateSandboxBrowserSession(context.Background(), "sbx 1")
	if err != nil {
		t.Fatalf("CreateSandboxBrowserSession returned error: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	// The sandbox name is path-escaped: a name with a space must not split the path.
	if want := "/ns/ns/bot-provider/bp/sandbox/sbx%201/browser/session"; gotPath != want {
		t.Errorf("path = %q, want %q", gotPath, want)
	}
	if gotKey != "key" {
		t.Errorf("X-API-KEY = %q, want %q", gotKey, "key")
	}
	if got.WsUrl != "wss://lease.example.com/api/ws" || got.Token != "tok-123" {
		t.Errorf("session = %+v, want the wsUrl/token from the response body", got)
	}
}

// A 200 whose data is missing either field must be an error, not a zero-valued session.
func TestCreateSandboxBrowserSessionRejectsIncompleteData(t *testing.T) {
	for _, body := range []string{
		`{"isSuccess":true,"data":{"wsUrl":"wss://lease.example.com/api/ws"}}`,
		`{"isSuccess":true,"data":{"token":"tok-123"}}`,
		`{"isSuccess":true,"data":{}}`,
	} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(body))
		}))

		c := NewBotProviderClientWithConfig(testConfig(srv.URL))
		if got, err := c.CreateSandboxBrowserSession(context.Background(), "sbx-1"); err == nil {
			t.Errorf("body %s: got %+v, want error", body, got)
		}
		srv.Close()
	}
}

// 412 is what EdgeServer returns when the sandbox does not exist yet — the caller has to
// see that rather than a nil session with a nil error.
func TestCreateSandboxBrowserSessionPropagatesHttpError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusPreconditionFailed)
		_, _ = w.Write([]byte(`{"isSuccess":false,"error":"no matched sandbox found"}`))
	}))
	defer srv.Close()

	c := NewBotProviderClientWithConfig(testConfig(srv.URL))
	if got, err := c.CreateSandboxBrowserSession(context.Background(), "sbx-1"); err == nil {
		t.Errorf("got %+v, want error for a 412 response", got)
	}
}
