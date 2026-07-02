package parser

import (
	"encoding/base64"
	"strings"
)

// ParseSubscription parses a base64-encoded subscription string into proxy URIs.
func ParseSubscription(data []byte) ([]string, error) {
	str := strings.TrimSpace(string(data))

	// Try base64 decode first (typical subscription format)
	decoded, err := base64Decode(str)
	if err != nil {
		// Not base64, try plain text
		decoded = str
	}

	// Split by newlines
	var uris []string
	for _, line := range strings.Split(decoded, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}
		uris = append(uris, line)
	}

	return uris, nil
}

// base64Decode attempts to decode a base64 string with various paddings.
func base64Decode(s string) (string, error) {
	// Normalize: remove whitespace
	s = strings.TrimSpace(s)

	// Try standard base64
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err == nil {
		return string(decoded), nil
	}

	// Try URL-safe base64
	decoded, err = base64.URLEncoding.DecodeString(s)
	if err == nil {
		return string(decoded), nil
	}

	// Try with padding fix
	pad := len(s) % 4
	if pad > 0 {
		s += strings.Repeat("=", 4-pad)
	}
	decoded, err = base64.StdEncoding.DecodeString(s)
	if err == nil {
		return string(decoded), nil
	}

	// Try URL-safe with padding fix
	decoded, err = base64.URLEncoding.DecodeString(s)
	if err == nil {
		return string(decoded), nil
	}

	return "", err
}

// base64RawDecode decodes raw base64 (no padding) URL-safe variant.
func base64RawDecode(s string) (string, error) {
	s = strings.TrimSpace(s)
	decoded, err := base64.RawURLEncoding.DecodeString(s)
	if err == nil {
		return string(decoded), nil
	}
	decoded, err = base64.RawStdEncoding.DecodeString(s)
	if err == nil {
		return string(decoded), nil
	}
	return "", err
}

// IsSubscriptionFormat checks if data looks like a subscription (base64 encoded).
func IsSubscriptionFormat(data []byte) bool {
	s := strings.TrimSpace(string(data))
	_, err := base64Decode(s)
	return err == nil && len(s) > 50 && !strings.HasPrefix(s, "{") && !strings.HasPrefix(s, "proxies:")
}