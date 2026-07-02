package parser

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ParseURI parses a single proxy URI string into a Proxy struct.
func ParseURI(uri string) (*Proxy, error) {
	if strings.HasPrefix(uri, "ss://") {
		return parseSS(uri)
	}
	if strings.HasPrefix(uri, "ssr://") {
		return parseSSR(uri)
	}
	if strings.HasPrefix(uri, "vmess://") {
		return parseVMess(uri)
	}
	if strings.HasPrefix(uri, "vless://") {
		return parseVLESS(uri)
	}
	if strings.HasPrefix(uri, "trojan://") {
		return parseTrojan(uri)
	}
	if strings.HasPrefix(uri, "hysteria2://") || strings.HasPrefix(uri, "hy2://") {
		return parseHysteria2(uri)
	}
	if strings.HasPrefix(uri, "tuic://") {
		return parseTUIC(uri)
	}
	if strings.HasPrefix(uri, "ssd://") {
		return parseSSD(uri)
	}
	if strings.HasPrefix(uri, "socks5://") || strings.HasPrefix(uri, "http://") || strings.HasPrefix(uri, "https://") {
		return parseSocksHTTP(uri)
	}
	return nil, fmt.Errorf("unknown URI scheme: %.20s", uri)
}

// parseSS parses shadowsocks URI: ss://method:password@server:port#name (legacy)
// or ss://base64(method:password)@server:port#name (SIP002)
func parseSS(uri string) (*Proxy, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("parse ss uri: %w", err)
	}

	p := &Proxy{
		Type:   "ss",
		Server: u.Hostname(),
	}

	portStr := u.Port()
	if portStr != "" {
		p.Port, _ = strconv.Atoi(portStr)
	}

	// SIP002: the user-info is base64(method:password)
	if u.User != nil {
		userInfo := u.User.String()
		// Try base64 decode first (SIP002)
		decoded, err := base64Decode(userInfo)
		if err == nil {
			parts := strings.SplitN(decoded, ":", 2)
			if len(parts) == 2 {
				p.Cipher = parts[0]
				p.Password = parts[1]
			} else {
				p.Cipher = parts[0]
			}
		} else {
			// Legacy format: method:password@host
			p.Cipher = u.User.Username()
			p.Password, _ = u.User.Password()
		}
	}

	// URL-decode the fragment (emojis etc.)
	if u.Fragment != "" {
		if decoded, err := url.QueryUnescape(u.Fragment); err == nil {
			p.Name = decoded
		} else {
			p.Name = u.Fragment
		}
	}
	if p.Name == "" {
		p.Name = u.Host
	}

	// SIP002 plugin support
	plugin := u.Query().Get("plugin")
	if plugin != "" {
		parts := strings.SplitN(plugin, ";", 2)
		if len(parts) > 0 {
			p.Obfs = parts[0]
			if len(parts) > 1 {
				p.ObfsParam = parts[1]
			}
		}
	}

	return p, nil
}

// parseSSR parses shadowsocksr URI
func parseSSR(uri string) (*Proxy, error) {
	// SSR format: ssr://base64(server:port:protocol:method:obfs:base64pass/?params)
	raw := strings.TrimPrefix(uri, "ssr://")
	decoded, err := base64Decode(raw)
	if err != nil {
		return nil, fmt.Errorf("decode ssr: %w", err)
	}

	// Remove trailing /? if present
	decoded = strings.TrimSuffix(decoded, "/")
	decoded = strings.TrimSuffix(decoded, "?")

	parts := strings.SplitN(decoded, "/?", 2)
	mainPart := parts[0]

	fields := strings.Split(mainPart, ":")
	if len(fields) < 6 {
		return nil, fmt.Errorf("invalid ssr: not enough fields (%d)", len(fields))
	}

	p := &Proxy{
		Type:     "ssr",
		Server:   fields[0],
		Protocol: fields[2],
		Cipher:   fields[3],
		Obfs:     fields[4],
	}

	p.Port, _ = strconv.Atoi(fields[1])

	// Password is base64 encoded
	passDecoded, err := base64Decode(fields[5])
	if err == nil {
		p.Password = passDecoded
	} else {
		p.Password = fields[5]
	}

	// Parse query params
	if len(parts) > 1 {
		queryStr := parts[1]
		// Remove trailing slashes
		queryStr = strings.TrimRight(queryStr, "/")
		params := strings.Split(queryStr, "&")
		for _, param := range params {
			kv := strings.SplitN(param, "=", 2)
			if len(kv) != 2 {
				continue
			}
			val, _ := base64Decode(kv[1])
			switch kv[0] {
			case "obfsparam":
				p.ObfsParam = val
			case "protoparam":
				p.ProtocolParam = val
			case "remarks":
				p.Name = val
			case "group":
				p.Group = val
			}
		}
	}

	if p.Name == "" {
		p.Name = p.Server
	}

	return p, nil
}

