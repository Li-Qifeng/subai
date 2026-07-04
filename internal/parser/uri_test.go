package parser

import (
	"encoding/base64"
	"testing"
)

func TestParseURI_SS(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		wantName string
		wantType string
		wantSrv  string
		wantPort int
		wantCiph string
	}{
		{
			name:     "SIP002 with base64 userinfo",
			uri:      "ss://YWVzLTI1Ni1nY206cGFzc3dk@1.1.1.1:443#TestNode",
			wantName: "TestNode",
			wantType: "ss",
			wantSrv:  "1.1.1.1",
			wantPort: 443,
			wantCiph: "aes-256-gcm",
		},
		{
			name:     "legacy format",
			uri:      "ss://aes-256-gcm:password@2.2.2.2:8388#LegacyNode",
			wantName: "LegacyNode",
			wantType: "ss",
			wantSrv:  "2.2.2.2",
			wantPort: 8388,
			wantCiph: "aes-256-gcm",
		},
		{
			name:     "no fragment uses host",
			uri:      "ss://YWVzLTI1Ni1nY206cGFzc3dk@3.3.3.3:443",
			wantName: "3.3.3.3:443",
			wantType: "ss",
			wantSrv:  "3.3.3.3",
			wantPort: 443,
		},
		{
			name:     "with plugin",
			uri:      "ss://YWVzLTI1Ni1nY206cGFzc3dk@1.1.1.1:443?plugin=obfs-local;obfs=http;obfs-host=www.example.com#PluginNode",
			wantName: "PluginNode",
			wantType: "ss",
			wantSrv:  "1.1.1.1",
			wantPort: 443,
			wantCiph: "aes-256-gcm",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := ParseURI(tt.uri)
			if err != nil {
				t.Fatalf("ParseURI failed: %v", err)
			}
			if p.Name != tt.wantName {
				t.Errorf("name: got %q, want %q", p.Name, tt.wantName)
			}
			if p.Type != tt.wantType {
				t.Errorf("type: got %q, want %q", p.Type, tt.wantType)
			}
			if p.Server != tt.wantSrv {
				t.Errorf("server: got %q, want %q", p.Server, tt.wantSrv)
			}
			if p.Port != tt.wantPort {
				t.Errorf("port: got %d, want %d", p.Port, tt.wantPort)
			}
			if tt.wantCiph != "" && p.Cipher != tt.wantCiph {
				t.Errorf("cipher: got %q, want %q", p.Cipher, tt.wantCiph)
			}
		})
	}
}

func TestParseURI_SS_Plugin(t *testing.T) {
	// Test that plugin info is parsed into Obfs/ObfsParam
	uri := "ss://YWVzLTI1Ni1nY206cGFzc3dkQDEuMS4xLjE6NDQz?plugin=obfs-local%3Bobfs%3Dhttp%3Bobfs-host%3Dwww.example.com#PluginNode"
	p, err := ParseURI(uri)
	if err != nil {
		t.Fatalf("ParseURI failed: %v", err)
	}
	if p.Obfs != "obfs-local" {
		t.Errorf("obfs: got %q, want %q", p.Obfs, "obfs-local")
	}
	if p.ObfsParam == "" {
		t.Error("expected non-empty obfs-param")
	}
}

func TestParseURI_SSR(t *testing.T) {
	// ssr://base64(server:port:protocol:method:obfs:base64(password)/?params)
	passB64 := base64.RawURLEncoding.EncodeToString([]byte("testpass"))
	remarksB64 := base64.RawURLEncoding.EncodeToString([]byte("SSR-Node"))
	// Inner part includes /?params inside the base64
	innerPart := "server.example.com:1234:auth_chain_a:aes-256-cfb:tls1.2_ticket_auth:" + passB64 + "/?remarks=" + remarksB64
	ssrURI := "ssr://" + base64.RawURLEncoding.EncodeToString([]byte(innerPart))

	p, err := ParseURI(ssrURI)
	if err != nil {
		t.Fatalf("ParseURI SSR failed: %v", err)
	}
	if p.Type != "ssr" {
		t.Errorf("type: got %q, want %q", p.Type, "ssr")
	}
	if p.Server != "server.example.com" {
		t.Errorf("server: got %q", p.Server)
	}
	if p.Port != 1234 {
		t.Errorf("port: got %d, want 1234", p.Port)
	}
	if p.Protocol != "auth_chain_a" {
		t.Errorf("protocol: got %q", p.Protocol)
	}
	if p.Cipher != "aes-256-cfb" {
		t.Errorf("cipher: got %q", p.Cipher)
	}
	if p.Obfs != "tls1.2_ticket_auth" {
		t.Errorf("obfs: got %q", p.Obfs)
	}
	if p.Password != "testpass" {
		t.Errorf("password: got %q", p.Password)
	}
	if p.Name != "SSR-Node" {
		t.Errorf("name: got %q, want %q", p.Name, "SSR-Node")
	}
}

