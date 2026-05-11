package client

import (
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

// SourceSetClient defines the interface for interacting with Edge Server SourceSet volume APIs.
type SourceSetClient interface {
	ListDirectory(ctx context.Context, path string, page, pageSize *int64) (*models.SourceSetListDirectoryResult, error)
	Stat(ctx context.Context, path string) (*models.SourceSetStatResult, error)
	ReadFile(ctx context.Context, path string, offsetBytes, limitBytes *int64) ([]byte, error)
	WriteFile(ctx context.Context, path string, reader io.Reader, filename string) (*models.SourceSetWriteFileResult, error)
	MakeDirectory(ctx context.Context, path string) error
	Remove(ctx context.Context, path string) error
	RemoveAll(ctx context.Context, path string) error
}

// SourceSetConfig holds the configuration for connecting to a SourceSet.
type SourceSetConfig struct {
	HTTPClient      *http.Client
	EdgeServerHost  string
	Namespace       string
	SourceSetName   string
	SourceSetApiKey string
	Headers         map[string]string
}

type sourceSetClient struct {
	config *SourceSetConfig
}

// NewSourceSetClient creates a SourceSet client with default HTTP settings.
func NewSourceSetClient(edgeServerHost, namespace, sourceSetName, apiKey string) SourceSetClient {
	return NewSourceSetClientWithConfig(&SourceSetConfig{
		HTTPClient:      &http.Client{Timeout: defaultHTTPTimeout},
		EdgeServerHost:  edgeServerHost,
		Namespace:       namespace,
		SourceSetName:   sourceSetName,
		SourceSetApiKey: apiKey,
	})
}

// NewSourceSetClientWithConfig creates a SourceSet client from config.
func NewSourceSetClientWithConfig(config *SourceSetConfig) SourceSetClient {
	if config == nil {
		config = &SourceSetConfig{}
	}
	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{Timeout: defaultHTTPTimeout}
	}
	return &sourceSetClient{config: config}
}

func (c *sourceSetClient) baseURL() string {
	return fmt.Sprintf("%s/ns/%s/source-set/%s",
		c.config.EdgeServerHost,
		url.PathEscape(c.config.Namespace),
		url.PathEscape(c.config.SourceSetName),
	)
}

func (c *sourceSetClient) newRequest(ctx context.Context, method, rawURL string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, rawURL, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-KEY", c.config.SourceSetApiKey)
	for k, v := range c.config.Headers {
		req.Header.Set(k, v)
	}
	return req, nil
}

func (c *sourceSetClient) doJSON(req *http.Request, out interface{}) error {
	resp, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	var wrapper ApiResponse[json.RawMessage]
	if err := json.Unmarshal(b, &wrapper); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}
	if resp.StatusCode != http.StatusOK || !wrapper.IsSuccess {
		return fmt.Errorf("request failed (%d): %s", resp.StatusCode, responseError(wrapper.Error, wrapper.ErrorCode))
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(wrapper.Data, out); err != nil {
		return fmt.Errorf("failed to decode response data: %w", err)
	}
	return nil
}

// ListDirectory lists directory contents at path. page and pageSize are optional.
func (c *sourceSetClient) ListDirectory(ctx context.Context, path string, page, pageSize *int64) (*models.SourceSetListDirectoryResult, error) {
	u, err := url.Parse(c.baseURL() + "/volume/list")
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}
	q := u.Query()
	q.Set("path", path)
	if page != nil {
		q.Set("page", strconv.FormatInt(*page, 10))
	}
	if pageSize != nil {
		q.Set("page_size", strconv.FormatInt(*pageSize, 10))
	}
	u.RawQuery = q.Encode()

	req, err := c.newRequest(ctx, http.MethodGet, u.String(), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	var result models.SourceSetListDirectoryResult
	if err := c.doJSON(req, &result); err != nil {
		return nil, fmt.Errorf("list directory failed: %w", err)
	}
	return &result, nil
}

