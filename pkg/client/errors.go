package client

import (
	"errors"
	"fmt"
	"net/http"
)

// APIError is returned from every SDK HTTP call when the edgeserver responds
// with a non-2xx status or an envelope with isSuccess=false. Use errors.As to
// inspect StatusCode / ErrorCode, or one of the Is<Status> helpers below.
type APIError struct {
	StatusCode int    // HTTP status, e.g. 412
	ErrorCode  string // server errorCode, e.g. "FAILED_PRECONDITION"
	Message    string // server error string; empty if the body was not parseable
	Op         string // SDK operation that failed, e.g. "sandbox heartbeat"
}

func (e *APIError) Error() string {
	parts := fmt.Sprintf("%d", e.StatusCode)
	if e.ErrorCode != "" {
		parts = fmt.Sprintf("%s %s", parts, e.ErrorCode)
	}
	if e.Message != "" {
		return fmt.Sprintf("%s failed (%s): %s", e.Op, parts, e.Message)
	}
	return fmt.Sprintf("%s failed (%s)", e.Op, parts)
}

// StatusCode returns the HTTP status from an APIError reachable via errors.As,
// or 0 if err is nil or does not wrap an APIError.
func StatusCode(err error) int {
	var ae *APIError
	if errors.As(err, &ae) {
		return ae.StatusCode
	}
	return 0
}

// IsBadRequest reports whether err is an APIError with HTTP 400.
func IsBadRequest(err error) bool { return statusIs(err, http.StatusBadRequest) }

// IsUnauthorized reports whether err is an APIError with HTTP 401.
func IsUnauthorized(err error) bool { return statusIs(err, http.StatusUnauthorized) }

// IsForbidden reports whether err is an APIError with HTTP 403.
func IsForbidden(err error) bool { return statusIs(err, http.StatusForbidden) }

// IsNotFound reports whether err is an APIError with HTTP 404.
func IsNotFound(err error) bool { return statusIs(err, http.StatusNotFound) }

// IsConflict reports whether err is an APIError with HTTP 409.
func IsConflict(err error) bool { return statusIs(err, http.StatusConflict) }

// IsPreconditionFailed reports whether err is an APIError with HTTP 412.
// asgard-core uses 412 to signal "sandbox not found, please launch via message
// API first" and similar precondition errors.
func IsPreconditionFailed(err error) bool { return statusIs(err, http.StatusPreconditionFailed) }

func statusIs(err error, code int) bool {
	var ae *APIError
	return errors.As(err, &ae) && ae.StatusCode == code
}
