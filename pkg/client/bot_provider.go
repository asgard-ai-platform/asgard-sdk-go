package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strconv"
	"time"

	"go.asgard-ai.com/asgard-sdk-go/pkg/models"
)

const defaultHTTPTimeout = 300 * time.Second

// BotProviderClient defines the interface for interacting with Edge Server BotProvider APIs.
type BotProviderClient interface {
	NewStreamer(ctx context.Context, message *models.GenericBotMessage, opts *MessageRequestOptions) (BotProviderStreamer, error)
	NewChannelStreamer(ctx context.Context, customChannelID string, opts *ChannelStreamOptions) (BotProviderStreamer, error)
	SuspendChannel(ctx context.Context, customChannelID string, opts *SuspendOptions) error
	DeleteChannel(ctx context.Context, customChannelID string) error
	SendMessage(ctx context.Context, message *models.GenericBotMessage, opts *MessageRequestOptions) (*models.GenericBotReply, error)
	SendMessageFeedback(ctx context.Context, feedback *models.MessageFeedback, opts *FeedbackOptions) (*models.MessageFeedbackReply, error)
	Dispatch(ctx context.Context, message *models.GenericBotMessage, opts *MessageRequestOptions) (*models.GenericBotDispatchReply, error)
	TriggerJSON(ctx context.Context, payload map[string]interface{}) (interface{}, error)
	TriggerForm(ctx context.Context, payload map[string]interface{}, reader io.Reader, filename string, mime *string) (interface{}, error)
	UploadBlob(ctx context.Context, customChannelID string, reader io.Reader, filename string, mime *string) (*models.Blob, error)
	GenerateSandboxEditorOpenUrl(ctx context.Context, sandboxName string) (string, error)
	GenerateSandboxBrowserOpenUrl(ctx context.Context, sandboxName string) (string, error)
	SandboxFsList(ctx context.Context, sandboxName, path string) (*models.SandboxFsListResult, error)
	SandboxFsStat(ctx context.Context, sandboxName, path string) (*models.SandboxFsStatResult, error)
	SandboxFsRead(ctx context.Context, sandboxName, path string, offsetBytes, limitBytes *int64) ([]byte, *models.SandboxFsReadMeta, error)
	SandboxFsWrite(ctx context.Context, sandboxName, path string, reader io.Reader, filename string, mode *uint32, createOnly bool) (*models.SandboxFsWriteResult, error)
	SandboxFsMkdir(ctx context.Context, sandboxName, path string) error
	SandboxFsRemove(ctx context.Context, sandboxName, path string) error
	SandboxFsRemoveAll(ctx context.Context, sandboxName, path string) error
	SandboxFsCopy(ctx context.Context, sandboxName, src, dst string, overwrite bool) (*models.SandboxFsCopyResult, error)
	SandboxFsMove(ctx context.Context, sandboxName, src, dst string, overwrite bool) error
	SandboxFsWatch(ctx context.Context, sandboxName, path string, recursive bool) (io.ReadCloser, error)
	SandboxHeartbeat(ctx context.Context, sandboxName string) (*models.SandboxHeartbeatResult, error)
	DownloadChannelHomeFile(ctx context.Context, customChannelID, relativePath string) ([]byte, *models.ChannelHomeDownloadMeta, error)
	ChannelMetadata(ctx context.Context, customChannelID string) (*models.ChannelMetadata, error)
}

// BotProviderConfig holds the configuration for connecting to the bot provider.
type BotProviderConfig struct {
	HTTPClient        *http.Client
	EdgeServerHost    string
	Namespace         string
	BotProviderName   string
	BotProviderApiKey string
	Headers           map[string]string
}

type botProviderClient struct {
	config *BotProviderConfig
}

// NewBotProviderClient creates a BotProvider API client with default HTTP settings.
func NewBotProviderClient(edgeServerHost, namespace, botProviderName, botProviderAPIKey string) BotProviderClient {
	return NewBotProviderClientWithConfig(&BotProviderConfig{
		HTTPClient:        &http.Client{Timeout: defaultHTTPTimeout},
		EdgeServerHost:    edgeServerHost,
		Namespace:         namespace,
		BotProviderName:   botProviderName,
		BotProviderApiKey: botProviderAPIKey,
	})
}