// vmessJSON is the internal structure of a VMess share link.
type vmessJSON struct {
	Add  string `json:"add"`
	Aid  string `json:"aid"`
	Host string `json:"host"`
	ID   string `json:"id"`
	Net  string `json:"net"`
	Path string `json:"path"`
	Port string `json:"port"`
	PS   string `json:"ps"`
	TLS  string `json:"tls"`
	Type string `json:"type"`
	V    string `json:"v"`
	Scy  string `json:"scy"` // security
	SNI  string `json:"sni"`
}

// parseVMess parses vmess://base64(json) URI
func parseVMess(uri string) (*Proxy, error) {
	raw := strings.TrimPrefix(uri, "vmess://")

	// Try to decode as base64 JSON
	decoded, err := base64Decode(raw)
	if err != nil {
		return nil, fmt.Errorf("decode vmess: %w", err)
	}

	var vmj vmessJSON
	if err := json.Unmarshal([]byte(decoded), &vmj); err != nil {
		return nil, fmt.Errorf("parse vmess json: %w", err)
	}

	p := &Proxy{
		Type:    "vmess",
		Name:    vmj.PS,
		Server:  vmj.Add,
		Network: vmj.Net,
		WSPath:  vmj.Path,
		UUID:    vmj.ID,
	}

	if vmj.Port != "" {
		p.Port, _ = strconv.Atoi(vmj.Port)
	}

	// Security encryption
	if vmj.Scy != "" {
		p.Encryption = vmj.Scy
	}

	// TLS
	if vmj.TLS == "tls" {
		p.SNI = vmj.Host
	}

	// WS Headers
	if vmj.Host != "" && vmj.Net == "ws" {
		p.WSHeaders = map[string]string{"Host": vmj.Host}
		p.SNI = vmj.Host
	}

	if p.Name == "" {
		p.Name = p.Server
	}

	return p, nil
}

// parseVLESS parses vless:// URI
func parseVLESS(uri string) (*Proxy, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("parse vless uri: %w", err)
	}

	p := &Proxy{
		Type:   "vless",
		Server: u.Hostname(),
	}

	if u.Port() != "" {
		p.Port, _ = strconv.Atoi(u.Port())
	}

	if u.User != nil {
		p.UUID = u.User.Username()
	}

	query := u.Query()
	p.Encryption = query.Get("encryption")
	p.Network = query.Get("type")
	if p.Network == "" {
		p.Network = "tcp"
	}
	p.Security = query.Get("security")
	p.Flow = query.Get("flow")
	p.SNI = query.Get("sni")
	p.Fingerprint = query.Get("fp")
	p.PublicKey = query.Get("pbk")
	p.ShortID = query.Get("sid")

	if p.Network == "ws" {
		p.WSPath = query.Get("path")
		host := query.Get("host")
		if host != "" {
			p.WSHeaders = map[string]string{"Host": host}
		}
	}

	if u.Fragment != "" {
		p.Name = u.Fragment
	} else {
		p.Name = p.Server
	}

	return p, nil
}

// parseTrojan parses trojan:// URI
func parseTrojan(uri string) (*Proxy, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("parse trojan uri: %w", err)
	}

	p := &Proxy{
		Type:   "trojan",
		Server: u.Hostname(),
	}

	if u.Port() != "" {
		p.Port, _ = strconv.Atoi(u.Port())
	}

	if u.User != nil {
		p.Password = u.User.Username()
	}

	query := u.Query()
	p.SNI = query.Get("sni")
	p.Fingerprint = query.Get("fp")
	p.Network = query.Get("type")
	if p.Network == "" {
		p.Network = "tcp"
	}

	if sec := query.Get("security"); sec != "" {
		p.Security = sec
	} else {
		p.Security = "tls"
	}

	if p.Network == "ws" {
		p.WSPath = query.Get("path")
		host := query.Get("host")
		if host != "" {
			p.WSHeaders = map[string]string{"Host": host}
		}
	}

	if u.Fragment != "" {
		p.Name = u.Fragment
	} else {
		p.Name = p.Server
	}

	return p, nil
}

