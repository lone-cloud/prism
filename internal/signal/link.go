package signal

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type LinkDevice struct {
	client      *Client
	qrCode      string
	generatedAt time.Time
	ttl         time.Duration
}

func NewLinkDevice(client *Client) *LinkDevice {
	return &LinkDevice{
		client: client,
		ttl:    10 * time.Minute,
	}
}

type StartLinkResponse struct {
	DeviceLinkURI string `json:"deviceLinkUri"`
}

func (l *LinkDevice) GenerateQR() (string, error) {
	if l.qrCode != "" && time.Since(l.generatedAt) < l.ttl {
		return l.qrCode, nil
	}

	result, err := l.client.Call("startLink", nil)
	if err != nil {
		return "", fmt.Errorf("failed to start link: %w", err)
	}

	var response StartLinkResponse
	if err := json.Unmarshal(result, &response); err != nil {
		return "", fmt.Errorf("failed to parse link response: %w", err)
	}

	uri := response.DeviceLinkURI
	if uri == "" {
		return "", fmt.Errorf("empty device link URI")
	}

	// Generate QR code via qrserver.com API
	qrURL := fmt.Sprintf("https://api.qrserver.com/v1/create-qr-code/?size=300x300&data=%s", url.QueryEscape(uri))
	//nolint:gosec // QR code generation URL is from trusted API
	resp, err := http.Get(qrURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch QR code: %w", err)
	}
	defer resp.Body.Close()

	qrData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read QR code: %w", err)
	}

	// Convert to base64 data URL
	base64Data := base64.StdEncoding.EncodeToString(qrData)
	l.qrCode = fmt.Sprintf("data:image/png;base64,%s", base64Data)
	l.generatedAt = time.Now()

	go l.finishLink(uri)

	return l.qrCode, nil
}

func (l *LinkDevice) finishLink(uri string) {
	params := map[string]interface{}{
		"deviceLinkUri": uri,
	}

	// This blocks until device is linked or times out
	_, err := l.client.Call("finishLink", params)
	if err == nil {
		// Clear cache on successful link
		l.qrCode = ""
		l.generatedAt = time.Time{}
	}
}
