// Package api is a Go client for the OpenStreetMap API v0.6.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"strings"
)

const (
	DefaultBaseURL   = "https://api.openstreetmap.org/api/0.6"
	SandboxBaseURL   = "https://master.apis.dev.openstreetmap.org/api/0.6"
	DefaultUserAgent = "github.com/raspbeguy/osm/api"

	// errBodyCap is the max bytes read from an error response body. Larger
	// helps debug verbose server errors, but keeps memory bounded on hostile
	// or oversized payloads.
	errBodyCap = 4096
)

// marshalXMLDoc returns the XML header followed by the encoded value, ready
// to PUT/POST to the API. Shared by changeset/trace metadata builders.
func marshalXMLDoc(v any) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	enc := xml.NewEncoder(&buf)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	if err := enc.Flush(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type Client struct {
	BaseURL   string
	HTTP      *http.Client
	UserAgent string
}

func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{BaseURL: DefaultBaseURL, HTTP: httpClient, UserAgent: DefaultUserAgent}
}

// Do sends req, applies the User-Agent, and maps non-2xx into typed errors.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, errBodyCap))
		resp.Body.Close()
		return nil, mapHTTPError(resp.StatusCode, resp.Status, string(b))
	}
	return resp, nil
}

func (c *Client) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	return http.NewRequestWithContext(ctx, method, strings.TrimRight(c.BaseURL, "/")+path, body)
}

func (c *Client) getJSON(ctx context.Context, path string, into any) error {
	r, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	r.Header.Set("Accept", "application/json")
	resp, err := c.Do(r)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(into)
}

func (c *Client) getRaw(ctx context.Context, path, accept string) (string, error) {
	r, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return "", err
	}
	if accept != "" {
		r.Header.Set("Accept", accept)
	}
	resp, err := c.Do(r)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(io.LimitReader(resp.Body, 1<<24))
	return string(out), err
}

func (c *Client) getXML(ctx context.Context, path string, into any) error {
	r, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	r.Header.Set("Accept", "application/xml")
	resp, err := c.Do(r)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return xml.NewDecoder(resp.Body).Decode(into)
}

func (c *Client) sendBody(ctx context.Context, method, path string, body []byte, contentType string) (string, error) {
	var br io.Reader
	if body != nil {
		br = bytes.NewReader(body)
	}
	r, err := c.newRequest(ctx, method, path, br)
	if err != nil {
		return "", err
	}
	if contentType != "" {
		r.Header.Set("Content-Type", contentType)
	}
	resp, err := c.Do(r)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(io.LimitReader(resp.Body, 1<<24))
	return string(out), err
}
