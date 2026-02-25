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

	"go.asgard-ai.com/asgard-sdk-go/pkg/models"
)

type apiResponse[T any] struct {
	IsSuccess bool    `json:"isSuccess"`
	Data      T       `json:"data"`
	Error     *string `json:"error"`
	ErrorCode *string `json:"errorCode"`
}

func (c *BotProviderClient) NewStreamer(ctx context.Context, message *models.GenericBotMessage) (BotProviderStreamer, error) {
	return NewStreaming(ctx, c.config, message)
}

func (c *BotProviderClient) SendMessage(ctx context.Context, message *models.GenericBotMessage, isDebug bool) (*models.GenericBotReply, error) {
	if message == nil {
		return nil, fmt.Errorf("message cannot be nil")
	}

	u := fmt.Sprintf("%s/ns/%s/bot-provider/%s/message",
		c.config.EdgeServerHost,
		url.PathEscape(c.config.Namespace),
		url.PathEscape(c.config.BotProviderName),
	)

	if isDebug {
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

	resp, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var payload apiResponse[models.GenericBotReply]
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

	var wrapper apiResponse[json.RawMessage]
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

	var wrapper apiResponse[json.RawMessage]
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

	var payload apiResponse[[]models.Blob]
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
