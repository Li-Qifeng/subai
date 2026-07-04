package converter

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/user/subai/internal/parser"
	"github.com/user/subai/internal/template"
)

func makeTestProxies() []parser.Proxy {
	return []parser.Proxy{
		{Name: "SS-Node", Type: "ss", Server: "1.1.1.1", Port: 443, Cipher: "aes-256-gcm", Password: "sspass"},
		{Name: "Trojan-Node", Type: "trojan", Server: "trojan.example.com", Port: 443, Password: "trojanpass", Network: "tcp", Security: "tls"},
		{Name: "VMess-Node", Type: "vmess", Server: "vmess.example.com", Port: 443, UUID: "550e8400-e29b-41d4-a716-446655440000", Network: "tcp"},
	}
}

func TestNewEngine(t *testing.T) {
	eng := New()
	if eng == nil {
		t.Fatal("New() returned nil")
	}
	if eng.tmplCfg == nil {
		t.Fatal("expected non-nil template config")
	}
}

func TestNewWithTemplate_Nil(t *testing.T) {
	eng := NewWithTemplate(nil)
	if eng == nil {
		t.Fatal("NewWithTemplate(nil) returned nil")
	}
}

func TestNewWithTemplate_Builtin(t *testing.T) {
	cfg := &template.Config{Template: "basic"}
	eng := NewWithTemplate(cfg)
	if eng == nil {
		t.Fatal("NewWithTemplate(basic) returned nil")
	}
}

func TestNewWithTemplate_Custom(t *testing.T) {
	cfg := &template.Config{
		ProxyGroups: []template.ProxyGroup{
			{Name: "Custom-Proxy", Type: "select", Proxies: []string{"Proxy"}},
		},
		RuleSets: []template.RuleSet{
			{Group: "Proxy", Rule: "MATCH,Proxy"},
		},
	}
	eng := NewWithTemplate(cfg)
	if eng == nil {
		t.Fatal("NewWithTemplate(custom) returned nil")
	}
}

func TestConvert_Clash(t *testing.T) {
	eng := New()
	proxies := makeTestProxies()

	data, err := eng.Convert(proxies, "clash")
	if err != nil {
		t.Fatalf("Convert clash failed: %v", err)
	}

	output := string(data)
	// Check basic structure
	if !strings.Contains(output, "port: 7890") {
		t.Error("missing port in output")
	}
	if !strings.Contains(output, "proxies:") {
		t.Error("missing proxies section")
	}
	if !strings.Contains(output, "proxy-groups:") {
		t.Error("missing proxy-groups section")
	}
	if !strings.Contains(output, "rules:") {
		t.Error("missing rules section")
	}
	if !strings.Contains(output, "SS-Node") {
		t.Error("missing SS-Node in output")
	}
	if !strings.Contains(output, "Trojan-Node") {
		t.Error("missing Trojan-Node in output")
	}
	if !strings.Contains(output, "VMess-Node") {
		t.Error("missing VMess-Node in output")
	}
}

func TestConvert_Base64(t *testing.T) {
	eng := New()
	proxies := makeTestProxies()

	data, err := eng.Convert(proxies, "base64")
	if err != nil {
		t.Fatalf("Convert base64 failed: %v", err)
	}

	// Decode to verify it's valid base64
	decoded, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		t.Fatalf("output is not valid base64: %v", err)
	}

	content := string(decoded)
	if !strings.Contains(content, "ss://") {
		t.Error("missing ss:// URI in base64 output")
	}
	if !strings.Contains(content, "trojan://") {
		t.Error("missing trojan:// URI in base64 output")
	}
	if !strings.Contains(content, "vmess://") {
		t.Error("missing vmess:// URI in base64 output")
	}
}

func TestConvert_Mixed(t *testing.T) {
	eng := New()
	proxies := makeTestProxies()

	data, err := eng.Convert(proxies, "mixed")
	if err != nil {
		t.Fatalf("Convert mixed failed: %v", err)
	}

	output := string(data)
	if !strings.Contains(output, "# Clash Config") {
		t.Error("missing Clash section header")
	}
	if !strings.Contains(output, "# Base64 Subscription") {
		t.Error("missing Base64 section header")
	}
	if !strings.Contains(output, "port: 7890") {
		t.Error("missing Clash config in mixed output")
	}
}

func TestConvert_UnknownTarget(t *testing.T) {
	eng := New()
	_, err := eng.Convert(nil, "unknown-format")
	if err == nil {
		t.Fatal("expected error for unknown target format")
	}
}

