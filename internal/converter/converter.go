package converter

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/user/subai/internal/parser"
	"gopkg.in/yaml.v3"
)

// Engine is the conversion engine.
type Engine struct{}

// New creates a new Engine.
func New() *Engine {
	return &Engine{}
}

// Convert converts a list of proxies to the target format.
// target can be: clash, base64, mixed
func (e *Engine) Convert(proxies []parser.Proxy, target string) ([]byte, error) {
	switch target {
	case "clash":
		return toClash(proxies)
	case "base64":
		return toBase64(proxies)
	case "mixed":
		return toMixed(proxies)
	default:
		return nil, fmt.Errorf("unknown target format: %s", target)
	}
}

// toClash generates a complete Clash-compatible YAML configuration.
func toClash(proxies []parser.Proxy) ([]byte, error) {
	out := map[string]interface{}{
		"port":                 7890,
		"socks-port":           7891,
		"allow-lan":            false,
		"mode":                 "Rule",
		"log-level":            "info",
		"external-controller":  "127.0.0.1:9090",
		"proxies":              proxies,
	}

	if len(proxies) > 0 {
		names := make([]string, len(proxies))
		for i, p := range proxies {
			if p.Name != "" {
				names[i] = p.Name
			} else {
				names[i] = fmt.Sprintf("%s-%d", p.Server, p.Port)
			}
		}

		out["proxy-groups"] = []map[string]interface{}{
			{
				"name":    "Proxy",
				"type":    "select",
				"proxies": names,
			},
			{
				"name":     "Auto",
				"type":     "url-test",
				"proxies":  names,
				"url":      "http://www.gstatic.com/generate_204",
				"interval": 300,
			},
			{
				"name":     "Fallback",
				"type":     "fallback",
				"proxies":  names,
				"url":      "http://www.gstatic.com/generate_204",
				"interval": 300,
			},
			{
				"name":    "Final",
				"type":    "select",
				"proxies": []string{"Proxy", "Auto", "DIRECT"},
			},
		}
	}

	out["rules"] = []string{
		"MATCH,Final",
	}

	return yaml.Marshal(out)
}

// toBase64 generates a base64-encoded subscription (URI list).
func toBase64(proxies []parser.Proxy) ([]byte, error) {
	var buf bytes.Buffer
	for _, p := range proxies {
		uri := proxyToURI(p)
		if uri != "" {
			if buf.Len() > 0 {
				buf.WriteByte('\n')
			}
			buf.WriteString(uri)
		}
	}
	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())
	return []byte(encoded), nil
}

// toMixed generates both Clash config and base64 output combined.
func toMixed(proxies []parser.Proxy) ([]byte, error) {
	clashBytes, err := toClash(proxies)
	if err != nil {
		return nil, fmt.Errorf("clash section: %w", err)
	}

	b64Bytes, err := toBase64(proxies)
	if err != nil {
		return nil, fmt.Errorf("base64 section: %w", err)
	}

	var buf bytes.Buffer
	buf.WriteString("# Clash Config\n")
	buf.Write(clashBytes)
	buf.WriteString("\n# Base64 Subscription\n")
	buf.Write(b64Bytes)
	buf.WriteByte('\n')

	return buf.Bytes(), nil
}

