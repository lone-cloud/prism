package util

import (
	"crypto/subtle"
	"encoding/base64"
	"net"
	"net/http"
	"strings"
)

func VerifyAPIKey(r *http.Request, apiKey string, allowInsecureHTTP bool) bool {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return false
	}

	var password string

	switch {
	case strings.HasPrefix(auth, "Bearer "):
		password = strings.TrimPrefix(auth, "Bearer ")
	case strings.HasPrefix(auth, "Basic "):
		payload := strings.TrimPrefix(auth, "Basic ")
		decoded, err := base64.StdEncoding.DecodeString(payload)
		if err != nil {
			return false
		}
		parts := strings.SplitN(string(decoded), ":", 2)
		if len(parts) != 2 {
			return false
		}
		password = parts[1]
	default:
		return false
	}

	if len(password) != len(apiKey) {
		return false
	}

	// Enforce HTTPS for non-local connections unless explicitly allowed
	if !allowInsecureHTTP {
		proto := r.Header.Get("X-Forwarded-Proto")
		if proto == "" {
			if r.TLS != nil {
				proto = "https"
			} else {
				proto = "http"
			}
		}

		clientIP := GetClientIP(r)
		if proto != "https" && !isLocalIP(clientIP) {
			return false
		}
	}

	return subtle.ConstantTimeCompare([]byte(password), []byte(apiKey)) == 1
}

func isLocalIP(addr string) bool {
	if addr == "" || addr == "::1" || addr == "localhost" {
		return true
	}

	ip := net.ParseIP(addr)
	if ip == nil {
		return false
	}

	if ip.IsLoopback() {
		return true
	}

	if ip.To4() != nil {
		return ip.IsPrivate()
	}

	return false
}

func GetClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}

	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func IsLocalhost(ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}
	return parsedIP.IsLoopback()
}

func GetLANIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}

	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String()
			}
		}
	}

	return ""
}