// Stat returns metadata for the file or directory at path.
func (c *sourceSetClient) Stat(ctx context.Context, path string) (*models.SourceSetStatResult, error) {
	u, err := url.Parse(c.baseURL() + "/volume/stat")
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}
	q := u.Query()
	q.Set("path", path)
	u.RawQuery = q.Encode()

	req, err := c.newRequest(ctx, http.MethodGet, u.String(), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	var result models.SourceSetStatResult
	if err := c.doJSON(req, &result); err != nil {
		return nil, fmt.Errorf("stat failed: %w", err)
	}
	return &result, nil
}

// ReadFile downloads the file at path as raw bytes. offsetBytes and limitBytes are optional.
func (c *sourceSetClient) ReadFile(ctx context.Context, path string, offsetBytes, limitBytes *int64) ([]byte, error) {
	u, err := url.Parse(c.baseURL() + "/volume/file")
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}
	q := u.Query()
	q.Set("path", path)
	if offsetBytes != nil {
		q.Set("offset", strconv.FormatInt(*offsetBytes, 10))
	}
	if limitBytes != nil {
		q.Set("limit", strconv.FormatInt(*limitBytes, 10))
	}
	u.RawQuery = q.Encode()

	req, err := c.newRequest(ctx, http.MethodGet, u.String(), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("read file failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ApiResponse[json.RawMessage]
		if jsonErr := json.Unmarshal(body, &errResp); jsonErr == nil && !errResp.IsSuccess {
			return nil, fmt.Errorf("read file failed (%d): %s", resp.StatusCode, responseError(errResp.Error, errResp.ErrorCode))
		}
		return nil, fmt.Errorf("read file failed (%d)", resp.StatusCode)
	}
	return body, nil
}

// WriteFile uploads reader as a multipart file to path.
func (c *sourceSetClient) WriteFile(ctx context.Context, path string, reader io.Reader, filename string) (*models.SourceSetWriteFileResult, error) {
	u, err := url.Parse(c.baseURL() + "/volume/file")
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}
	q := u.Query()
	q.Set("path", path)
	u.RawQuery = q.Encode()

	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	req, err := c.newRequest(ctx, http.MethodPut, u.String(), pr)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

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

	var result models.SourceSetWriteFileResult
	if err := c.doJSON(req, &result); err != nil {
		return nil, fmt.Errorf("write file failed: %w", err)
	}
	return &result, nil
}

// MakeDirectory creates a directory (and all missing parents) at path.
func (c *sourceSetClient) MakeDirectory(ctx context.Context, path string) error {
	u, err := url.Parse(c.baseURL() + "/volume/mkdir")
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}
	q := u.Query()
	q.Set("path", path)
	u.RawQuery = q.Encode()

	req, err := c.newRequest(ctx, http.MethodPost, u.String(), http.NoBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if err := c.doJSON(req, nil); err != nil {
		return fmt.Errorf("mkdir failed: %w", err)
	}
	return nil
}

// Remove deletes the file or empty directory at path.
func (c *sourceSetClient) Remove(ctx context.Context, path string) error {
	u, err := url.Parse(c.baseURL() + "/volume/item")
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}
	q := u.Query()
	q.Set("path", path)
	u.RawQuery = q.Encode()

	req, err := c.newRequest(ctx, http.MethodDelete, u.String(), http.NoBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if err := c.doJSON(req, nil); err != nil {
		return fmt.Errorf("remove failed: %w", err)
	}
	return nil
}

// RemoveAll recursively deletes path and everything under it.
func (c *sourceSetClient) RemoveAll(ctx context.Context, path string) error {
	u, err := url.Parse(c.baseURL() + "/volume/all")
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}
	q := u.Query()
	q.Set("path", path)
	u.RawQuery = q.Encode()

	req, err := c.newRequest(ctx, http.MethodDelete, u.String(), http.NoBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if err := c.doJSON(req, nil); err != nil {
		return fmt.Errorf("remove all failed: %w", err)
	}
	return nil
}