func TestConvert_EmptyProxies(t *testing.T) {
	eng := New()
	data, err := eng.Convert(nil, "clash")
	if err != nil {
		t.Fatalf("Convert empty proxies failed: %v", err)
	}
	output := string(data)
	if !strings.Contains(output, "port: 7890") {
		t.Error("missing basic config")
	}
}

func TestProxyToURI_SS(t *testing.T) {
	uri := proxyToURI(parser.Proxy{
		Name: "SS-Node", Type: "ss", Server: "1.1.1.1", Port: 443,
		Cipher: "aes-256-gcm", Password: "sspass",
	})
	if !strings.HasPrefix(uri, "ss://") {
		t.Errorf("expected ss:// prefix, got %q", uri)
	}
	if !strings.Contains(uri, "#SS-Node") {
		t.Errorf("expected name in fragment, got %q", uri)
	}
}

func TestProxyToURI_SSR(t *testing.T) {
	uri := proxyToURI(parser.Proxy{
		Name: "SSR-Node", Type: "ssr", Server: "ssr.example.com", Port: 1234,
		Protocol: "auth_aes128_md5", Cipher: "aes-256-cfb", Obfs: "http_simple",
		Password: "ssrpass",
	})
	if !strings.HasPrefix(uri, "ssr://") {
		t.Errorf("expected ssr:// prefix, got %q", uri)
	}
}

func TestProxyToURI_VMess(t *testing.T) {
	uri := proxyToURI(parser.Proxy{
		Name: "VMess-Node", Type: "vmess", Server: "vmess.example.com", Port: 443,
		UUID: "550e8400-e29b-41d4-a716-446655440000", Network: "tcp",
	})
	if !strings.HasPrefix(uri, "vmess://") {
		t.Errorf("expected vmess:// prefix, got %q", uri)
	}
}

func TestProxyToURI_VMess_WS(t *testing.T) {
	uri := proxyToURI(parser.Proxy{
		Name: "VMess-WS", Type: "vmess", Server: "ws.example.com", Port: 443,
		UUID: "550e8400-e29b-41d4-a716-446655440000", Network: "ws",
		WSPath: "/ws", SNI: "ws.example.com",
		Encryption: "auto",
		Security:   "tls",
	})
	if !strings.HasPrefix(uri, "vmess://") {
		t.Errorf("expected vmess:// prefix, got %q", uri)
	}
}

func TestProxyToURI_VLESS(t *testing.T) {
	uri := proxyToURI(parser.Proxy{
		Name: "VLESS-Node", Type: "vless", Server: "vless.example.com", Port: 443,
		UUID: "550e8400-e29b-41d4-a716-446655440000", Encryption: "none",
		Network: "tcp", Security: "reality", Flow: "xtls-rprx-vision",
		SNI: "www.example.com",
	})
	if !strings.HasPrefix(uri, "vless://") {
		t.Errorf("expected vless:// prefix, got %q", uri)
	}
	if !strings.Contains(uri, "encryption=none") {
		t.Errorf("expected encryption=none, got %q", uri)
	}
	if !strings.Contains(uri, "flow=xtls-rprx-vision") {
		t.Errorf("expected flow param, got %q", uri)
	}
	if !strings.Contains(uri, "security=reality") {
		t.Errorf("expected security=reality, got %q", uri)
	}
}

func TestProxyToURI_Trojan(t *testing.T) {
	uri := proxyToURI(parser.Proxy{
		Name: "Trojan-Node", Type: "trojan", Server: "trojan.example.com", Port: 443,
		Password: "trojanpass", Network: "tcp", Security: "tls", SNI: "trojan.example.com",
	})
	if !strings.HasPrefix(uri, "trojan://") {
		t.Errorf("expected trojan:// prefix, got %q", uri)
	}
	if !strings.Contains(uri, "type=tcp") {
		t.Errorf("expected type=tcp, got %q", uri)
	}
	if !strings.Contains(uri, "sni=trojan.example.com") {
		t.Errorf("expected sni param, got %q", uri)
	}
}

func TestProxyToURI_Hysteria2(t *testing.T) {
	uri := proxyToURI(parser.Proxy{
		Name: "Hy2-Node", Type: "hysteria2", Server: "hy2.example.com", Port: 8443,
		Password: "hy2pass", SNI: "hy2.example.com",
	})
	if !strings.HasPrefix(uri, "hysteria2://") {
		t.Errorf("expected hysteria2:// prefix, got %q", uri)
	}
}

