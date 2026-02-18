package webpush

import (
	"crypto/ecdh"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
)

func validatePushEndpoint(raw string, requireHTTPS bool) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u == nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("invalid pushEndpoint URL")
	}

	if u.Scheme != "https" && u.Scheme != "http" {
		return fmt.Errorf("pushEndpoint must use http or https")
	}

	if requireHTTPS && u.Scheme != "https" {
		return fmt.Errorf("encrypted webpush endpoint must use https")
	}

	return nil
}

func normalizeVAPIDPrivateKey(raw string) (string, error) {
	decoded, err := decodeBase64URL(raw)
	if err != nil {
		return "", fmt.Errorf("invalid VAPID private key encoding")
	}

	if _, err := ecdh.P256().NewPrivateKey(decoded); err != nil {
		return "", fmt.Errorf("invalid VAPID private key scalar")
	}

	return base64.RawURLEncoding.EncodeToString(decoded), nil
}

func deriveVAPIDPublicKey(privateKey string) (string, error) {
	normalizedPrivateKey, err := normalizeVAPIDPrivateKey(privateKey)
	if err != nil {
		return "", err
	}

	privateBytes, err := decodeBase64URL(normalizedPrivateKey)
	if err != nil {
		return "", fmt.Errorf("invalid VAPID private key encoding")
	}

	privateECKey, err := ecdh.P256().NewPrivateKey(privateBytes)
	if err != nil {
		return "", fmt.Errorf("invalid VAPID private key scalar")
	}
	publicBytes := privateECKey.PublicKey().Bytes()

	return base64.RawURLEncoding.EncodeToString(publicBytes), nil
}

func normalizeP256DH(raw string) (string, error) {
	decoded, err := decodeBase64URL(raw)
	if err != nil {
		return "", fmt.Errorf("invalid p256dh encoding")
	}

	if len(decoded) != 65 || decoded[0] != 0x04 {
		return "", fmt.Errorf("invalid p256dh key format")
	}

	if _, err := ecdh.P256().NewPublicKey(decoded); err != nil {
		return "", fmt.Errorf("invalid p256dh point")
	}

	return base64.RawURLEncoding.EncodeToString(decoded), nil
}

func normalizeAuthSecret(raw string) (string, error) {
	decoded, err := decodeBase64URL(raw)
	if err != nil {
		return "", fmt.Errorf("invalid auth encoding")
	}

	if len(decoded) != 16 {
		return "", fmt.Errorf("invalid auth length: expected 16 bytes, got %d", len(decoded))
	}

	return base64.RawURLEncoding.EncodeToString(decoded), nil
}

func decodeBase64URL(raw string) ([]byte, error) {
	key := strings.TrimSpace(raw)
	if decoded, err := base64.RawURLEncoding.DecodeString(key); err == nil {
		return decoded, nil
	}
	decoded, err := base64.URLEncoding.DecodeString(key)
	if err == nil {
		return decoded, nil
	}
	return nil, err
}
