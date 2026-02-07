package signal

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"
)

type LinkDevice struct {
	client        *Client
	deviceName    string
	qrCode        string
	deviceLinkUri string
	generatedAt   time.Time
	ttl           time.Duration
}

func NewLinkDevice(client *Client, deviceName string) *LinkDevice {
	return &LinkDevice{
		client:     client,
		deviceName: deviceName,
		ttl:        10 * time.Minute,
	}
}

type StartLinkResponse struct {
	DeviceLinkURI string `json:"deviceLinkUri"`
}

func (l *LinkDevice) GenerateQR() (string, error) {
	if l.client == nil {
		return "", fmt.Errorf("signal client not initialized")
	}

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

	l.deviceLinkUri = uri
	qrURL := fmt.Sprintf("https://api.qrserver.com/v1/create-qr-code/?size=300x300&data=%s", url.QueryEscape(uri))
	l.qrCode = qrURL
	l.generatedAt = time.Now()
	go l.finishLink()

	return l.qrCode, nil
}

func (l *LinkDevice) finishLink() {
	params := map[string]interface{}{
		"deviceLinkUri": l.deviceLinkUri,
		"deviceName":    l.deviceName,
	}

	_, err := l.client.Call("finishLink", params)
	if err == nil {
		l.qrCode = ""
		l.generatedAt = time.Time{}
	}
}
