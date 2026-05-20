package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// decodeAPIResponse reads resp.Body, decodes it into ApiResponse[T], and
// returns the unwrapped Data on success. On non-2xx or isSuccess=false it
// returns an *APIError populated from the envelope.
func decodeAPIResponse[T any](resp *http.Response, op string) (T, error) {
	var zero T
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return zero, fmt.Errorf("%s: failed to read response body: %w", op, readErr)
	}

	var wrapper ApiResponse[T]
	// Ignore unmarshal errors — server may return non-JSON on infrastructure
	// failures; the *APIError below still surfaces the status code.
	_ = json.Unmarshal(body, &wrapper)

	if resp.StatusCode/100 != 2 || !wrapper.IsSuccess {
		return zero, newAPIError(op, resp.StatusCode, wrapper.Error, wrapper.ErrorCode)
	}
	return wrapper.Data, nil
}

// decodeAPIError reads an error envelope from a response whose success body
// is binary (e.g. SandboxFsRead, ReadFile). Caller must only invoke this when
// resp.StatusCode is non-2xx.
func decodeAPIError(resp *http.Response, op string) error {
	body, _ := io.ReadAll(resp.Body)
	var wrapper ApiResponse[json.RawMessage]
	_ = json.Unmarshal(body, &wrapper)
	return newAPIError(op, resp.StatusCode, wrapper.Error, wrapper.ErrorCode)
}

func newAPIError(op string, status int, msg, code *string) *APIError {
	e := &APIError{StatusCode: status, Op: op}
	if msg != nil {
		e.Message = *msg
	}
	if code != nil {
		e.ErrorCode = *code
	}
	return e
}