func TestParseURI_VMess(t *testing.T) {
	vmessJSON := `{"v":"2","ps":"VMess-Node","add":"vmess.example.com","port":"443","id":"550e8400-e29b-41d4-a716-446655440000","aid":"0","net":"ws","type":"none","host":"vmess.example.com","path":"/ws","tls":"tls","scy":"auto"}`
	vmessB64 := base64.RawURLEncoding.EncodeToString([]byte(vmessJSON))
	uri := "vmess://" + vmessB64

	p, err := ParseURI(uri)
	if err != nil {
		t.Fatalf("ParseURI VMess failed: %v", err)
	}
	if p.Type != "vmess" {
		t.Errorf("type: got %q, want %q", p.Type, "vmess")
	}
	if p.Name != "VMess-Node" {
		t.Errorf("name: got %q", p.Name)
	}
	if p.Server != "vmess.example.com" {
		t.Errorf("server: got %q", p.Server)
	}
	if p.Port != 443 {
		t.Errorf("port: got %d", p.Port)
	}
	if p.UUID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("uuid: got %q", p.UUID)
	}
	if p.Network != "ws" {
		t.Errorf("network: got %q", p.Network)
	}
	if p.WSPath != "/ws" {
		t.Errorf("ws-path: got %q", p.WSPath)
	}
	if p.Encryption != "auto" {
		t.Errorf("encryption/scy: got %q", p.Encryption)
	}
	if p.SNI != "vmess.example.com" {
		t.Errorf("sni: got %q", p.SNI)
	}
}

func TestParseURI_VMess_NoTLS(t *testing.T) {
	vmessJSON := `{"v":"2","ps":"VMess-NoTLS","add":"no-tls.example.com","port":"80","id":"550e8400-e29b-41d4-a716-446655440000","aid":"0","net":"tcp","type":"none"}`
	vmessB64 := base64.RawURLEncoding.EncodeToString([]byte(vmessJSON))
	uri := "vmess://" + vmessB64

	p, err := ParseURI(uri)
	if err != nil {
		t.Fatalf("ParseURI failed: %v", err)
	}
	if p.SNI != "" {
		t.Errorf("expected empty sni for non-TLS, got %q", p.SNI)
	}
	if p.Encryption != "" {
		t.Errorf("expected empty encryption, got %q", p.Encryption)
	}
}

func TestParseURI_VMess_Invalid(t *testing.T) {
	_, err := ParseURI("vmess://invalid-base64!!!")
	if err == nil {
		t.Fatal("expected error for invalid vmess")
	}
}

func TestParseURI_VLETS(t *testing.T) {
	uri := "vless://550e8400-e29b-41d4-a716-446655440000@vless.example.com:443?encryption=none&type=tcp&security=reality&flow=xtls-rprx-vision&sni=www.example.com&fp=chrome&pbk=publicKeyHere&sid=123456#VLESS-Node"

	p, err := ParseURI(uri)
	if err != nil {
		t.Fatalf("ParseURI VLESS failed: %v", err)
	}
	if p.Type != "vless" {
		t.Errorf("type: got %q", p.Type)
	}
	if p.Name != "VLESS-Node" {
		t.Errorf("name: got %q", p.Name)
	}
	if p.Server != "vless.example.com" {
		t.Errorf("server: got %q", p.Server)
	}
	if p.Port != 443 {
		t.Errorf("port: got %d", p.Port)
	}
	if p.UUID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("uuid: got %q", p.UUID)
	}
	if p.Encryption != "none" {
		t.Errorf("encryption: got %q", p.Encryption)
	}
	if p.Network != "tcp" {
		t.Errorf("network: got %q", p.Network)
	}
	if p.Security != "reality" {
		t.Errorf("security: got %q", p.Security)
	}
	if p.Flow != "xtls-rprx-vision" {
		t.Errorf("flow: got %q", p.Flow)
	}
	if p.SNI != "www.example.com" {
		t.Errorf("sni: got %q", p.SNI)
	}
	if p.Fingerprint != "chrome" {
		t.Errorf("fp: got %q", p.Fingerprint)
	}
	if p.PublicKey != "publicKeyHere" {
		t.Errorf("pbk: got %q", p.PublicKey)
	}
	if p.ShortID != "123456" {
		t.Errorf("sid: got %q", p.ShortID)
	}
}

