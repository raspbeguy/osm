package api

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrNotFound           = errors.New("not found")
	ErrConflict           = errors.New("conflict")
	ErrChangesetClosed    = errors.New("changeset closed")
	ErrGone               = errors.New("gone")
	ErrPreconditionFailed = errors.New("precondition failed")
	ErrTooManyRequests    = errors.New("too many requests")
)

type HTTPError struct {
	StatusCode int
	Status     string
	Body       string
}

func (e *HTTPError) Error() string {
	if e.Body != "" {
		return fmt.Sprintf("osm api %s: %s", e.Status, strings.TrimSpace(e.Body))
	}
	return "osm api " + e.Status
}

func mapHTTPError(code int, status, body string) error {
	he := &HTTPError{StatusCode: code, Status: status, Body: body}
	var sentinel error
	switch code {
	case 401:
		sentinel = ErrUnauthorized
	case 403:
		sentinel = ErrForbidden
	case 404:
		sentinel = ErrNotFound
	case 409:
		// OSM uses 409 both for element-version conflicts and "changeset closed".
		if strings.Contains(strings.ToLower(body), "closed") {
			sentinel = ErrChangesetClosed
		} else {
			sentinel = ErrConflict
		}
	case 410:
		sentinel = ErrGone
	case 412:
		sentinel = ErrPreconditionFailed
	case 429:
		sentinel = ErrTooManyRequests
	}
	if sentinel != nil {
		return fmt.Errorf("%w: %s", sentinel, he.Error())
	}
	return he
}
