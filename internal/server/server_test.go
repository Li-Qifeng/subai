package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/user/subai/internal/config"
	"github.com/user/subai/internal/parser"
)

func writeTestConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "subai.yaml")
	os.WriteFile(path, []byte(content), 0644)
	return path
}

func TestNew(t *testing.T) {
	s := New(":9090", "test-token", "/path/to/config.yaml")
	if s == nil {
		t.Fatal("New returned nil")
	}
	if s.listen != ":9090" {
		t.Errorf("listen: got %q", s.listen)
	}
	if s.token != "test-token" {
		t.Errorf("token: got %q", s.token)
	}
	if s.configPath != "/path/to/config.yaml" {
		t.Errorf("configPath: got %q", s.configPath)
	}
}

func TestHandlers_Health(t *testing.T) {
	cfgPath := writeTestConfig(t, `sources: []`)
	s := New(":8080", "", cfgPath)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	s.handleHealth(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(string(body), `"status":"ok"`) {
		t.Errorf("unexpected body: %s", string(body))
	}
}

func TestHandlers_Health_WrongMethod(t *testing.T) {
	cfgPath := writeTestConfig(t, `sources: []`)
	s := New(":8080", "", cfgPath)

	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	w := httptest.NewRecorder()
	s.handleHealth(w, req)

	if w.Result().StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Result().StatusCode)
	}
}

func TestHandlers_Version(t *testing.T) {
	cfgPath := writeTestConfig(t, `sources: []`)
	s := New(":8080", "", cfgPath)

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	w := httptest.NewRecorder()
	s.handleVersion(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]string
	json.Unmarshal(body, &result)
	if result["version"] == "" {
		t.Errorf("expected non-empty version, got %v", result)
	}
}

func TestAuthMiddleware_NoToken(t *testing.T) {
	cfgPath := writeTestConfig(t, `sources: []`)
	s := New(":8080", "secret", cfgPath)

	req := httptest.NewRequest(http.MethodGet, "/sub", nil)
	w := httptest.NewRecorder()
	s.authMiddleware(s.handleSub)(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestAuthMiddleware_WithToken(t *testing.T) {
	cfgPath := writeTestConfig(t, `sources: []`)
	s := New(":8080", "secret", cfgPath)

	req := httptest.NewRequest(http.MethodGet, "/sub?token=secret", nil)
	w := httptest.NewRecorder()
	s.authMiddleware(s.handleSub)(w, req)

	resp := w.Result()
	if resp.StatusCode == http.StatusUnauthorized {
		t.Error("should have passed auth")
	}
}

func TestAuthMiddleware_WrongToken(t *testing.T) {
	cfgPath := writeTestConfig(t, `sources: []`)
	s := New(":8080", "secret", cfgPath)

	req := httptest.NewRequest(http.MethodGet, "/sub?token=wrong", nil)
	w := httptest.NewRecorder()
	s.authMiddleware(s.handleSub)(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestHandleSub_WrongMethod(t *testing.T) {
	cfgPath := writeTestConfig(t, `sources: []`)
	s := New(":8080", "", cfgPath)

	req := httptest.NewRequest(http.MethodPost, "/sub", nil)
	w := httptest.NewRecorder()
	s.handleSub(w, req)

	if w.Result().StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Result().StatusCode)
	}
}

func TestHandleRefresh_WrongMethod(t *testing.T) {
	cfgPath := writeTestConfig(t, `sources: []`)
	s := New(":8080", "", cfgPath)

	req := httptest.NewRequest(http.MethodGet, "/refresh", nil)
	w := httptest.NewRecorder()
	s.handleRefresh(w, req)

	if w.Result().StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Result().StatusCode)
	}
}