func TestParseURI_Trojan(t *testing.T) {
	tests := []struct {
		name    string
		uri     string
		wantSrv string
		wantPwd string
		wantSNI string
	}{
		{
			name:    "basic trojan",
			uri:     "trojan://password123@trojan.example.com:443?type=tcp&sni=trojan.example.com#Trojan-Node",
			wantSrv: "trojan.example.com",
			wantPwd: "password123",
			wantSNI: "trojan.example.com",
		},
		{
			name:    "trojan with ws",
			uri:     "trojan://pass@ws-trojan.com:443?type=ws&path=/ws&host=ws-trojan.com&security=tls#WSTrojan",
			wantSrv: "ws-trojan.com",
			wantPwd: "pass",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := ParseURI(tt.uri)
			if err != nil {
				t.Fatalf("ParseURI failed: %v", err)
			}
			if p.Type != "trojan" {
				t.Errorf("type: got %q", p.Type)
			}
			if p.Server != tt.wantSrv {
				t.Errorf("server: got %q", p.Server)
			}
			if p.Password != tt.wantPwd {
				t.Errorf("password: got %q", p.Password)
			}
			if p.Security != "tls" {
				t.Errorf("security: expected tls, got %q", p.Security)
			}
		})
	}
}

func TestParseURI_Hysteria2(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		wantSrv  string
		wantPwd  string
		wantType string
	}{
		{
			name:     "hysteria2",
			uri:      "hysteria2://hy2pass@hy2.example.com:8443?sni=hy2.example.com&insecure=1#Hy2-Node",
			wantSrv:  "hy2.example.com",
			wantPwd:  "hy2pass",
			wantType: "hysteria2",
		},
		{
			name:     "hy2 alias",
			uri:      "hy2://hy2pass@hy2-alt.com:8443#Hy2-Alias",
			wantSrv:  "hy2-alt.com",
			wantPwd:  "hy2pass",
			wantType: "hysteria2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := ParseURI(tt.uri)
			if err != nil {
				t.Fatalf("ParseURI failed: %v", err)
			}
			if p.Type != tt.wantType {
				t.Errorf("type: got %q, want %q", p.Type, tt.wantType)
			}
			if p.Server != tt.wantSrv {
				t.Errorf("server: got %q", p.Server)
			}
			if p.Password != tt.wantPwd {
				t.Errorf("password: got %q", p.Password)
			}
		})
	}
}

func TestParseURI_Hysteria2_Insecure(t *testing.T) {
	uri := "hysteria2://pass@host.com:443?insecure=1&sni=host.com"
	p, err := ParseURI(uri)
	if err != nil {
		t.Fatalf("ParseURI failed: %v", err)
	}
	if !p.SkipCertVerify {
		t.Error("expected SkipCertVerify=true")
	}
}

func TestParseURI_TUIC(t *testing.T) {
	uri := "tuic://uuid@tuic.example.com:12345?password=tuicpass&sni=tuic.example.com#TUIC-Node"

	p, err := ParseURI(uri)
	if err != nil {
		t.Fatalf("ParseURI TUIC failed: %v", err)
	}
	if p.Type != "tuic" {
		t.Errorf("type: got %q", p.Type)
	}
	if p.Name != "TUIC-Node" {
		t.Errorf("name: got %q", p.Name)
	}
	if p.Server != "tuic.example.com" {
		t.Errorf("server: got %q", p.Server)
	}
	if p.Port != 12345 {
		t.Errorf("port: got %d", p.Port)
	}
	if p.UUID != "uuid" {
		t.Errorf("uuid: got %q", p.UUID)
	}
	if p.Password != "tuicpass" {
		t.Errorf("password: got %q", p.Password)
	}
	if p.SNI != "tuic.example.com" {
		t.Errorf("sni: got %q", p.SNI)
	}
}

