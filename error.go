package web

import (
	"fmt"
	"net/http"
)

type HTTPError struct {
	StatusCode int
	Internal   error
}

func NewHTTPError(statusCode int) *HTTPError {
	return &HTTPError{StatusCode: statusCode}
}

func NewHTTPErrorWithInternalError(statusCode int, err error) *HTTPError {
	return &HTTPError{StatusCode: statusCode, Internal: err}
}

func (e *HTTPError) Error() string {
	if e.Internal != nil {
		return fmt.Sprintf("%d %s: %v", e.StatusCode, http.StatusText(e.StatusCode), e.Internal)
	}
	return fmt.Sprintf("%d %s", e.StatusCode, http.StatusText(e.StatusCode))
}