func TestProxyToURI_TUIC(t *testing.T) {
	uri := proxyToURI(parser.Proxy{
		Name: "TUIC-Node", Type: "tuic", Server: "tuic.example.com", Port: 12345,
		UUID: "tuic-uuid", Password: "tuicpass", SNI: "tuic.example.com",
	})
	if !strings.HasPrefix(uri, "tuic://") {
		t.Errorf("expected tuic:// prefix, got %q", uri)
	}
	if !strings.Contains(uri, "password=tuicpass") {
		t.Errorf("expected password param, got %q", uri)
	}
}

func TestProxyToURI_SOCKS5(t *testing.T) {
	uri := proxyToURI(parser.Proxy{
		Name: "SOCKS-Node", Type: "socks5", Server: "socks.example.com", Port: 1080,
	})
	if !strings.HasPrefix(uri, "socks5://") {
		t.Errorf("expected socks5:// prefix, got %q", uri)
	}
}

func TestProxyToURI_SOCKS5_WithAuth(t *testing.T) {
	uri := proxyToURI(parser.Proxy{
		Name: "SOCKS-Auth", Type: "socks5", Server: "socks-auth.com", Port: 1080,
		Username: "user", Password: "pass",
	})
	if !strings.Contains(uri, "user:pass@") {
		t.Errorf("expected auth in URI, got %q", uri)
	}
}

func TestProxyToURI_HTTP(t *testing.T) {
	uri := proxyToURI(parser.Proxy{
		Name: "HTTP-Node", Type: "http", Server: "http.example.com", Port: 3128,
	})
	if !strings.HasPrefix(uri, "http://") {
		t.Errorf("expected http:// prefix, got %q", uri)
	}
}

func TestProxyToURI_HTTP_WithAuth(t *testing.T) {
	uri := proxyToURI(parser.Proxy{
		Name: "HTTP-Auth", Type: "http", Server: "http-auth.com", Port: 3128,
		Username: "user", Password: "pass",
	})
	if !strings.Contains(uri, "user:pass@") {
		t.Errorf("expected auth in URI, got %q", uri)
	}
}

func TestProxyToURI_Unknown(t *testing.T) {
	uri := proxyToURI(parser.Proxy{
		Name: "Unknown", Type: "unknown", Server: "x.com", Port: 1234,
	})
	if uri != "" {
		t.Errorf("expected empty string for unknown type, got %q", uri)
	}
}

func TestProxyToURI_SS_NoName(t *testing.T) {
	uri := proxyToURI(parser.Proxy{
		Type: "ss", Server: "1.1.1.1", Port: 443, Cipher: "aes-256-gcm", Password: "pass",
	})
	if !strings.HasPrefix(uri, "ss://") {
		t.Errorf("expected ss:// prefix, got %q", uri)
	}
	// No fragment when name is empty
	if strings.Contains(uri, "#") {
		t.Errorf("expected no fragment when name is empty, got %q", uri)
	}
}

func TestConvert_Clash_WithTemplate(t *testing.T) {
	cfg := &template.Config{Template: "loyalsoldier"}
	eng := NewWithTemplate(cfg)
	proxies := makeTestProxies()

	data, err := eng.Convert(proxies, "clash")
	if err != nil {
		t.Fatalf("Convert with template failed: %v", err)
	}

	output := string(data)
	// Loyalsoldier template should have more groups and rules
	if !strings.Contains(output, "proxy-groups:") {
		t.Error("missing proxy-groups")
	}
	if !strings.Contains(output, "rules:") {
		t.Error("missing rules")
	}
}

func TestConvert_Base64_Empty(t *testing.T) {
	eng := New()
	data, err := eng.Convert([]parser.Proxy{}, "base64")
	if err != nil {
		t.Fatalf("Convert base64 empty failed: %v", err)
	}
	// Empty proxy list should produce empty base64
	decoded, _ := base64.StdEncoding.DecodeString(string(data))
	if len(decoded) > 0 {
		t.Errorf("expected empty base64 output, got %q", string(decoded))
	}
}

func TestProxyToURI_VLESS_NoEncryption(t *testing.T) {
	uri := proxyToURI(parser.Proxy{
		Name: "VLESS", Type: "vless", Server: "vless.com", Port: 443,
		UUID: "uuid", Network: "tcp",
	})
	if !strings.Contains(uri, "encryption=none") {
		t.Errorf("expected encryption=none default, got %q", uri)
	}
}

