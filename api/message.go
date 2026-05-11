package api

import (
	"context"
	"errors"
	"fmt"
	"net/url"
)

type Message struct {
	ID         int64  `json:"id"`
	FromUser   string `json:"from_display_name"`
	FromUserID int64  `json:"from_user_id"`
	ToUser     string `json:"to_display_name"`
	ToUserID   int64  `json:"to_user_id"`
	SentOn     string `json:"sent_on"`
	Title      string `json:"title"`
	Body       string `json:"body,omitempty"`
	BodyFormat string `json:"body_format,omitempty"`
	Read       bool   `json:"message_read"`
	Deleted    bool   `json:"deleted"`
}

func (c *Client) ListInbox(ctx context.Context) ([]*Message, error) {
	return c.listMessages(ctx, "/user/messages/inbox.json")
}

func (c *Client) ListOutbox(ctx context.Context) ([]*Message, error) {
	return c.listMessages(ctx, "/user/messages/outbox.json")
}

func (c *Client) listMessages(ctx context.Context, path string) ([]*Message, error) {
	var wrap struct {
		Messages []*Message `json:"messages"`
	}
	if err := c.getJSON(ctx, path, &wrap); err != nil {
		return nil, err
	}
	return wrap.Messages, nil
}

// GetMessage returns a single message including body.
func (c *Client) GetMessage(ctx context.Context, id int64) (*Message, error) {
	var wrap struct {
		Message *Message `json:"message"`
	}
	if err := c.getJSON(ctx, fmt.Sprintf("/user/messages/%d.json", id), &wrap); err != nil {
		return nil, err
	}
	if wrap.Message == nil {
		return nil, errors.New("empty message in response")
	}
	return wrap.Message, nil
}

// MarkRead toggles the read flag on a message.
func (c *Client) MarkRead(ctx context.Context, id int64, read bool) error {
	status := "unread"
	if read {
		status = "read"
	}
	body := url.Values{"read_status": {status}}.Encode()
	_, err := c.sendBody(ctx, "PUT", fmt.Sprintf("/user/messages/%d", id), []byte(body), "application/x-www-form-urlencoded")
	return err
}

func (c *Client) DeleteMessage(ctx context.Context, id int64) error {
	_, err := c.sendBody(ctx, "DELETE", fmt.Sprintf("/user/messages/%d", id), nil, "")
	return err
}
