package parser

import (
	"encoding/base64"
	"testing"
)

func TestParseSubscription_Base64(t *testing.T) {
	uris := "ss://YWVzLTI1Ni1nY206cGFzc3dkQDEuMS4xLjE6NDQz#Node1\nss://YWVzLTI1Ni1nY206cGFzczJAMi4yLjIuMjo0NDM#Node2"
	encoded := base64.StdEncoding.EncodeToString([]byte(uris))

	result, err := ParseSubscription([]byte(encoded))
	if err != nil {
		t.Fatalf("ParseSubscription failed: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 URIs, got %d", len(result))
	}
}

func TestParseSubscription_PlainText(t *testing.T) {
	data := "ss://YWVzLTI1Ni1nY206cGFzc3dkQDEuMS4xLjE6NDQz#Node1\nvmess://abc123\n"
	result, err := ParseSubscription([]byte(data))
	if err != nil {
		t.Fatalf("ParseSubscription failed: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 URIs, got %d", len(result))
	}
}

func TestParseSubscription_SkipComments(t *testing.T) {
	data := "# this is a comment\n// this is also a comment\nss://YWVzLTI1Ni1nY206cGFzc3dkQDEuMS4xLjE6NDQz#Node1\n"
	result, err := ParseSubscription([]byte(data))
	if err != nil {
		t.Fatalf("ParseSubscription failed: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 URI (comments skipped), got %d", len(result))
	}
}

func TestParseSubscription_EmptyLines(t *testing.T) {
	data := "ss://YWVzLTI1Ni1nY206cGFzc3dkQDEuMS4xLjE6NDQz#Node1\n\n\nss://YWVzLTI1Ni1nY206cGFzc3dkQDIuMi4yLjI6NDQz#Node2"
	result, err := ParseSubscription([]byte(data))
	if err != nil {
		t.Fatalf("ParseSubscription failed: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 URIs, got %d", len(result))
	}
}

func TestParseSubscription_Empty(t *testing.T) {
	result, err := ParseSubscription([]byte(""))
	if err != nil {
		t.Fatalf("ParseSubscription failed: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected 0 URIs, got %d", len(result))
	}
}

func TestBase64Decode_Standard(t *testing.T) {
	decoded, err := base64Decode(base64.StdEncoding.EncodeToString([]byte("hello world")))
	if err != nil {
		t.Fatalf("base64Decode failed: %v", err)
	}
	if decoded != "hello world" {
		t.Errorf("got %q, want %q", decoded, "hello world")
	}
}

func TestBase64Decode_URLSafe(t *testing.T) {
	decoded, err := base64Decode(base64.URLEncoding.EncodeToString([]byte("url-safe-data")))
	if err != nil {
		t.Fatalf("base64Decode(URLSafe) failed: %v", err)
	}
	if decoded != "url-safe-data" {
		t.Errorf("got %q", decoded)
	}
}

func TestBase64Decode_NoPadding(t *testing.T) {
	raw := base64.RawStdEncoding.EncodeToString([]byte("no-padding-data"))
	decoded, err := base64Decode(raw)
	if err != nil {
		t.Fatalf("base64Decode(no-padding) failed: %v", err)
	}
	if decoded != "no-padding-data" {
		t.Errorf("got %q", decoded)
	}
}