func TestProxyToURI_VLESS_WithCustomEncryption(t *testing.T) {
	uri := proxyToURI(parser.Proxy{
		Name: "VLESS", Type: "vless", Server: "vless.com", Port: 443,
		UUID: "uuid", Encryption: "aes-128-gcm", Network: "tcp",
	})
	if !strings.Contains(uri, "encryption=aes-128-gcm") {
		t.Errorf("expected encryption=aes-128-gcm, got %q", uri)
	}
}

func TestProxyToURI_VLESS_WithRealityParams(t *testing.T) {
	uri := proxyToURI(parser.Proxy{
		Name: "Reality", Type: "vless", Server: "reality.com", Port: 443,
		UUID: "uuid", Encryption: "none", Network: "tcp",
		Security: "reality", Fingerprint: "chrome", PublicKey: "pbk123",
		ShortID: "sid123", Flow: "xtls-rprx-vision",
	})
	if !strings.Contains(uri, "fp=chrome") {
		t.Errorf("expected fp=chrome, got %q", uri)
	}
	if !strings.Contains(uri, "pbk=pbk123") {
		t.Errorf("expected pbk param, got %q", uri)
	}
	if !strings.Contains(uri, "sid=sid123") {
		t.Errorf("expected sid param, got %q", uri)
	}
}

func TestProxyToURI_Trojan_NoName(t *testing.T) {
	uri := proxyToURI(parser.Proxy{
		Type: "trojan", Server: "troj.com", Port: 443,
		Password: "pass", Network: "tcp", Security: "tls",
	})
	if !strings.Contains(uri, "trojan://pass@troj.com:443") {
		t.Errorf("unexpected trojan URI: %q", uri)
	}
}

func TestProxyToURI_Hysteria2_SkipCert(t *testing.T) {
	uri := proxyToURI(parser.Proxy{
		Name: "Hy2", Type: "hysteria2", Server: "hy.com", Port: 443,
		Password: "pass", SNI: "hy.com", SkipCertVerify: true,
	})
	if !strings.Contains(uri, "insecure=1") {
		t.Errorf("expected insecure=1, got %q", uri)
	}
}

func TestProxyToURI_TUIC_NoPassword(t *testing.T) {
	uri := proxyToURI(parser.Proxy{
		Name: "TUIC", Type: "tuic", Server: "tuic.com", Port: 443,
		UUID: "uuid",
	})
	if strings.Contains(uri, "password=") {
		t.Errorf("expected no password param, got %q", uri)
	}
}

func TestProxyToURI_SSR_WithParams(t *testing.T) {
	uri := proxyToURI(parser.Proxy{
		Name: "SSR", Type: "ssr", Server: "ssr.com", Port: 1234,
		Protocol: "auth_aes128_md5", Cipher: "aes-256-cfb", Obfs: "http_simple",
		Password: "pass", ObfsParam: "obfs123", ProtocolParam: "proto456",
	})
	if !strings.HasPrefix(uri, "ssr://") {
		t.Errorf("expected ssr:// prefix, got %q", uri)
	}
}

func TestProxyToURI_VMess_WithWS(t *testing.T) {
	uri := proxyToURI(parser.Proxy{
		Name: "VMess-WS", Type: "vmess", Server: "ws.com", Port: 443,
		UUID: "uuid", Network: "ws", WSPath: "/ws-path", SNI: "ws.com",
		Security: "tls",
	})
	if !strings.HasPrefix(uri, "vmess://") {
		t.Errorf("expected vmess:// prefix, got %q", uri)
	}
}

func TestProxyToURI_VMess_NoTLS(t *testing.T) {
	uri := proxyToURI(parser.Proxy{
		Name: "VMess", Type: "vmess", Server: "vmess.com", Port: 80,
		UUID: "uuid", Network: "tcp",
	})
	if strings.Contains(uri, "\"tls\":\"tls\"") {
		t.Errorf("expected no TLS, got %q", uri)
	}
}

func TestProxyToURI_SS_NoPassword(t *testing.T) {
	uri := proxyToURI(parser.Proxy{
		Type: "ss", Server: "1.1.1.1", Port: 443, Cipher: "aes-256-gcm",
	})
	if !strings.HasPrefix(uri, "ss://") {
		t.Errorf("expected ss:// prefix, got %q", uri)
	}
}

func TestProxyToURI_EmptyType(t *testing.T) {
	uri := proxyToURI(parser.Proxy{
		Type: "", Server: "x.com", Port: 80,
	})
	if uri != "" {
		t.Errorf("expected empty for empty type, got %q", uri)
	}
}