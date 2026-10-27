// Package httpx is the outbound HTTP client providers share: one timeout per
// call, JSON and form helpers, and a response type that keeps the status and
// a body snippet for error mapping.
package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Response struct {
	Status int
	Body   []byte
	Header http.Header
}

func (r *Response) JSON(v any) error {
	if len(r.Body) == 0 {
		return nil
	}
	return json.Unmarshal(r.Body, v)
}

// Snippet returns the first n bytes of the body for log and error messages.
func (r *Response) Snippet(n int) string {
	s := strings.TrimSpace(string(r.Body))
	if len(s) > n {
		return s[:n]
	}
	return s
}

type Client struct {
	HTTP      *http.Client
	UserAgent string
	// MaxBody caps what is read from a response (default 16 MB).
	MaxBody int64
}

func New(timeout time.Duration) *Client {
	return &Client{HTTP: &http.Client{Timeout: timeout}, UserAgent: "MixistAnalytics/1.0", MaxBody: 16 << 20}
}

func (c *Client) Do(ctx context.Context, method, rawURL string, body io.Reader, headers map[string]string) (*Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, rawURL, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	max := c.MaxBody
	if max == 0 {
		max = 16 << 20
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, max))
	if err != nil {
		return nil, err
	}
	return &Response{Status: res.StatusCode, Body: data, Header: res.Header}, nil
}

func (c *Client) Get(ctx context.Context, rawURL string, headers map[string]string) (*Response, error) {
	return c.Do(ctx, http.MethodGet, rawURL, nil, headers)
}

func (c *Client) PostJSON(ctx context.Context, rawURL string, payload any, headers map[string]string) (*Response, error) {
	buf, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	h := map[string]string{"Content-Type": "application/json"}
	for k, v := range headers {
		h[k] = v
	}
	return c.Do(ctx, http.MethodPost, rawURL, bytes.NewReader(buf), h)
}

func (c *Client) PostForm(ctx context.Context, rawURL string, form url.Values, headers map[string]string) (*Response, error) {
	h := map[string]string{"Content-Type": "application/x-www-form-urlencoded"}
	for k, v := range headers {
		h[k] = v
	}
	return c.Do(ctx, http.MethodPost, rawURL, strings.NewReader(form.Encode()), h)
}

// WithQuery appends query parameters to a URL.
func WithQuery(rawURL string, q url.Values) string {
	if len(q) == 0 {
		return rawURL
	}
	sep := "?"
	if strings.Contains(rawURL, "?") {
		sep = "&"
	}
	return rawURL + sep + q.Encode()
}

// IsTimeout reports whether an error came from the client deadline.
func IsTimeout(err error) bool {
	if err == nil {
		return false
	}
	if ne, ok := err.(interface{ Timeout() bool }); ok && ne.Timeout() {
		return true
	}
	return strings.Contains(err.Error(), "deadline exceeded") || strings.Contains(err.Error(), "Client.Timeout")
}

func Describe(r *Response) string {
	return fmt.Sprintf("HTTP %d: %s", r.Status, r.Snippet(200))
}
