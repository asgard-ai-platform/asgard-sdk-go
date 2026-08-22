package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.asgard-ai.com/asgard-sdk-go/pkg/models"
)

// rawMessageUserWithBlobs is a verbatim asgard.message.user frame as EdgeServer
// emits it on a history rejoin, for an ATTACHMENT-ONLY turn (text is empty).
// Copied from a live stream rather than produced by marshalling our own structs:
// a round-trip through the same types cannot catch the failure that matters here,
// which is our field names disagreeing with the platform's.
//
// Only the fact's own fields are kept verbatim; the sibling null facts in the real
// frame are elided as noise.
const rawMessageUserWithBlobs = `{"eventType":"asgard.message.user","requestId":"","eventId":"2091078163256840192","namespace":"ns","botProviderName":"bp","customChannelId":"c1","fact":{"messageUser":{"messageId":"2091078161394569216","text":"","identityHint":"primary","customMessageId":"m-19475","blobIds":["2091078155488989184","2091078159192559616"],"blobs":[{"blobId":"2091078155488989184","fileType":"DOCUMENT","fileName":"attach-doc-2677226055.txt","size":39,"mime":"text/plain"},{"blobId":"2091078159192559616","fileType":"IMAGE","fileName":null,"size":70,"mime":"image/png"}]}}}`

func rawFrame(id, eventType, data string) string {
	return "id: " + id + "\nevent: " + eventType + "\ndata: " + data + "\n\n"
}

// A real frame's attachment metadata must survive the streamer, and must survive
// being re-encoded by a relay. Both halves matter: asgard-agent-hub-api's rejoin
// relay hands Current() straight back to gin, so anything this type drops or
// invents is what the browser ends up seeing.
func TestChannelStreamerDecodesMessageUserBlobs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, rawFrame("1", string(models.SseEventTypeMessageUser), rawMessageUserWithBlobs))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		et, ev := doneEvent()
		writeFrame(w, "2", et, ev)
	}))
	defer srv.Close()

	s, err := NewChannelStreaming(context.Background(), testConfig(srv.URL), "c1", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	var mu *models.GenericBotSseEventFactMessageUser
	for s.Next() {
		if s.Current().EventType == models.SseEventTypeMessageUser {
			mu = s.Current().Fact.MessageUser
		}
	}
	if err := s.Err(); err != nil {
		t.Fatalf("stream: %v", err)
	}
	if mu == nil {
		t.Fatal("messageUser fact never surfaced")
	}

	// The ids were never the missing half; the metadata beside them was.
	if len(mu.BlobIds) != 2 {
		t.Fatalf("blobIds: %+v", mu.BlobIds)
	}
	if len(mu.Blobs) != 2 {
		t.Fatalf("blobs did not decode: %+v", mu.Blobs)
	}

	doc := mu.Blobs[0]
	if doc.BlobId != "2091078155488989184" || doc.FileType != models.FileTypeDocument || doc.Mime != "text/plain" || doc.Size != 39 {
		t.Fatalf("document blob wrong: %+v", doc)
	}
	// The file name is what labels the chip — its absence was the reported symptom.
	if doc.FileName == nil || *doc.FileName != "attach-doc-2677226055.txt" {
		t.Fatalf("document fileName wrong: %+v", doc.FileName)
	}

	img := mu.Blobs[1]
	if img.FileType != models.FileTypeImage || img.Mime != "image/png" {
		t.Fatalf("image blob wrong: %+v", img)
	}
	// A null name must stay nil, not collapse to "": a client can only substitute
	// its own label if it can tell the difference.
	if img.FileName != nil {
		t.Fatalf("image fileName should stay nil, got %q", *img.FileName)
	}

	// The relay half. A decode-then-encode must not lose blobs, and must not
	// invent a channelId — models.Blob carries one, this shape does not, and
	// reusing it here would have put a field on the wire that was never sent.
	out, err := json.Marshal(mu)
	if err != nil {
		t.Fatalf("re-encode: %v", err)
	}
	if !strings.Contains(string(out), `"blobs"`) {
		t.Fatalf("blobs lost on re-encode: %s", out)
	}
	if strings.Contains(string(out), "channelId") {
		t.Fatalf("re-encode invented a channelId: %s", out)
	}
	if !strings.Contains(string(out), `"fileName":null`) {
		t.Fatalf("re-encode lost the null fileName: %s", out)
	}
}

// A turn with no attachments must leave blobs off the wire entirely. A relay
// re-encoding an empty list into `"blobs":[]` would tell a client "this turn had
// attachments, none of them renderable" — which is what the old ids-only frame
// already said, and the fallback keys on absence to avoid saying it.
func TestChannelStreamerOmitsBlobsWhenNoAttachments(t *testing.T) {
	raw := `{"eventType":"asgard.message.user","fact":{"messageUser":{"messageId":"u9","text":"just words"}}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, rawFrame("1", string(models.SseEventTypeMessageUser), raw))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		et, ev := doneEvent()
		writeFrame(w, "2", et, ev)
	}))
	defer srv.Close()

	s, err := NewChannelStreaming(context.Background(), testConfig(srv.URL), "c1", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	var mu *models.GenericBotSseEventFactMessageUser
	for s.Next() {
		if s.Current().EventType == models.SseEventTypeMessageUser {
			mu = s.Current().Fact.MessageUser
		}
	}
	if mu == nil {
		t.Fatal("messageUser fact never surfaced")
	}
	if mu.Blobs != nil {
		t.Fatalf("blobs should stay nil on an attachment-less turn: %+v", mu.Blobs)
	}
	out, err := json.Marshal(mu)
	if err != nil {
		t.Fatalf("re-encode: %v", err)
	}
	if strings.Contains(string(out), "blobs") {
		t.Fatalf("re-encode added a blobs key: %s", out)
	}
}
