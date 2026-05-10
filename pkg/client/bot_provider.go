package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strconv"

	"go.asgard-ai.com/asgard-sdk-go/pkg/models"
)

type ApiResponse[T any] struct {
	IsSuccess bool    `json:"isSuccess"`
	Data      T       `json:"data"`
	Error     *string `json:"error"`
	ErrorCode *string `json:"errorCode"`
}

func (c *BotProviderClient) NewStreamer(ctx context.Context, message *models.GenericBotMessage, opts *MessageRequestOptions) (BotProviderStreamer, error) {
	return NewStreaming(ctx, c.config, message, opts)
}

func (c *BotProviderClient) SendMessage(ctx context.Context, message *models.GenericBotMessage, opts *MessageRequestOptions) (*models.GenericBotReply, error) {
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

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var payload ApiResponse[models.GenericBotReply]
	if err := json.Unmarshal(respBytes, &payload); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if resp.StatusCode != http.StatusOK || !payload.IsSuccess {
		return nil, fmt.Errorf("send message failed (%d): %s", resp.StatusCode, responseError(payload.Error, payload.ErrorCode))
	}

	return &payload.Data, nil
}

func (c *BotProviderClient) TriggerJSON(ctx context.Context, payload map[string]interface{}) (interface{}, error) {
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

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var wrapper ApiResponse[json.RawMessage]
	if err := json.Unmarshal(respBytes, &wrapper); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if resp.StatusCode != http.StatusOK || !wrapper.IsSuccess {
		return nil, fmt.Errorf("trigger json failed (%d): %s", resp.StatusCode, responseError(wrapper.Error, wrapper.ErrorCode))
	}

	if len(wrapper.Data) == 0 || string(wrapper.Data) == "null" {
		return nil, nil
	}

	var result interface{}
	if err := json.Unmarshal(wrapper.Data, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response data: %w", err)
	}

	return result, nil
}

func (c *BotProviderClient) TriggerForm(ctx context.Context, payload map[string]interface{}, reader io.Reader, filename string, mime *string) (interface{}, error) {
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

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var wrapper ApiResponse[json.RawMessage]
	if err := json.Unmarshal(respBytes, &wrapper); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if resp.StatusCode != http.StatusOK || !wrapper.IsSuccess {
		return nil, fmt.Errorf("trigger form failed (%d): %s", resp.StatusCode, responseError(wrapper.Error, wrapper.ErrorCode))
	}

	if len(wrapper.Data) == 0 || string(wrapper.Data) == "null" {
		return nil, nil
	}

	var result interface{}
	if err := json.Unmarshal(wrapper.Data, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response data: %w", err)
	}

	return result, nil
}

func (c *BotProviderClient) UploadBlob(ctx context.Context, customChannelID string, reader io.Reader, filename string, mime *string) (*models.Blob, error) {
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

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var payload ApiResponse[[]models.Blob]
	if err := json.Unmarshal(respBytes, &payload); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if resp.StatusCode != http.StatusOK || !payload.IsSuccess {
		return nil, fmt.Errorf("upload blob failed (%d): %s", resp.StatusCode, responseError(payload.Error, payload.ErrorCode))
	}

	if len(payload.Data) == 0 {
		return nil, fmt.Errorf("upload blob succeeded but no blob metadata returned")
	}

	return &payload.Data[0], nil
}

func (c *BotProviderClient) GenerateSandboxEditorOpenUrl(ctx context.Context, sandboxName string) (string, error) {
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

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	var payload ApiResponse[map[string]string]
	if err := json.Unmarshal(respBytes, &payload); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if resp.StatusCode != http.StatusOK || !payload.IsSuccess {
		return "", fmt.Errorf("generate sandbox editor open URL failed (%d): %s", resp.StatusCode, responseError(payload.Error, payload.ErrorCode))
	}

	openURL, ok := payload.Data["openURL"]
	if !ok {
		return "", fmt.Errorf("response missing openURL field")
	}

	return openURL, nil
}

func (c *BotProviderClient) SandboxFsList(ctx context.Context, sandboxName, path string) (*models.SandboxFsListResult, error) {
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

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var payload ApiResponse[models.SandboxFsListResult]
	if err := json.Unmarshal(respBytes, &payload); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	if resp.StatusCode != http.StatusOK || !payload.IsSuccess {
		return nil, fmt.Errorf("sandbox fs list failed (%d): %s", resp.StatusCode, responseError(payload.Error, payload.ErrorCode))
	}
	return &payload.Data, nil
}

func (c *BotProviderClient) SandboxFsRead(ctx context.Context, sandboxName, path string, offsetBytes, limitBytes *int64) ([]byte, *models.SandboxFsReadMeta, error) {
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ApiResponse[json.RawMessage]
		if jsonErr := json.Unmarshal(body, &errResp); jsonErr == nil && !errResp.IsSuccess {
			return nil, nil, fmt.Errorf("sandbox fs read failed (%d): %s", resp.StatusCode, responseError(errResp.Error, errResp.ErrorCode))
		}
		return nil, nil, fmt.Errorf("sandbox fs read failed (%d)", resp.StatusCode)
	}

	meta := &models.SandboxFsReadMeta{}
	if v := resp.Header.Get("X-Total-Bytes"); v != "" {
		meta.TotalBytes, _ = strconv.ParseInt(v, 10, 64)
	}
	meta.Truncated = resp.Header.Get("X-Truncated") == "true"

	return body, meta, nil
}

func (c *BotProviderClient) SandboxFsWrite(ctx context.Context, sandboxName, path string, reader io.Reader, filename string, mode *uint32, createOnly bool) (*models.SandboxFsWriteResult, error) {
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

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var payload ApiResponse[models.SandboxFsWriteResult]
	if err := json.Unmarshal(respBytes, &payload); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	if resp.StatusCode != http.StatusOK || !payload.IsSuccess {
		return nil, fmt.Errorf("sandbox fs write failed (%d): %s", resp.StatusCode, responseError(payload.Error, payload.ErrorCode))
	}
	return &payload.Data, nil
}

func (c *BotProviderClient) SandboxHeartbeat(ctx context.Context, sandboxName string) (*models.SandboxHeartbeatResult, error) {
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

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var payload ApiResponse[models.SandboxHeartbeatResult]
	if err := json.Unmarshal(respBytes, &payload); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	if resp.StatusCode != http.StatusOK || !payload.IsSuccess {
		return nil, fmt.Errorf("sandbox heartbeat failed (%d): %s", resp.StatusCode, responseError(payload.Error, payload.ErrorCode))
	}
	return &payload.Data, nil
}

func responseError(errMsg, errCode *string) string {
	if errMsg == nil && errCode == nil {
		return "unknown error"
	}
	if errMsg != nil && errCode != nil {
		return fmt.Sprintf("%s (%s)", *errMsg, *errCode)
	}
	if errMsg != nil {
		return *errMsg
	}
	return *errCode
}