// proxyToURI converts a Proxy back to a subscription URI string.
func proxyToURI(p parser.Proxy) string {
	switch p.Type {
	case "ss":
		userInfo := base64.RawStdEncoding.EncodeToString([]byte(p.Cipher + ":" + p.Password))
		uri := fmt.Sprintf("ss://%s@%s:%d", userInfo, p.Server, p.Port)
		if p.Name != "" {
			uri += "#" + p.Name
		}
		return uri
	case "ssr":
		passB64 := base64.RawURLEncoding.EncodeToString([]byte(p.Password))
		mainPart := fmt.Sprintf("%s:%d:%s:%s:%s:%s", p.Server, p.Port, p.Protocol, p.Cipher, p.Obfs, passB64)
		params := ""
		if p.ObfsParam != "" {
			params += "&obfsparam=" + base64.RawURLEncoding.EncodeToString([]byte(p.ObfsParam))
		}
		if p.ProtocolParam != "" {
			params += "&protoparam=" + base64.RawURLEncoding.EncodeToString([]byte(p.ProtocolParam))
		}
		if p.Name != "" {
			params += "&remarks=" + base64.RawURLEncoding.EncodeToString([]byte(p.Name))
		}
		if params != "" {
			mainPart += "/?" + strings.TrimPrefix(params, "&")
		}
		encoded := base64.RawURLEncoding.EncodeToString([]byte(mainPart))
		return "ssr://" + encoded
	case "vmess":
		vmess := map[string]interface{}{
			"v":    "2",
			"ps":   p.Name,
			"add":  p.Server,
			"port": p.Port,
			"id":   p.UUID,
			"aid":  "0",
			"net":  p.Network,
			"type": "none",
			"tls":  "",
		}
		if p.Network == "ws" && p.WSPath != "" {
			vmess["path"] = p.WSPath
		}
		if p.Network == "ws" && p.SNI != "" {
			vmess["host"] = p.SNI
		}
		if p.Encryption != "" {
			vmess["scy"] = p.Encryption
		}
		if p.Security == "tls" {
			vmess["tls"] = "tls"
		}
		jsonBytes, _ := json.Marshal(vmess)
		return "vmess://" + base64.RawURLEncoding.EncodeToString(jsonBytes)
	case "vless":
		u := fmt.Sprintf("vless://%s@%s:%d", p.UUID, p.Server, p.Port)
		params := []string{}
		if p.Encryption == "" {
			params = append(params, "encryption=none")
		} else {
			params = append(params, "encryption="+p.Encryption)
		}
		params = append(params, "type="+p.Network)
		if p.Security != "" {
			params = append(params, "security="+p.Security)
		}
		if p.SNI != "" {
			params = append(params, "sni="+p.SNI)
		}
		if p.Flow != "" {
			params = append(params, "flow="+p.Flow)
		}
		if p.Fingerprint != "" {
			params = append(params, "fp="+p.Fingerprint)
		}
		if p.PublicKey != "" {
			params = append(params, "pbk="+p.PublicKey)
		}
		if p.ShortID != "" {
			params = append(params, "sid="+p.ShortID)
		}
		if p.Network == "ws" && p.WSPath != "" {
			params = append(params, "path="+p.WSPath)
		}
		if len(params) > 0 {
			u += "?" + strings.Join(params, "&")
		}
		if p.Name != "" {
			u += "#" + p.Name
		}
		return u
	case "trojan":
		u := fmt.Sprintf("trojan://%s@%s:%d", p.Password, p.Server, p.Port)
		params := []string{"type=" + p.Network}
		if p.SNI != "" {
			params = append(params, "sni="+p.SNI)
		}
		if p.Security != "" {
			params = append(params, "security="+p.Security)
		}
		if len(params) > 0 {
			u += "?" + strings.Join(params, "&")
		}
		if p.Name != "" {
			u += "#" + p.Name
		}
		return u
	case "hysteria2", "hy2":
		u := fmt.Sprintf("hysteria2://%s@%s:%d", p.Password, p.Server, p.Port)
		params := []string{}
		if p.SNI != "" {
			params = append(params, "sni="+p.SNI)
		}
		if p.SkipCertVerify {
			params = append(params, "insecure=1")
		}
		if len(params) > 0 {
			u += "?" + strings.Join(params, "&")
		}
		if p.Name != "" {
			u += "#" + p.Name
		}
		return u
	case "tuic":
		u := fmt.Sprintf("tuic://%s@%s:%d", p.UUID, p.Server, p.Port)
		params := []string{}
		if p.Password != "" {
			params = append(params, "password="+p.Password)
		}
		if p.SNI != "" {
			params = append(params, "sni="+p.SNI)
		}
		if len(params) > 0 {
			u += "?" + strings.Join(params, "&")
		}
		if p.Name != "" {
			u += "#" + p.Name
		}
		return u
	case "socks5":
		u := fmt.Sprintf("socks5://%s:%d", p.Server, p.Port)
		if p.Username != "" {
			u = fmt.Sprintf("socks5://%s:%s@%s:%d", p.Username, p.Password, p.Server, p.Port)
		}
		if p.Name != "" {
			u += "#" + p.Name
		}
		return u
	case "http":
		u := fmt.Sprintf("http://%s:%d", p.Server, p.Port)
		if p.Username != "" {
			u = fmt.Sprintf("http://%s:%s@%s:%d", p.Username, p.Password, p.Server, p.Port)
		}
		if p.Name != "" {
			u += "#" + p.Name
		}
		return u
	default:
		return ""
	}
}