// NewBotProviderClientWithConfig creates a BotProvider API client from config.
func NewBotProviderClientWithConfig(config *BotProviderConfig) BotProviderClient {
	if config == nil {
		config = &BotProviderConfig{}
	}

	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{Timeout: defaultHTTPTimeout}
	}

	return &botProviderClient{config: config}
}

type ApiResponse[T any] struct {
	IsSuccess bool    `json:"isSuccess"`
	Data      T       `json:"data"`
	Error     *string `json:"error"`
	ErrorCode *string `json:"errorCode"`
}

func (c *botProviderClient) NewStreamer(ctx context.Context, message *models.GenericBotMessage, opts *MessageRequestOptions) (BotProviderStreamer, error) {
	return NewStreaming(ctx, c.config, message, opts)
}

// NewChannelStreamer opens a GET /message/sse rejoin: it replays the channel's
// collapsed history then streams the in-flight turn until its terminal, without
// dispatching a message. Use it to (re)attach a viewer to a channel — e.g. a
// backend relay serving a browser's reconnect. Pass opts.LastEventID to resume
// from a known cursor; the stream auto-resumes transient drops in between.
func (c *botProviderClient) NewChannelStreamer(ctx context.Context, customChannelID string, opts *ChannelStreamOptions) (BotProviderStreamer, error) {
	return NewChannelStreaming(ctx, c.config, customChannelID, opts)
}