func TestHandleRefresh(t *testing.T) {
	cfgPath := writeTestConfig(t, `sources:
  - name: test
    url: https://example.com/sub
output:
  target: clash`)
	s := New(":8080", "", cfgPath)

	req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
	w := httptest.NewRecorder()
	s.handleRefresh(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestHandleRefresh_BadConfig(t *testing.T) {
	cfgPath := writeTestConfig(t, `invalid: yaml: [bad`)
	s := New(":8080", "", cfgPath)

	req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
	w := httptest.NewRecorder()
	s.handleRefresh(w, req)

	if w.Result().StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500 for bad config, got %d", w.Result().StatusCode)
	}
}

func TestFilterProxies_NoRules(t *testing.T) {
	proxies := parser.ProxyList{
		{Name: "HK Node"},
		{Name: "JP Node"},
	}

	result := filterProxies(proxies, config.Rules{})
	if len(result) != 2 {
		t.Fatalf("expected 2, got %d", len(result))
	}
}

func TestFilterProxies_Include(t *testing.T) {
	proxies := parser.ProxyList{
		{Name: "🇭🇰 HK Node"},
		{Name: "🇯🇵 JP Node"},
	}

	result := filterProxies(proxies, config.Rules{
		Include: []string{"HK"},
	})
	if len(result) != 1 {
		t.Fatalf("expected 1, got %d", len(result))
	}
}

func TestFilterProxies_Exclude(t *testing.T) {
	proxies := parser.ProxyList{
		{Name: "🇭🇰 HK Node"},
		{Name: "🇭🇰 HK 过期 Node"},
	}

	result := filterProxies(proxies, config.Rules{
		Exclude: []string{"过期"},
	})
	if len(result) != 1 {
		t.Fatalf("expected 1, got %d", len(result))
	}
}

func TestFilterProxies_IncludeAndExclude(t *testing.T) {
	proxies := parser.ProxyList{
		{Name: "🇭🇰 HK Node"},
		{Name: "🇭🇰 HK 过期 Node"},
		{Name: "🇯🇵 JP Node"},
		{Name: "🇯🇵 JP 剩余 Node"},
	}

	result := filterProxies(proxies, config.Rules{
		Include: []string{"HK", "JP"},
		Exclude: []string{"过期", "剩余"},
	})
	if len(result) != 2 {
		t.Fatalf("expected 2, got %d", len(result))
	}
}

func TestMatchAny(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		patterns []string
		want     bool
	}{
		{"match", "HK Node", []string{"HK"}, true},
		{"no match", "JP Node", []string{"HK"}, false},
		{"empty patterns", "HK Node", nil, false},
		{"multi patterns", "SG Node", []string{"HK", "SG"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchAny(tt.input, tt.patterns)
			if got != tt.want {
				t.Errorf("matchAny(%q, %v) = %v, want %v", tt.input, tt.patterns, got, tt.want)
			}
		})
	}
}

func TestRenderBase64(t *testing.T) {
	proxies := parser.ProxyList{
		{Name: "SS-Node", Type: "ss", Server: "1.1.1.1", Port: 443, Cipher: "aes-256-gcm", Password: "sspass"},
	}

	w := httptest.NewRecorder()
	s := &Server{}
	s.renderBase64(w, proxies)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if len(body) == 0 {
		t.Error("expected non-empty body")
	}
}

func TestRenderClash(t *testing.T) {
	cfg := &config.Config{
		Output: config.Output{Target: "clash", Pretty: true},
	}
	proxies := parser.ProxyList{
		{Name: "SS-Node", Type: "ss", Server: "1.1.1.1", Port: 443, Cipher: "aes-256-gcm", Password: "sspass"},
	}

	w := httptest.NewRecorder()
	s := &Server{}
	s.renderClash(w, cfg, proxies)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := yaml.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid YAML output: %v", err)
	}
	if result["proxies"] == nil {
		t.Error("missing proxies in output")
	}
	if result["proxy-groups"] == nil {
		t.Error("missing proxy-groups in output")
	}
	if result["port"] != 7890 {
		t.Errorf("expected port 7890, got %v", result["port"])
	}
}

func TestRenderClash_EmptyProxies(t *testing.T) {
	cfg := &config.Config{
		Output: config.Output{Target: "clash", Pretty: true},
	}

	w := httptest.NewRecorder()
	s := &Server{}
	s.renderClash(w, cfg, parser.ProxyList{})

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if strings.Contains(string(body), "proxy-groups:") {
		t.Error("expected no proxy-groups for empty proxies")
	}
}

func TestRenderBase64_Empty(t *testing.T) {
	w := httptest.NewRecorder()
	s := &Server{}
	s.renderBase64(w, parser.ProxyList{})

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	// Empty proxy list should produce empty base64 (encoding of empty string)
	if len(body) != 0 {
		t.Errorf("expected empty body for empty proxies, got %d bytes", len(body))
	}
}