func TestParseURI_SOCKS5(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		wantSrv  string
		wantPort int
		wantUser string
		wantPass string
	}{
		{
			name:     "socks5 without auth",
			uri:      "socks5://socks.example.com:1080#SOCKS-Node",
			wantSrv:  "socks.example.com",
			wantPort: 1080,
		},
		{
			name:     "socks5 with auth",
			uri:      "socks5://user:pass@socks-auth.com:1080#SOCKS-Auth",
			wantSrv:  "socks-auth.com",
			wantPort: 1080,
			wantUser: "user",
			wantPass: "pass",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := ParseURI(tt.uri)
			if err != nil {
				t.Fatalf("ParseURI failed: %v", err)
			}
			if p.Type != "socks5" {
				t.Errorf("type: got %q", p.Type)
			}
			if p.Server != tt.wantSrv {
				t.Errorf("server: got %q", p.Server)
			}
			if p.Port != tt.wantPort {
				t.Errorf("port: got %d", p.Port)
			}
			if p.Username != tt.wantUser {
				t.Errorf("username: got %q", p.Username)
			}
			if p.Password != tt.wantPass {
				t.Errorf("password: got %q", p.Password)
			}
		})
	}
}

func TestParseURI_HTTP(t *testing.T) {
	tests := []struct {
		name    string
		uri     string
		wantSrv string
	}{
		{
			name:    "http proxy",
			uri:     "http://http.example.com:3128#HTTP-Proxy",
			wantSrv: "http.example.com",
		},
		{
			name:    "http with auth",
			uri:     "http://user:pass@http-auth.com:3128",
			wantSrv: "http-auth.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := ParseURI(tt.uri)
			if err != nil {
				t.Fatalf("ParseURI failed: %v", err)
			}
			if p.Type != "http" {
				t.Errorf("type: got %q, want http", p.Type)
			}
			if p.Server != tt.wantSrv {
				t.Errorf("server: got %q", p.Server)
			}
		})
	}
}

func TestParseURI_Unknown(t *testing.T) {
	_, err := ParseURI("unknown://scheme")
	if err == nil {
		t.Fatal("expected error for unknown scheme")
	}
}

func TestParseURI_SSD(t *testing.T) {
	ssdJSON := `{"servers":[{"server":"ssd.example.com","port":443,"remarks":"SSD-Node","method":"aes-256-gcm","password":"ssdpass"}],"encryption":"aes-256-gcm","password":"globalpass"}`
	ssdB64 := base64.StdEncoding.EncodeToString([]byte(ssdJSON))
	uri := "ssd://" + ssdB64

	p, err := ParseURI(uri)
	if err != nil {
		t.Fatalf("ParseURI SSD failed: %v", err)
	}
	if p.Type != "ss" {
		t.Errorf("type: got %q, want ss", p.Type)
	}
	if p.Server != "ssd.example.com" {
		t.Errorf("server: got %q", p.Server)
	}
	if p.Port != 443 {
		t.Errorf("port: got %d", p.Port)
	}
	if p.Name != "SSD-Node" {
		t.Errorf("name: got %q", p.Name)
	}
	if p.Cipher != "aes-256-gcm" {
		t.Errorf("cipher: got %q", p.Cipher)
	}
	if p.Password != "ssdpass" {
		t.Errorf("password: got %q", p.Password)
	}
}

func TestParseURI_SSD_GlobalFallback(t *testing.T) {
	// SSD with per-server missing fields that fall back to global
	ssdJSON := `{"servers":[{"server":"ssd2.example.com","port":8443,"remarks":"SSD2"}],"encryption":"chacha20-ietf-poly1305","password":"globalpwd"}`
	ssdB64 := base64.StdEncoding.EncodeToString([]byte(ssdJSON))
	uri := "ssd://" + ssdB64

	p, err := ParseURI(uri)
	if err != nil {
		t.Fatalf("ParseURI failed: %v", err)
	}
	if p.Cipher != "chacha20-ietf-poly1305" {
		t.Errorf("expected global cipher, got %q", p.Cipher)
	}
	if p.Password != "globalpwd" {
		t.Errorf("expected global password, got %q", p.Password)
	}
}

func TestParseURI_Empty(t *testing.T) {
	_, err := ParseURI("")
	if err == nil {
		t.Fatal("expected error for empty URI")
	}
}