// SuspendChannel stops the channel's in-flight run. A run keeps going in the
// background after a streamer is closed — closing one only stops watching —
// so this is what actually stops the work.
//
// It returns as soon as the request is registered, NOT when the run has
// stopped. The run stops asynchronously and announces it the same way it
// announces finishing: with a terminal event on the channel's stream. A caller
// that needs to know the agent is idle should keep reading its streamer and
// wait for that terminal, rather than treating this return as the signal.
//
// The conversation is preserved: the transcript survives and the next message
// continues it. Because a stopped run never reaches its end, the turn is rolled
// back — the channel's context is left as the turn found it.
//
// Idempotent: stopping a run that already finished, or that a newer turn
// superseded, succeeds and does nothing.
func (c *botProviderClient) SuspendChannel(ctx context.Context, customChannelID string, opts *SuspendOptions) error {
	if customChannelID == "" {
		return fmt.Errorf("customChannelID cannot be empty")
	}
	q := url.Values{}
	q.Set("custom_channel_id", customChannelID)
	if opts != nil {
		if opts.RequestID != "" {
			q.Set("request_id", opts.RequestID)
		}
		if opts.Force {
			q.Set("force", "true")
		}
	}
	u := fmt.Sprintf("%s/ns/%s/bot-provider/%s/message/suspend?%s",
		c.config.EdgeServerHost,
		url.PathEscape(c.config.Namespace),
		url.PathEscape(c.config.BotProviderName),
		q.Encode(),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, http.NoBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("X-API-KEY", c.config.BotProviderApiKey)
	if opts != nil && opts.UserIdentityHint != "" {
		req.Header.Set("X-ASGARD-USER-IDENTITY-HINT", opts.UserIdentityHint)
	}

	resp, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("suspend channel failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return decodeAPIError(resp, "suspend channel")
	}
	return nil
}

// DeleteChannel ends a conversation and releases everything it holds: the
// in-flight run (its results are dropped), the transcript, every blob uploaded
// to the channel, the tool-call allow-list, and the channel's Sandbox with its
// Channel Home. The channel row itself is removed, so the customChannelID is
// free again — the next UploadBlob or message on it starts a fresh conversation
// with a new session. "Start over on the same id" is DeleteChannel followed by
// an ordinary action=NONE turn; that sequence (delete → upload → send) is also
// the only way to open a fresh conversation WITH attachments, since a
// RESET_CHANNEL turn deletes the blobs it names before dispatching and the
// server refuses that request (400).
//
// Unlike SuspendChannel this returns when the teardown has actually completed
// (a channel backing a live Sandbox waits for the pod to terminate, bounded by
// the server's teardown timeout), so a caller may start a new turn on the same
// id as soon as it returns without racing the dying pod.
//
// Idempotent: deleting a channel that never existed, or was already deleted,
// succeeds and does nothing.
func (c *botProviderClient) DeleteChannel(ctx context.Context, customChannelID string) error {
	if customChannelID == "" {
		return fmt.Errorf("customChannelID cannot be empty")
	}
	q := url.Values{}
	q.Set("custom_channel_id", customChannelID)
	u := fmt.Sprintf("%s/ns/%s/bot-provider/%s/channel?%s",
		c.config.EdgeServerHost,
		url.PathEscape(c.config.Namespace),
		url.PathEscape(c.config.BotProviderName),
		q.Encode(),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u, http.NoBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("X-API-KEY", c.config.BotProviderApiKey)

	resp, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("delete channel failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return decodeAPIError(resp, "delete channel")
	}
	return nil
}

func (c *botProviderClient) SendMessage(ctx context.Context, message *models.GenericBotMessage, opts *MessageRequestOptions) (*models.GenericBotReply, error) {
	if message == nil {
		return nil, fmt.Errorf("message cannot be nil")
	}

	if opts == nil {
		opts = &MessageRequestOptions{}
	}

	u := fmt.Sprintf("%s/ns/%s/bot-provider/%s/message",
		c.config.EdgeServerHost,
		url.PathEscape(c.config.Namespace),
		url.PathEscape(c.config.BotProviderName),
	)

	if opts.IsDebug {
		u = fmt.Sprintf("%s?is_debug=true", u)
	}

	body, err := json.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", c.config.BotProviderApiKey)
	if opts.UserIdentityHint != "" {
		req.Header.Set("X-ASGARD-USER-IDENTITY-HINT", opts.UserIdentityHint)
	}

	resp, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	reply, err := decodeAPIResponse[models.GenericBotReply](resp, "send message")
	if err != nil {
		return nil, err
	}
	return &reply, nil
}

// SendMessageFeedback records the user's Good/Bad verdict on one assistant
// reply — the thumbs up / thumbs down of a chat UI. The rated reply is named by
// feedback.MessageId (the messageId of a message-complete the client received
// live or on replay); the verdict is GOOD or BAD, with an optional free-text
// comment (at most 8 KiB — longer is a 400).
//
// The feedback becomes a first-class part of the conversation transcript: the
// server persists it, publishes it live to other viewers of the channel, and
// replays it when a client rejoins (as an asgard.message.feedback event) — so a
// reopened conversation still shows which replies were rated. It also lands in
// the platform's audit log for offline analysis. Append-only: rating the same
// reply again appends a newer entry and the latest wins; there is no un-rate.
//
// Rating a reply does not tell the AGENT anything by itself. When the user asks
// to share the feedback with the agent ("Send to AI as well"), follow this call
// with an ordinary message that opens with models.ResponseFeedbackPrefixGood or
// ...Bad — the platform's system prompt teaches the agent to treat that message
// as an interlude and then continue the conversation.
//
// Errors: an unknown channel, an unknown messageId, or a messageId that is not
// an assistant reply (a thinking block, the user's own message) map to
// IsNotFound; an invalid verdict or an oversized comment to IsBadRequest.
func (c *botProviderClient) SendMessageFeedback(ctx context.Context, feedback *models.MessageFeedback, opts *FeedbackOptions) (*models.MessageFeedbackReply, error) {
	if feedback == nil {
		return nil, fmt.Errorf("feedback cannot be nil")
	}
	if feedback.CustomChannelId == "" {
		return nil, fmt.Errorf("customChannelId cannot be empty")
	}
	if feedback.MessageId == "" {
		return nil, fmt.Errorf("messageId cannot be empty")
	}
	if feedback.Verdict != models.FeedbackVerdictGood && feedback.Verdict != models.FeedbackVerdictBad {
		return nil, fmt.Errorf("verdict %q must be %s or %s", feedback.Verdict, models.FeedbackVerdictGood, models.FeedbackVerdictBad)
	}
	if opts == nil {
		opts = &FeedbackOptions{}
	}

	u := fmt.Sprintf("%s/ns/%s/bot-provider/%s/message/feedback",
		c.config.EdgeServerHost,
		url.PathEscape(c.config.Namespace),
		url.PathEscape(c.config.BotProviderName),
	)

	body, err := json.Marshal(feedback)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal feedback: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", c.config.BotProviderApiKey)
	if opts.UserIdentityHint != "" {
		req.Header.Set("X-ASGARD-USER-IDENTITY-HINT", opts.UserIdentityHint)
	}

	resp, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send message feedback: %w", err)
	}
	defer resp.Body.Close()

	reply, err := decodeAPIResponse[models.MessageFeedbackReply](resp, "send message feedback")
	if err != nil {
		return nil, err
	}
	return &reply, nil
}

// Dispatch starts a run and returns as soon as the server has accepted it. The reply
// is a receipt — a requestId and the channel it landed on — not the run's output.
//
// Use this instead of SendMessage or NewStreamer when the caller has no use for the
// run's output on the wire: a scheduled trigger, a fire-and-forget integration. Those
// two make the caller hold a connection for the entire run, and holding it is pure
// cost once nothing is read from it. Worse, it is actively harmful: a connection that
// drops mid-run is indistinguishable from a failure, so a run that actually succeeded
// gets recorded as failed.
//
// A run is fully background and its transcript is the durable record, so nothing is
// lost by hanging up. Rejoin later with NewChannelStreamer, or read ChannelMetadata.
//
// The deadline is the caller's: pass a ctx with a timeout that bounds ACCEPTANCE
// (seconds), not the run. The client's own HTTP timeout defaults to 300s because it is
// sized for streaming, which is far too generous for this call.
func (c *botProviderClient) Dispatch(ctx context.Context, message *models.GenericBotMessage, opts *MessageRequestOptions) (*models.GenericBotDispatchReply, error) {
	if message == nil {
		return nil, fmt.Errorf("message cannot be nil")
	}

	if opts == nil {
		opts = &MessageRequestOptions{}
	}

	u := fmt.Sprintf("%s/ns/%s/bot-provider/%s/message/dispatch",
		c.config.EdgeServerHost,
		url.PathEscape(c.config.Namespace),
		url.PathEscape(c.config.BotProviderName),
	)

	query := url.Values{}
	if opts.IsDebug {
		query.Set("is_debug", "true")
	}
	if opts.BypassToolCallConsent {
		query.Set("bypass_tool_call_consent", "true")
	}
	if len(query) > 0 {
		u = u + "?" + query.Encode()
	}

	body, err := json.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", c.config.BotProviderApiKey)
	if opts.UserIdentityHint != "" {
		req.Header.Set("X-ASGARD-USER-IDENTITY-HINT", opts.UserIdentityHint)
	}

	resp, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to dispatch message: %w", err)
	}
	defer resp.Body.Close()

	reply, err := decodeAPIResponse[models.GenericBotDispatchReply](resp, "dispatch message")
	if err != nil {
		return nil, err
	}
	return &reply, nil
}