func TestBase64Decode_Invalid(t *testing.T) {
	_, err := base64Decode("!!!invalid base64!!!")
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestIsSubscriptionFormat(t *testing.T) {
	tests := []struct {
		name string
		data string
		want bool
	}{
		{
			name: "base64 encoded subscription",
			data: base64.StdEncoding.EncodeToString([]byte("ss://YWVzLTI1Ni1nY206cGFzc3dkQDEuMS4xLjE6NDQz#node1\nss://YWVzLTI1Ni1nY206cGFzc3dkQDIuMi4yLjI6NDQz#node2\nss://YWVzLTI1Ni1nY206cGFzc3dkQDMuMy4zLjM6NDQz#node3")),
			want: true,
		},
		{
			name: "plain text too short",
			data: "hello",
			want: false,
		},
		{
			name: "JSON content",
			data: base64.StdEncoding.EncodeToString([]byte(`{"v":"2","ps":"test"}`)),
			want: false,
		},
		{
			name: "empty",
			data: "",
			want: false,
		},
		{
			name: "clash YAML content",
			data: base64.StdEncoding.EncodeToString([]byte("proxies:\n  - name: test")),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsSubscriptionFormat([]byte(tt.data))
			if got != tt.want {
				t.Errorf("IsSubscriptionFormat: got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseAuto_ClashYAML(t *testing.T) {
	yaml := `proxies:
  - name: "🇭🇰 HK Node"
    type: ss
    server: 1.1.1.1
    port: 443
    cipher: aes-256-gcm
    password: testpass
  - name: "🇯🇵 JP Node"
    type: trojan
    server: 2.2.2.2
    port: 443
    password: trojanpass
    network: tcp
    security: tls`

	proxies, err := ParseAuto([]byte(yaml))
	if err != nil {
		t.Fatalf("ParseAuto ClashYAML failed: %v", err)
	}
	if len(proxies) != 2 {
		t.Fatalf("expected 2 proxies, got %d", len(proxies))
	}
	if proxies[0].Type != "ss" || proxies[0].Server != "1.1.1.1" {
		t.Errorf("proxy[0] mismatch: %+v", proxies[0])
	}
	if proxies[1].Type != "trojan" || proxies[1].Server != "2.2.2.2" {
		t.Errorf("proxy[1] mismatch: %+v", proxies[1])
	}
}

func TestParseAuto_ClashProxyList(t *testing.T) {
	yaml := `- name: "Node1"
  type: ss
  server: 1.1.1.1
  port: 443
  cipher: aes-256-gcm
  password: pass1
- name: "Node2"
  type: vmess
  server: 2.2.2.2
  port: 443
  uuid: 550e8400-e29b-41d4-a716-446655440000
  network: ws`

	proxies, err := ParseAuto([]byte(yaml))
	if err != nil {
		t.Fatalf("ParseAuto proxy list failed: %v", err)
	}
	if len(proxies) != 2 {
		t.Fatalf("expected 2 proxies, got %d", len(proxies))
	}
}

func TestParseAuto_Base64Subscription(t *testing.T) {
	uris := "ss://YWVzLTI1Ni1nY206cGFzc3dkQDEuMS4xLjE6NDQz#Node1"
	encoded := base64.StdEncoding.EncodeToString([]byte(uris))

	proxies, err := ParseAuto([]byte(encoded))
	if err != nil {
		t.Fatalf("ParseAuto base64 failed: %v", err)
	}
	if len(proxies) != 1 {
		t.Fatalf("expected 1 proxy, got %d", len(proxies))
	}
	if proxies[0].Name != "Node1" {
		t.Errorf("name: got %q", proxies[0].Name)
	}
}

func TestParseAuto_SingleURI(t *testing.T) {
	uri := "ss://YWVzLTI1Ni1nY206cGFzc3dkQDEuMS4xLjE6NDQz#DirectNode"

	proxies, err := ParseAuto([]byte(uri))
	if err != nil {
		t.Fatalf("ParseAuto single URI failed: %v", err)
	}
	if len(proxies) != 1 {
		t.Fatalf("expected 1 proxy, got %d", len(proxies))
	}
	if proxies[0].Name != "DirectNode" {
		t.Errorf("name: got %q", proxies[0].Name)
	}
}

func TestParseAuto_Unrecognized(t *testing.T) {
	_, err := ParseAuto([]byte("some random text that is not a valid format"))
	if err == nil {
		t.Fatal("expected error for unrecognized format")
	}
}

func TestParseAuto_Empty(t *testing.T) {
	_, err := ParseAuto([]byte(""))
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestParseClashYAML(t *testing.T) {
	yaml := `port: 7890
socks-port: 7891
mode: Rule
proxies:
  - name: "SS-Node"
    type: ss
    server: ss.example.com
    port: 443
    cipher: aes-256-gcm
    password: sspass`

	proxies, err := ParseClashYAML([]byte(yaml))
	if err != nil {
		t.Fatalf("ParseClashYAML failed: %v", err)
	}
	if len(proxies) != 1 {
		t.Fatalf("expected 1 proxy, got %d", len(proxies))
	}
	if proxies[0].Name != "SS-Node" {
		t.Errorf("name: got %q", proxies[0].Name)
	}
}

func TestParseClashYAML_NoProxies(t *testing.T) {
	yaml := `port: 7890
mode: Rule`

	proxies, err := ParseClashYAML([]byte(yaml))
	if err != nil {
		t.Fatalf("ParseClashYAML failed: %v", err)
	}
	if len(proxies) != 0 {
		t.Fatalf("expected 0 proxies, got %d", len(proxies))
	}
}

func TestParseClashYAML_Invalid(t *testing.T) {
	_, err := ParseClashYAML([]byte("invalid: yaml: ["))
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestParseClashProxyList(t *testing.T) {
	yaml := `- name: "Node1"
  type: ss
  server: 1.1.1.1
  port: 443
  cipher: aes-256-gcm
  password: pass1`

	proxies, err := ParseClashProxyList([]byte(yaml))
	if err != nil {
		t.Fatalf("ParseClashProxyList failed: %v", err)
	}
	if len(proxies) != 1 {
		t.Fatalf("expected 1 proxy, got %d", len(proxies))
	}
}

func TestParseClashProxyList_Invalid(t *testing.T) {
	_, err := ParseClashProxyList([]byte("invalid"))
	if err == nil {
		t.Fatal("expected error for invalid proxy list")
	}
}

func TestBase64RawDecode(t *testing.T) {
	encoded := base64.RawURLEncoding.EncodeToString([]byte("raw-data"))
	decoded, err := base64RawDecode(encoded)
	if err != nil {
		t.Fatalf("base64RawDecode failed: %v", err)
	}
	if decoded != "raw-data" {
		t.Errorf("got %q", decoded)
	}
}

func TestBase64RawDecode_Invalid(t *testing.T) {
	_, err := base64RawDecode("!!!")
	if err == nil {
		t.Fatal("expected error for invalid raw base64")
	}
}

func TestParseAuto_MixedFormats(t *testing.T) {
	// Multiple formats should be handled
	proxies, err := ParseAuto([]byte("ss://YWVzLTI1Ni1nY206cGFzc3dkQDEuMS4xLjE6NDQz#AB"))
	if err != nil {
		t.Fatalf("ParseAuto failed: %v", err)
	}
	if len(proxies) != 1 {
		t.Fatalf("expected 1 proxy, got %d", len(proxies))
	}
}

func TestParseURI_SS_Base64UserInfo(t *testing.T) {
	// SIP002 format: ss://base64(method:password)@server:port#name
	userInfo := base64.RawStdEncoding.EncodeToString([]byte("aes-256-gcm:password123"))
	uri := "ss://" + userInfo + "@example.com:443#MyNode"
	p, err := ParseURI(uri)
	if err != nil {
		t.Fatalf("ParseURI failed: %v", err)
	}
	if p.Cipher != "aes-256-gcm" || p.Password != "password123" {
		t.Errorf("cipher/password: %q/%q", p.Cipher, p.Password)
	}
}

func TestParseURI_SSR_NoPassword(t *testing.T) {
	mainPart := "server.com:1234:auth_aes128_md5:aes-256-cfb:http_simple:"
	ssrURI := "ssr://" + base64.RawURLEncoding.EncodeToString([]byte(mainPart))
	p, err := ParseURI(ssrURI)
	if err != nil {
		t.Fatalf("ParseURI SSR failed: %v", err)
	}
	if p.Name != "server.com" {
		t.Errorf("name should fallback to server, got %q", p.Name)
	}
}

func TestParseURI_VMess_NoName(t *testing.T) {
	vmessJSON := `{"v":"2","add":"vmess.example.com","port":"443","id":"550e8400-e29b-41d4-a716-446655440000","aid":"0","net":"tcp"}`
	vmessB64 := base64.RawURLEncoding.EncodeToString([]byte(vmessJSON))
	uri := "vmess://" + vmessB64

	p, err := ParseURI(uri)
	if err != nil {
		t.Fatalf("ParseURI failed: %v", err)
	}
	if p.Name != "vmess.example.com" {
		t.Errorf("expected name fallback to server, got %q", p.Name)
	}
}

func TestParseURI_VLESS_NoName(t *testing.T) {
	uri := "vless://uuid@vless.example.com:443?encryption=none&type=tcp"
	p, err := ParseURI(uri)
	if err != nil {
		t.Fatalf("ParseURI failed: %v", err)
	}
	if p.Name != "vless.example.com" {
		t.Errorf("expected name fallback to server, got %q", p.Name)
	}
}

func TestParseURI_Trojan_NoName(t *testing.T) {
	uri := "trojan://pass@trojan.example.com:443"
	p, err := ParseURI(uri)
	if err != nil {
		t.Fatalf("ParseURI failed: %v", err)
	}
	if p.Name != "trojan.example.com" {
		t.Errorf("expected name fallback to server, got %q", p.Name)
	}
}

func TestContainsLine(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   bool
	}{
		{"hello\nworld\nproxies:\n  - name: test", "proxies:", true},
		{"hello\nworld", "proxies:", false},
		{"- name: test", "- name:", true},
		{"", "anything", false},
	}

	for _, tt := range tests {
		got := containsLine(tt.s, tt.substr)
		if got != tt.want {
			t.Errorf("containsLine(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
		}
	}
}

func TestParseURI_SSR_WithGroup(t *testing.T) {
	// All params inside base64
	passB64 := base64.RawURLEncoding.EncodeToString([]byte("testpass"))
	remarksB64 := base64.RawURLEncoding.EncodeToString([]byte("SSR-Node"))
	groupB64 := base64.RawURLEncoding.EncodeToString([]byte("My-Group"))
	innerPart := "srv.com:1234:auth_aes128_md5:aes-256-cfb:tls1.2_ticket_auth:" + passB64 +
		"/?remarks=" + remarksB64 + "&group=" + groupB64
	ssrURI := "ssr://" + base64.RawURLEncoding.EncodeToString([]byte(innerPart))

	p, err := ParseURI(ssrURI)
	if err != nil {
		t.Fatalf("ParseURI failed: %v", err)
	}
	if p.Group != "My-Group" {
		t.Errorf("group: got %q", p.Group)
	}
	if p.Name != "SSR-Node" {
		t.Errorf("name: got %q", p.Name)
	}
}

func TestParseURI_VLESS_WS(t *testing.T) {
	uri := "vless://uuid@ws.example.com:443?encryption=none&type=ws&path=/ws&host=ws.example.com&security=tls#WS-VLESS"
	p, err := ParseURI(uri)
	if err != nil {
		t.Fatalf("ParseURI failed: %v", err)
	}
	if p.Network != "ws" {
		t.Errorf("network: got %q", p.Network)
	}
	if p.WSPath != "/ws" {
		t.Errorf("ws-path: got %q", p.WSPath)
	}
	if p.WSHeaders == nil || p.WSHeaders["Host"] != "ws.example.com" {
		t.Errorf("ws-headers: got %v", p.WSHeaders)
	}
}

func TestParseURI_Trojan_WS(t *testing.T) {
	uri := "trojan://pass@ws-trojan.com:443?type=ws&path=/ws&host=ws-trojan.com&security=tls#WS-Trojan"
	p, err := ParseURI(uri)
	if err != nil {
		t.Fatalf("ParseURI failed: %v", err)
	}
	if p.Network != "ws" {
		t.Errorf("network: got %q", p.Network)
	}
	if p.WSPath != "/ws" {
		t.Errorf("ws-path: got %q", p.WSPath)
	}
}

func TestParseURI_SS_LegacyFormat(t *testing.T) {
	// Legacy format: ss://method:password@server:port#name
	uri := "ss://aes-256-cfb:legacyPass@5.5.5.5:8888#Legacy"
	p, err := ParseURI(uri)
	if err != nil {
		t.Fatalf("ParseURI failed: %v", err)
	}
	if p.Cipher != "aes-256-cfb" || p.Password != "legacyPass" {
		t.Errorf("cipher/password: %q/%q", p.Cipher, p.Password)
	}
	if p.Port != 8888 {
		t.Errorf("port: got %d", p.Port)
	}
}

func TestParseURI_SSR_NoParams(t *testing.T) {
	passB64 := base64.RawURLEncoding.EncodeToString([]byte("testpass"))
	mainPart := "ssr.com:1234:auth_aes128_md5:aes-256-cfb:http_simple:" + passB64
	ssrURI := "ssr://" + base64.RawURLEncoding.EncodeToString([]byte(mainPart))
	p, err := ParseURI(ssrURI)
	if err != nil {
		t.Fatalf("ParseURI failed: %v", err)
	}
	if p.Name != "ssr.com" {
		t.Errorf("name fallback: got %q", p.Name)
	}
}

func TestParseURI_Hysteria2_NoName(t *testing.T) {
	uri := "hysteria2://pass@hy.example.com:443"
	p, err := ParseURI(uri)
	if err != nil {
		t.Fatalf("ParseURI failed: %v", err)
	}
	if p.Name != "hy.example.com" {
		t.Errorf("name fallback: got %q", p.Name)
	}
}

func TestParseURI_TUIC_NoName(t *testing.T) {
	uri := "tuic://uuid@tuic.example.com:443"
	p, err := ParseURI(uri)
	if err != nil {
		t.Fatalf("ParseURI failed: %v", err)
	}
	if p.Name != "tuic.example.com" {
		t.Errorf("name fallback: got %q", p.Name)
	}
}

func TestParseURI_SOCKS5_NoName(t *testing.T) {
	uri := "socks5://1.2.3.4:1080"
	p, err := ParseURI(uri)
	if err != nil {
		t.Fatalf("ParseURI failed: %v", err)
	}
	if p.Name != "1.2.3.4" {
		t.Errorf("name fallback: got %q", p.Name)
	}
}

func TestParseURI_HTTP_NoName(t *testing.T) {
	uri := "http://1.2.3.4:3128"
	p, err := ParseURI(uri)
	if err != nil {
		t.Fatalf("ParseURI failed: %v", err)
	}
	if p.Name != "1.2.3.4" {
		t.Errorf("name fallback: got %q", p.Name)
	}
}

func TestParseURI_SSD_EmptyServers(t *testing.T) {
	ssdJSON := `{"servers":[],"encryption":"aes-256-gcm","password":"pass"}`
	ssdB64 := base64.StdEncoding.EncodeToString([]byte(ssdJSON))
	uri := "ssd://" + ssdB64

	p, err := ParseURI(uri)
	if err != nil {
		t.Fatalf("ParseURI failed: %v", err)
	}
	if p.Server != "" {
		t.Errorf("expected empty server, got %q", p.Server)
	}
}

func TestParseURI_SSD_InvalidJSON(t *testing.T) {
	ssdB64 := base64.StdEncoding.EncodeToString([]byte("not-json"))
	uri := "ssd://" + ssdB64

	p, err := ParseURI(uri)
	if err != nil {
		t.Fatalf("ParseURI failed: %v", err)
	}
	// Should still return a minimal placeholder
	if p.Type != "ss" {
		t.Errorf("type: got %q", p.Type)
	}
	if p.Name != "ssd-node" {
		t.Errorf("name: got %q", p.Name)
	}
}