// parseHysteria2 parses hysteria2:// or hy2:// URI
func parseHysteria2(uri string) (*Proxy, error) {
	uri = strings.Replace(uri, "hy2://", "hysteria2://", 1)
	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("parse hysteria2 uri: %w", err)
	}

	p := &Proxy{
		Type:   "hysteria2",
		Server: u.Hostname(),
	}

	if u.Port() != "" {
		p.Port, _ = strconv.Atoi(u.Port())
	}

	if u.User != nil {
		p.Password = u.User.Username()
	}

	query := u.Query()
	p.SNI = query.Get("sni")
	if query.Get("insecure") == "1" || query.Get("insecure") == "true" {
		p.SkipCertVerify = true
	}

	if u.Fragment != "" {
		p.Name = u.Fragment
	} else {
		p.Name = p.Server
	}

	return p, nil
}

// parseTUIC parses tuic:// URI
func parseTUIC(uri string) (*Proxy, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("parse tuic uri: %w", err)
	}

	p := &Proxy{
		Type:   "tuic",
		Server: u.Hostname(),
	}

	if u.Port() != "" {
		p.Port, _ = strconv.Atoi(u.Port())
	}

	if u.User != nil {
		p.UUID = u.User.Username()
	}

	query := u.Query()
	p.Password = query.Get("password")
	p.SNI = query.Get("sni")

	if u.Fragment != "" {
		p.Name = u.Fragment
	} else {
		p.Name = p.Server
	}

	return p, nil
}

// parseSSD parses SSD:// subscription
func parseSSD(uri string) (*Proxy, error) {
	// SSD is a complex nested JSON format, minimum support
	raw := strings.TrimPrefix(uri, "ssd://")
	decoded, err := base64Decode(raw)
	if err != nil {
		return nil, fmt.Errorf("decode ssd: %w", err)
	}

	// Return a minimal placeholder
	p := &Proxy{
		Type:   "ss",
		Name:   "ssd-node",
	}
	
	// Try to extract basic info from JSON
	var ssdData map[string]interface{}
	if err := json.Unmarshal([]byte(decoded), &ssdData); err == nil {
		if servers, ok := ssdData["servers"].([]interface{}); ok && len(servers) > 0 {
			if first, ok := servers[0].(map[string]interface{}); ok {
				if s, ok := first["server"].(string); ok {
					p.Server = s
				}
				if remark, ok := first["remarks"].(string); ok {
					p.Name = remark
				}
				if port, ok := first["port"].(float64); ok {
					p.Port = int(port)
				}
				if method, ok := first["method"].(string); ok {
					p.Cipher = method
				}
				if pass, ok := first["password"].(string); ok {
					p.Password = pass
				}
				// Get global encryption if not set per server
				if p.Cipher == "" {
					if encryption, ok := ssdData["encryption"].(string); ok {
						p.Cipher = encryption
					}
				}
				// Get global password if not set per server
				if p.Password == "" {
					if pwd, ok := ssdData["password"].(string); ok {
						p.Password = pwd
					}
				}
			}
		}
	}

	return p, nil
}

// parseSocksHTTP parses socks5/http proxy URIs
func parseSocksHTTP(uri string) (*Proxy, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("parse socks/http uri: %w", err)
	}

	proxyType := "http"
	if strings.HasPrefix(uri, "socks5://") {
		proxyType = "socks5"
	}

	p := &Proxy{
		Type:   proxyType,
		Server: u.Hostname(),
	}

	if u.Port() != "" {
		p.Port, _ = strconv.Atoi(u.Port())
	}

	if u.User != nil {
		p.Username = u.User.Username()
		p.Password, _ = u.User.Password()
	}

	if u.Fragment != "" {
		p.Name = u.Fragment
	} else {
		p.Name = p.Server
	}

	return p, nil
}

// Ensure extra field for socks/http username
func init() {
	// Patch types.go to add Username field if needed - handled via Proxy struct
}

var _ = base64.StdEncoding // ensure import