func (c *botProviderClient) TriggerJSON(ctx context.Context, payload map[string]interface{}) (interface{}, error) {
	u := fmt.Sprintf("%s/ns/%s/bot-provider/%s/json",
		c.config.EdgeServerHost,
		url.PathEscape(c.config.Namespace),
		url.PathEscape(c.config.BotProviderName),
	)

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal json payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", c.config.BotProviderApiKey)

	resp, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to trigger json api: %w", err)
	}
	defer resp.Body.Close()

	data, err := decodeAPIResponse[json.RawMessage](resp, "trigger json")
	if err != nil {
		return nil, err
	}

	if len(data) == 0 || string(data) == "null" {
		return nil, nil
	}

	var result interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response data: %w", err)
	}

	return result, nil
}

func (c *botProviderClient) TriggerForm(ctx context.Context, payload map[string]interface{}, reader io.Reader, filename string, mime *string) (interface{}, error) {
	u := fmt.Sprintf("%s/ns/%s/bot-provider/%s/form",
		c.config.EdgeServerHost,
		url.PathEscape(c.config.Namespace),
		url.PathEscape(c.config.BotProviderName),
	)

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal form json payload: %w", err)
	}

	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, pr)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-API-KEY", c.config.BotProviderApiKey)

	go func() {
		defer pw.Close()
		defer func() {
			if closeErr := writer.Close(); closeErr != nil {
				_ = pw.CloseWithError(fmt.Errorf("failed to close multipart writer: %w", closeErr))
			}
		}()

		if err := writer.WriteField("json", string(jsonPayload)); err != nil {
			_ = pw.CloseWithError(fmt.Errorf("failed to write json form field: %w", err))
			return
		}

		if reader == nil {
			return
		}

		header := make(textproto.MIMEHeader)
		header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename))
		if mime != nil && *mime != "" {
			header.Set("Content-Type", *mime)
		} else {
			header.Set("Content-Type", "application/octet-stream")
		}

		part, err := writer.CreatePart(header)
		if err != nil {
			_ = pw.CloseWithError(fmt.Errorf("failed to create multipart part: %w", err))
			return
		}

		if _, err := io.Copy(part, reader); err != nil {
			_ = pw.CloseWithError(fmt.Errorf("failed to copy file data: %w", err))
			return
		}
	}()

	resp, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to trigger form api: %w", err)
	}
	defer resp.Body.Close()

	data, err := decodeAPIResponse[json.RawMessage](resp, "trigger form")
	if err != nil {
		return nil, err
	}

	if len(data) == 0 || string(data) == "null" {
		return nil, nil
	}

	var result interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response data: %w", err)
	}

	return result, nil
}

