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
//
// Every `path` is relative to the SourceSet's volume root: it must not start with
// '/', must not contain a "." or ".." component, and must not have consecutive or
// trailing slashes. Copy and Move take two paths and the contract applies to each
// independently. The server enforces this, so a rejected path surfaces as an
// *APIError with status 400.
type SourceSetClient interface {
	ListDirectory(ctx context.Context, path string, page, pageSize *int64) (*models.SourceSetListDirectoryResult, error)
	Stat(ctx context.Context, path string) (*models.SourceSetStatResult, error)
	// ReadFile returns the file bytes plus the X-Total-Bytes / X-Truncated
	// metadata, so a caller can tell a bounded read from a whole file.
	ReadFile(ctx context.Context, path string, offsetBytes, limitBytes *int64) ([]byte, *models.SourceSetReadMeta, error)
	// WriteFile uploads reader to path. mode is the Unix permission bits (nil for
	// the server default 0644); createOnly fails with a 409 *APIError instead of
	// truncating a file that already exists.
	WriteFile(ctx context.Context, path string, reader io.Reader, filename string, mode *uint32, createOnly bool) (*models.SourceSetWriteFileResult, error)
	MakeDirectory(ctx context.Context, path string) error
	Remove(ctx context.Context, path string) error
	RemoveAll(ctx context.Context, path string) error
	// Copy duplicates src to dst, recursing when src is a directory. Without
	// overwrite an existing destination is a 409 *APIError.
	Copy(ctx context.Context, src, dst string, overwrite bool) (*models.SourceSetCopyResult, error)
	// Move relocates src to dst; a rename is a move within the same parent.
	// Without overwrite an existing destination is a 409 *APIError.
	Move(ctx context.Context, src, dst string, overwrite bool) error
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

func (c *sourceSetClient) doJSON(req *http.Request, op string, out interface{}) error {
	resp, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := decodeAPIResponse[json.RawMessage](resp, op)
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
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
	if err := c.doJSON(req, "list directory", &result); err != nil {
		return nil, err
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
	if err := c.doJSON(req, "stat", &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ReadFile downloads the file at path as raw bytes, along with the size metadata
// from the response headers. offsetBytes and limitBytes are optional.
func (c *sourceSetClient) ReadFile(ctx context.Context, path string, offsetBytes, limitBytes *int64) ([]byte, *models.SourceSetReadMeta, error) {
	u, err := url.Parse(c.baseURL() + "/volume/file")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse URL: %w", err)
	}
	q := u.Query()
	q.Set("path", path)
	// offset_bytes / limit_bytes match the sandbox filesystem API. The server also
	// still accepts the older offset / limit names.
	if offsetBytes != nil {
		q.Set("offset_bytes", strconv.FormatInt(*offsetBytes, 10))
	}
	if limitBytes != nil {
		q.Set("limit_bytes", strconv.FormatInt(*limitBytes, 10))
	}
	u.RawQuery = q.Encode()

	req, err := c.newRequest(ctx, http.MethodGet, u.String(), http.NoBody)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("read file failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return nil, nil, decodeAPIError(resp, "read file")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response body: %w", err)
	}

	meta := &models.SourceSetReadMeta{}
	if v := resp.Header.Get("X-Total-Bytes"); v != "" {
		meta.TotalBytes, _ = strconv.ParseInt(v, 10, 64)
	}
	meta.Truncated = resp.Header.Get("X-Truncated") == "true"

	return body, meta, nil
}

// WriteFile uploads reader as a multipart file to path.
func (c *sourceSetClient) WriteFile(ctx context.Context, path string, reader io.Reader, filename string, mode *uint32, createOnly bool) (*models.SourceSetWriteFileResult, error) {
	u, err := url.Parse(c.baseURL() + "/volume/file")
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
	if err := c.doJSON(req, "write file", &result); err != nil {
		return nil, err
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

	if err := c.doJSON(req, "mkdir", nil); err != nil {
		return err
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

	if err := c.doJSON(req, "remove", nil); err != nil {
		return err
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

	if err := c.doJSON(req, "remove all", nil); err != nil {
		return err
	}
	return nil
}

// srcDstRequest builds a POST to a src/dst volume endpoint (copy / move).
func (c *sourceSetClient) srcDstRequest(ctx context.Context, op, src, dst string, overwrite bool) (*http.Request, error) {
	u, err := url.Parse(c.baseURL() + "/volume/" + op)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}
	q := u.Query()
	q.Set("src", src)
	q.Set("dst", dst)
	if overwrite {
		q.Set("overwrite", "true")
	}
	u.RawQuery = q.Encode()

	req, err := c.newRequest(ctx, http.MethodPost, u.String(), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	return req, nil
}

// Copy duplicates src to dst, recursing when src is a directory.
func (c *sourceSetClient) Copy(ctx context.Context, src, dst string, overwrite bool) (*models.SourceSetCopyResult, error) {
	req, err := c.srcDstRequest(ctx, "copy", src, dst, overwrite)
	if err != nil {
		return nil, err
	}

	var result models.SourceSetCopyResult
	if err := c.doJSON(req, "copy", &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Move relocates src to dst. Renaming is a move within the same parent directory.
func (c *sourceSetClient) Move(ctx context.Context, src, dst string, overwrite bool) error {
	req, err := c.srcDstRequest(ctx, "move", src, dst, overwrite)
	if err != nil {
		return err
	}

	if err := c.doJSON(req, "move", nil); err != nil {
		return err
	}
	return nil
}
