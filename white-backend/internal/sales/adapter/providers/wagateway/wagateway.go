// Package wagateway talks to the self-hosted WhatsApp gateway of NüHabit
// (services/wa-gateway, HTTP only): POST {url}/send with x-gateway-token,
// body {target, message}, answer {success, error, messageId}. The same
// token authenticates the gateway's inbound forwards to us. Without
// WA_GATEWAY_URL and WA_GATEWAY_TOKEN the provider is disabled.
package wagateway

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/sales/app"
)

type Gateway struct {
	url    string
	token  string
	client *http.Client
}

func New(url, token string, client *http.Client) *Gateway {
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	return &Gateway{url: strings.TrimRight(url, "/"), token: token, client: client}
}

func (g *Gateway) Enabled() bool { return g.url != "" && g.token != "" }

func (g *Gateway) VerifyToken(token string) bool { return app.VerifyGatewayToken(g.token, token) }

type sendResponse struct {
	Success   bool   `json:"success"`
	Error     string `json:"error"`
	MessageID string `json:"messageId"`
}

func (g *Gateway) Send(ctx context.Context, phone, text string) (string, error) {
	if !g.Enabled() {
		return "", errors.New("WA gateway belum dikonfigurasi")
	}
	body, _ := json.Marshal(map[string]string{"target": phone, "message": text})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.url+"/send", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-gateway-token", g.token)
	res, err := g.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("WA gateway tidak terjangkau: %w", err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 64<<10))
	var out sendResponse
	_ = json.Unmarshal(raw, &out)
	if res.StatusCode != http.StatusOK || !out.Success {
		reason := out.Error
		if reason == "" {
			reason = fmt.Sprintf("status %d", res.StatusCode)
		}
		return "", errors.New("WA gagal terkirim: " + reason)
	}
	return out.MessageID, nil
}