func (c *botProviderClient) UploadBlob(ctx context.Context, customChannelID string, reader io.Reader, filename string, mime *string) (*models.Blob, error) {
	u := fmt.Sprintf("%s/ns/%s/bot-provider/%s/blob",
		c.config.EdgeServerHost,
		url.PathEscape(c.config.Namespace),
		url.PathEscape(c.config.BotProviderName),
	)

	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, pr)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-API-KEY", c.config.BotProviderApiKey)

	go func() {
		defer pw.Close()
		defer func() {
			if closeErr := writer.Close(); closeErr != nil {
				_ = pw.CloseWithError(fmt.Errorf("failed to close multipart writer: %w", closeErr))
			}
		}()

		if err := writer.WriteField("customChannelId", customChannelID); err != nil {
			_ = pw.CloseWithError(fmt.Errorf("failed to write customChannelId: %w", err))
			return
		}

		header := make(textproto.MIMEHeader)
		header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename))
		if mime != nil && *mime != "" {
			header.Set("Content-Type", *mime)
		} else {
			header.Set("Content-Type", "application/octet-stream")
		}

		part, err := writer.CreatePart(header)
		if err != nil {
			_ = pw.CloseWithError(fmt.Errorf("failed to create multipart part: %w", err))
			return
		}

		if _, err := io.Copy(part, reader); err != nil {
			_ = pw.CloseWithError(fmt.Errorf("failed to copy file data: %w", err))
			return
		}
	}()

	resp, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to upload blob: %w", err)
	}
	defer resp.Body.Close()

	blobs, err := decodeAPIResponse[[]models.Blob](resp, "upload blob")
	if err != nil {
		return nil, err
	}

	if len(blobs) == 0 {
		return nil, fmt.Errorf("upload blob succeeded but no blob metadata returned")
	}

	return &blobs[0], nil
}

func (c *botProviderClient) GenerateSandboxEditorOpenUrl(ctx context.Context, sandboxName string) (string, error) {
	u := fmt.Sprintf("%s/ns/%s/bot-provider/%s/sandbox/%s/editor/open-url",
		c.config.EdgeServerHost,
		url.PathEscape(c.config.Namespace),
		url.PathEscape(c.config.BotProviderName),
		url.PathEscape(sandboxName),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, http.NoBody)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", c.config.BotProviderApiKey)

	resp, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to generate sandbox editor open URL: %w", err)
	}
	defer resp.Body.Close()

	data, err := decodeAPIResponse[map[string]string](resp, "generate sandbox editor open url")
	if err != nil {
		return "", err
	}

	openURL, ok := data["openURL"]
	if !ok {
		return "", fmt.Errorf("response missing openURL field")
	}

	return openURL, nil
}