func TestHandleSub_NoSources(t *testing.T) {
	cfgPath := writeTestConfig(t, `sources: []
output:
  target: clash`)
	s := New(":8080", "", cfgPath)

	req := httptest.NewRequest(http.MethodGet, "/sub", nil)
	w := httptest.NewRecorder()
	s.handleSub(w, req)

	resp := w.Result()
	// Should succeed with empty output (no proxies to render)
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusInternalServerError {
		t.Errorf("unexpected 500: %s", string(body))
	}
}

func TestProxyToURI_SS(t *testing.T) {
	uri := proxyToURI(parser.Proxy{
		Name: "Test-Node", Type: "ss", Server: "1.1.1.1", Port: 443,
		Cipher: "aes-256-gcm", Password: "pass",
	})
	if !strings.HasPrefix(uri, "ss://") {
		t.Errorf("expected ss:// prefix, got %q", uri)
	}
}

func TestProxyToURI_VMess(t *testing.T) {
	uri := proxyToURI(parser.Proxy{
		Name: "VMess", Type: "vmess", Server: "vmess.com", Port: 443,
		UUID: "uuid", Network: "tcp",
	})
	if !strings.HasPrefix(uri, "vmess://") {
		t.Errorf("expected vmess:// prefix, got %q", uri)
	}
}

func TestProxyToURI_Trojan(t *testing.T) {
	uri := proxyToURI(parser.Proxy{
		Name: "Trojan", Type: "trojan", Server: "trojan.com", Port: 443,
		Password: "pass", Network: "tcp",
	})
	if !strings.HasPrefix(uri, "trojan://") {
		t.Errorf("expected trojan:// prefix, got %q", uri)
	}
}

func TestProxyToURI_Unknown(t *testing.T) {
	uri := proxyToURI(parser.Proxy{
		Type: "unknown", Server: "x.com", Port: 80,
	})
	if uri != "" {
		t.Errorf("expected empty for unknown type, got %q", uri)
	}
}

func TestProxyToURI_SSR(t *testing.T) {
	uri := proxyToURI(parser.Proxy{
		Name: "SSR", Type: "ssr", Server: "ssr.com", Port: 1234,
		Protocol: "auth_aes128_md5", Cipher: "aes-256-cfb", Obfs: "http_simple",
		Password: "pass",
	})
	if !strings.HasPrefix(uri, "ssr://") {
		t.Errorf("expected ssr:// prefix, got %q", uri)
	}
}

func TestProxyToURI_VLESS(t *testing.T) {
	uri := proxyToURI(parser.Proxy{
		Name: "VLESS", Type: "vless", Server: "vless.com", Port: 443,
		UUID: "uuid", Encryption: "none", Network: "tcp",
	})
	if !strings.HasPrefix(uri, "vless://") {
		t.Errorf("expected vless:// prefix, got %q", uri)
	}
}

func TestProxyToURI_Hysteria2(t *testing.T) {
	uri := proxyToURI(parser.Proxy{
		Name: "Hy2", Type: "hysteria2", Server: "hy.com", Port: 443,
		Password: "pass",
	})
	if !strings.HasPrefix(uri, "hysteria2://") {
		t.Errorf("expected hysteria2:// prefix, got %q", uri)
	}
}

func TestProxyToURI_TUIC(t *testing.T) {
	uri := proxyToURI(parser.Proxy{
		Name: "TUIC", Type: "tuic", Server: "tuic.com", Port: 443,
		UUID: "uuid",
	})
	if !strings.HasPrefix(uri, "tuic://") {
		t.Errorf("expected tuic:// prefix, got %q", uri)
	}
}

func TestProxyToURI_SOCKS5(t *testing.T) {
	uri := proxyToURI(parser.Proxy{
		Name: "SOCKS", Type: "socks5", Server: "socks.com", Port: 1080,
	})
	if !strings.HasPrefix(uri, "socks5://") {
		t.Errorf("expected socks5:// prefix, got %q", uri)
	}
}

func TestProxyToURI_HTTP(t *testing.T) {
	uri := proxyToURI(parser.Proxy{
		Name: "HTTP", Type: "http", Server: "http.com", Port: 3128,
	})
	if !strings.HasPrefix(uri, "http://") {
		t.Errorf("expected http:// prefix, got %q", uri)
	}
}