// GenerateSandboxBrowserOpenUrl mints a one-time URL to take over the sandbox's
// browser (Neko). Open the returned URL (e.g. in a new tab) to hand the human
// into the live browser session — for 2FA, sign-in, or captcha. The URL is
// single-use and short-lived; fetch a fresh one each time.
func (c *botProviderClient) GenerateSandboxBrowserOpenUrl(ctx context.Context, sandboxName string) (string, error) {
	u := fmt.Sprintf("%s/ns/%s/bot-provider/%s/sandbox/%s/browser/open-url",
		c.config.EdgeServerHost,
		url.PathEscape(c.config.Namespace),
		url.PathEscape(c.config.BotProviderName),
		url.PathEscape(sandboxName),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, http.NoBody)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", c.config.BotProviderApiKey)

	resp, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to generate sandbox browser open URL: %w", err)
	}
	defer resp.Body.Close()

	data, err := decodeAPIResponse[map[string]string](resp, "generate sandbox browser open url")
	if err != nil {
		return "", err
	}

	openURL, ok := data["openURL"]
	if !ok {
		return "", fmt.Errorf("response missing openURL field")
	}

	return openURL, nil
}

func (c *botProviderClient) SandboxFsList(ctx context.Context, sandboxName, path string) (*models.SandboxFsListResult, error) {
	u, err := url.Parse(fmt.Sprintf("%s/ns/%s/bot-provider/%s/sandbox/%s/fs/list",
		c.config.EdgeServerHost,
		url.PathEscape(c.config.Namespace),
		url.PathEscape(c.config.BotProviderName),
		url.PathEscape(sandboxName),
	))
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}
	q := u.Query()
	q.Set("path", path)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("X-API-KEY", c.config.BotProviderApiKey)

	resp, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sandbox fs list failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := decodeAPIResponse[models.SandboxFsListResult](resp, "sandbox fs list")
	if err != nil {
		return nil, err
	}
	return &data, nil
}

// sandboxFsRequest issues a request to a sandbox fs sub-path (e.g. "stat",
// "mkdir") with the given query params and returns the raw response for the
// caller to decode. The caller must Close the response body.
func (c *botProviderClient) sandboxFsRequest(ctx context.Context, method, sandboxName, subPath string, query url.Values) (*http.Response, error) {
	u, err := url.Parse(fmt.Sprintf("%s/ns/%s/bot-provider/%s/sandbox/%s/fs/%s",
		c.config.EdgeServerHost,
		url.PathEscape(c.config.Namespace),
		url.PathEscape(c.config.BotProviderName),
		url.PathEscape(sandboxName),
		subPath,
	))
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}
	u.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, method, u.String(), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("X-API-KEY", c.config.BotProviderApiKey)
	return c.config.HTTPClient.Do(req)
}

func (c *botProviderClient) SandboxFsStat(ctx context.Context, sandboxName, path string) (*models.SandboxFsStatResult, error) {
	q := url.Values{}
	q.Set("path", path)
	resp, err := c.sandboxFsRequest(ctx, http.MethodGet, sandboxName, "stat", q)
	if err != nil {
		return nil, fmt.Errorf("sandbox fs stat failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := decodeAPIResponse[models.SandboxFsStatResult](resp, "sandbox fs stat")
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (c *botProviderClient) SandboxFsMkdir(ctx context.Context, sandboxName, path string) error {
	q := url.Values{}
	q.Set("path", path)
	resp, err := c.sandboxFsRequest(ctx, http.MethodPost, sandboxName, "mkdir", q)
	if err != nil {
		return fmt.Errorf("sandbox fs mkdir failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return decodeAPIError(resp, "sandbox fs mkdir")
	}
	return nil
}

func (c *botProviderClient) SandboxFsRemove(ctx context.Context, sandboxName, path string) error {
	q := url.Values{}
	q.Set("path", path)
	resp, err := c.sandboxFsRequest(ctx, http.MethodDelete, sandboxName, "item", q)
	if err != nil {
		return fmt.Errorf("sandbox fs remove failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return decodeAPIError(resp, "sandbox fs remove")
	}
	return nil
}

func (c *botProviderClient) SandboxFsRemoveAll(ctx context.Context, sandboxName, path string) error {
	q := url.Values{}
	q.Set("path", path)
	resp, err := c.sandboxFsRequest(ctx, http.MethodDelete, sandboxName, "all", q)
	if err != nil {
		return fmt.Errorf("sandbox fs remove all failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return decodeAPIError(resp, "sandbox fs remove all")
	}
	return nil
}

func (c *botProviderClient) SandboxFsCopy(ctx context.Context, sandboxName, src, dst string, overwrite bool) (*models.SandboxFsCopyResult, error) {
	q := url.Values{}
	q.Set("src", src)
	q.Set("dst", dst)
	if overwrite {
		q.Set("overwrite", "true")
	}
	resp, err := c.sandboxFsRequest(ctx, http.MethodPost, sandboxName, "copy", q)
	if err != nil {
		return nil, fmt.Errorf("sandbox fs copy failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := decodeAPIResponse[models.SandboxFsCopyResult](resp, "sandbox fs copy")
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (c *botProviderClient) SandboxFsMove(ctx context.Context, sandboxName, src, dst string, overwrite bool) error {
	q := url.Values{}
	q.Set("src", src)
	q.Set("dst", dst)
	if overwrite {
		q.Set("overwrite", "true")
	}
	resp, err := c.sandboxFsRequest(ctx, http.MethodPost, sandboxName, "move", q)
	if err != nil {
		return fmt.Errorf("sandbox fs move failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return decodeAPIError(resp, "sandbox fs move")
	}
	return nil
}

// SandboxFsWatch opens the sandbox fs/watch SSE stream
// (GET .../sandbox/{name}/fs/watch?path=&recursive=) and returns the raw
// text/event-stream response body for the caller to relay or parse. Each frame
// is an "event: change" carrying a JSON models.SandboxFsWatchEvent payload. The
// caller MUST Close the returned stream.
//
// A non-2xx response (e.g. an *APIError with 404 for a missing path) is returned
// before any stream begins, so a relay can surface the right status ahead of the
// event-stream. This is deliberately a thin passthrough — a watch has no durable
// resume cursor, unlike the message streamer. Because the stream is long-lived,
// configure the client's HTTPClient without a read timeout.
func (c *botProviderClient) SandboxFsWatch(ctx context.Context, sandboxName, path string, recursive bool) (io.ReadCloser, error) {
	u, err := url.Parse(fmt.Sprintf("%s/ns/%s/bot-provider/%s/sandbox/%s/fs/watch",
		c.config.EdgeServerHost,
		url.PathEscape(c.config.Namespace),
		url.PathEscape(c.config.BotProviderName),
		url.PathEscape(sandboxName),
	))
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}
	q := u.Query()
	q.Set("path", path)
	if recursive {
		q.Set("recursive", "true")
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("X-API-KEY", c.config.BotProviderApiKey)

	resp, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sandbox fs watch failed: %w", err)
	}
	if resp.StatusCode/100 != 2 {
		defer resp.Body.Close()
		return nil, decodeAPIError(resp, "sandbox fs watch")
	}
	return resp.Body, nil
}

// ChannelMetadata fetches a channel's metadata — its conversation title, run
// state, and last activity time — so a client entering a chat room can restore
// its UI (e.g. the room title) without opening the stream. Returns an *APIError
// (check with IsNotFound) when the channel does not exist yet.
func (c *botProviderClient) ChannelMetadata(ctx context.Context, customChannelID string) (*models.ChannelMetadata, error) {
	u, err := url.Parse(fmt.Sprintf("%s/ns/%s/bot-provider/%s/channel/metadata",
		c.config.EdgeServerHost,
		url.PathEscape(c.config.Namespace),
		url.PathEscape(c.config.BotProviderName),
	))
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}
	q := u.Query()
	q.Set("custom_channel_id", customChannelID)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("X-API-KEY", c.config.BotProviderApiKey)

	resp, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("channel metadata failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := decodeAPIResponse[models.ChannelMetadata](resp, "channel metadata")
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (c *botProviderClient) SandboxFsRead(ctx context.Context, sandboxName, path string, offsetBytes, limitBytes *int64) ([]byte, *models.SandboxFsReadMeta, error) {
	u, err := url.Parse(fmt.Sprintf("%s/ns/%s/bot-provider/%s/sandbox/%s/fs/file",
		c.config.EdgeServerHost,
		url.PathEscape(c.config.Namespace),
		url.PathEscape(c.config.BotProviderName),
		url.PathEscape(sandboxName),
	))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse URL: %w", err)
	}
	q := u.Query()
	q.Set("path", path)
	if offsetBytes != nil {
		q.Set("offset_bytes", strconv.FormatInt(*offsetBytes, 10))
	}
	if limitBytes != nil {
		q.Set("limit_bytes", strconv.FormatInt(*limitBytes, 10))
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), http.NoBody)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("X-API-KEY", c.config.BotProviderApiKey)

	resp, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("sandbox fs read failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return nil, nil, decodeAPIError(resp, "sandbox fs read")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response body: %w", err)
	}

	meta := &models.SandboxFsReadMeta{}
	if v := resp.Header.Get("X-Total-Bytes"); v != "" {
		meta.TotalBytes, _ = strconv.ParseInt(v, 10, 64)
	}
	meta.Truncated = resp.Header.Get("X-Truncated") == "true"

	return body, meta, nil
}

func (c *botProviderClient) DownloadChannelHomeFile(ctx context.Context, customChannelID, relativePath string) ([]byte, *models.ChannelHomeDownloadMeta, error) {
	u, err := url.Parse(fmt.Sprintf("%s/ns/%s/bot-provider/%s/channel-home/download",
		c.config.EdgeServerHost,
		url.PathEscape(c.config.Namespace),
		url.PathEscape(c.config.BotProviderName),
	))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse URL: %w", err)
	}
	q := u.Query()
	q.Set("custom_channel_id", customChannelID)
	q.Set("relative_path", relativePath)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), http.NoBody)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("X-API-KEY", c.config.BotProviderApiKey)

	resp, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("channel home download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return nil, nil, decodeAPIError(resp, "channel home download")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response body: %w", err)
	}

	meta := &models.ChannelHomeDownloadMeta{MimeType: resp.Header.Get("Content-Type")}
	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		if _, params, perr := mime.ParseMediaType(cd); perr == nil {
			meta.FileName = params["filename"]
		}
	}

	return body, meta, nil
}

func (c *botProviderClient) SandboxFsWrite(ctx context.Context, sandboxName, path string, reader io.Reader, filename string, mode *uint32, createOnly bool) (*models.SandboxFsWriteResult, error) {
	u, err := url.Parse(fmt.Sprintf("%s/ns/%s/bot-provider/%s/sandbox/%s/fs/file",
		c.config.EdgeServerHost,
		url.PathEscape(c.config.Namespace),
		url.PathEscape(c.config.BotProviderName),
		url.PathEscape(sandboxName),
	))
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}
	q := u.Query()
	q.Set("path", path)
	if mode != nil {
		q.Set("mode", strconv.FormatUint(uint64(*mode), 10))
	}
	if createOnly {
		q.Set("create_only", "true")
	}
	u.RawQuery = q.Encode()

	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, u.String(), pr)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-API-KEY", c.config.BotProviderApiKey)

	go func() {
		defer pw.Close()
		defer func() {
			if closeErr := writer.Close(); closeErr != nil {
				_ = pw.CloseWithError(fmt.Errorf("failed to close multipart writer: %w", closeErr))
			}
		}()

		header := make(textproto.MIMEHeader)
		header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename))
		header.Set("Content-Type", "application/octet-stream")

		part, err := writer.CreatePart(header)
		if err != nil {
			_ = pw.CloseWithError(fmt.Errorf("failed to create multipart part: %w", err))
			return
		}
		if _, err := io.Copy(part, reader); err != nil {
			_ = pw.CloseWithError(fmt.Errorf("failed to copy file data: %w", err))
		}
	}()

	resp, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sandbox fs write failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := decodeAPIResponse[models.SandboxFsWriteResult](resp, "sandbox fs write")
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (c *botProviderClient) SandboxHeartbeat(ctx context.Context, sandboxName string) (*models.SandboxHeartbeatResult, error) {
	u := fmt.Sprintf("%s/ns/%s/bot-provider/%s/sandbox/%s/heartbeat",
		c.config.EdgeServerHost,
		url.PathEscape(c.config.Namespace),
		url.PathEscape(c.config.BotProviderName),
		url.PathEscape(sandboxName),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("X-API-KEY", c.config.BotProviderApiKey)

	resp, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sandbox heartbeat failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := decodeAPIResponse[models.SandboxHeartbeatResult](resp, "sandbox heartbeat")
	if err != nil {
		return nil, err
	}
	return &data, nil
}
