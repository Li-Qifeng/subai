package converter

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/user/subai/internal/parser"
	"github.com/user/subai/internal/template"
	"gopkg.in/yaml.v3"
)

// Engine is the conversion engine.
type Engine struct {
	tmplCfg *template.Config // optional template config
}

// New creates a new Engine with default (basic) template.
func New() *Engine {
	tmpl, err := template.Builtin("basic")
	if err != nil {
		// Built-in template should always be available; if not, fall back to empty
		tmpl = &template.Config{}
	}
	return &Engine{tmplCfg: tmpl}
}

// NewWithTemplate creates an Engine with a specific template config.
// If cfg is nil, uses basic template. If cfg.Template is a built-in name,
// loads that built-in and merges any custom overrides.
func NewWithTemplate(cfg *template.Config) *Engine {
	if cfg == nil {
		tmpl, _ := template.Builtin("basic")
		if tmpl == nil {
			tmpl = &template.Config{}
		}
		return &Engine{tmplCfg: tmpl}
	}

	// If a built-in template is referenced, load it as base and merge
	if cfg.Template != "" {
		base, err := template.Builtin(cfg.Template)
		if err == nil {
			if len(cfg.ProxyGroups) > 0 {
				base.ProxyGroups = cfg.ProxyGroups
			}
			if len(cfg.RuleSets) > 0 {
				base.RuleSets = append(base.RuleSets, cfg.RuleSets...)
			}
			if len(cfg.RulePatches) > 0 {
				base.RulePatches = append(base.RulePatches, cfg.RulePatches...)
			}
			// Config-level settings always override template
			base.RuleProviders = cfg.RuleProviders
			base.ProviderInterval = cfg.ProviderInterval
			base.ProviderProxy = cfg.ProviderProxy
			base.FetchRules = cfg.FetchRules
			return &Engine{tmplCfg: base}
		}
	}

	return &Engine{tmplCfg: cfg}
}

// Convert converts a list of proxies to the target format.
func (e *Engine) Convert(proxies []parser.Proxy, target string) ([]byte, error) {
	if e.tmplCfg == nil {
		return nil, fmt.Errorf("engine: template config is nil")
	}
	switch target {
	case "clash":
		return e.toClash(proxies)
	case "base64":
		return toBase64(proxies)
	case "mixed":
		return e.toMixed(proxies)
	default:
		return nil, fmt.Errorf("unknown target format: %s", target)
	}
}

// toClash generates a Clash-compatible YAML with template-based proxy groups and rules.
func (e *Engine) toClash(proxies []parser.Proxy) ([]byte, error) {
	// Convert proxies to Clash-compatible format (fix Reality field names)
	clashProxies := make([]map[string]interface{}, len(proxies))
	for i, p := range proxies {
		clashProxies[i] = p.ToClashProxy()
	}

	out := map[string]interface{}{
		"port":                 7890,
		"socks-port":           7891,
		"allow-lan":            false,
		"mode":                 "Rule",
		"log-level":            "info",
		"external-controller":  "127.0.0.1:9090",
		"proxies":              clashProxies,
	}

	// Build from template
	template.SetLogWriter(os.Stderr)
	result := template.Build(e.tmplCfg, proxies)

	if len(result.ProxyGroups) > 0 {
		out["proxy-groups"] = result.ProxyGroups
	}
	if len(result.Rules) > 0 {
		out["rules"] = result.Rules
	}
	if len(result.RuleProviders) > 0 {
		// Merge all providers into a single rule-providers map
		providers := map[string]interface{}{}
		for _, entry := range result.RuleProviders {
			for k, v := range entry {
				providers[k] = v
			}
		}
		out["rule-providers"] = providers
	}

	data, err := yaml.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("marshal yaml: %w", err)
	}
	return unescapeYAML(data), nil
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
func (e *Engine) toMixed(proxies []parser.Proxy) ([]byte, error) {
	clashBytes, err := e.toClash(proxies)
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

// unescapeYAML replaces \UXXXXXXXX escape sequences (from yaml.v3's non-BMP
// escaping) with the actual Unicode characters, preventing garbled display
// in Clash clients that don't support these YAML escape sequences.
func unescapeYAML(data []byte) []byte {
	re := regexp.MustCompile(`\\U([0-9a-fA-F]{8})`)
	return re.ReplaceAllFunc(data, func(match []byte) []byte {
		hex := string(match[2:]) // skip \U
		n, _ := strconv.ParseUint(hex, 16, 32)
		return []byte(string(rune(n)))
	})
}

// proxyToURI converts a Proxy back to a subscription URI string.
func proxyToURI(p parser.Proxy) string {
	// URL-encode the name fragment to handle spaces and special chars
	nameFragment := ""
	if p.Name != "" {
		nameFragment = "#" + url.PathEscape(p.Name)
	}

	switch p.Type {
	case "ss":
		userInfo := base64.RawURLEncoding.EncodeToString([]byte(p.Cipher + ":" + p.Password))
		uri := fmt.Sprintf("ss://%s@%s:%d", userInfo, p.Server, p.Port)
		return uri + nameFragment
	case "ssr":
		passB64 := base64.URLEncoding.EncodeToString([]byte(p.Password))
		mainPart := fmt.Sprintf("%s:%d:%s:%s:%s:%s", p.Server, p.Port, p.Protocol, p.Cipher, p.Obfs, passB64)
		params := ""
		if p.ObfsParam != "" {
			params += "&obfsparam=" + base64.URLEncoding.EncodeToString([]byte(p.ObfsParam))
		}
		if p.ProtocolParam != "" {
			params += "&protoparam=" + base64.URLEncoding.EncodeToString([]byte(p.ProtocolParam))
		}
		if p.Name != "" {
			params += "&remarks=" + base64.URLEncoding.EncodeToString([]byte(p.Name))
		}
		if params != "" {
			mainPart += "/?" + strings.TrimPrefix(params, "&")
		}
		encoded := base64.URLEncoding.EncodeToString([]byte(mainPart))
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
		jsonBytes, err := json.Marshal(vmess)
		if err != nil {
			return ""
		}
		return "vmess://" + base64.RawURLEncoding.EncodeToString(jsonBytes)
	case "vless":
		u := fmt.Sprintf("vless://%s@%s:%d", p.UUID, p.Server, p.Port)
		q := url.Values{}
		if p.Encryption == "" {
			q.Set("encryption", "none")
		} else {
			q.Set("encryption", p.Encryption)
		}
		q.Set("type", p.Network)
		if p.Security != "" {
			q.Set("security", p.Security)
		}
		if p.SNI != "" {
			q.Set("sni", p.SNI)
		}
		if p.Flow != "" {
			q.Set("flow", p.Flow)
		}
		if p.Fingerprint != "" {
			q.Set("fp", p.Fingerprint)
		}
		if p.PublicKey != "" {
			q.Set("pbk", p.PublicKey)
		}
		if p.ShortID != "" {
			q.Set("sid", p.ShortID)
		}
		if p.Network == "ws" && p.WSPath != "" {
			q.Set("path", p.WSPath)
		}
		if p.Network == "ws" && p.SNI != "" {
			q.Set("host", p.SNI)
		}
		return u + "?" + q.Encode() + nameFragment
	case "trojan":
		u := fmt.Sprintf("trojan://%s@%s:%d", p.Password, p.Server, p.Port)
		q := url.Values{}
		q.Set("type", p.Network)
		if p.SNI != "" {
			q.Set("sni", p.SNI)
		}
		if p.Security != "" {
			q.Set("security", p.Security)
		}
		return u + "?" + q.Encode() + nameFragment
	case "hysteria2", "hy2":
		u := fmt.Sprintf("hysteria2://%s@%s:%d", p.Password, p.Server, p.Port)
		q := url.Values{}
		if p.SNI != "" {
			q.Set("sni", p.SNI)
		}
		if p.SkipCertVerify {
			q.Set("insecure", "1")
		}
		qs := q.Encode()
		if qs != "" {
			u += "?" + qs
		}
		return u + nameFragment
	case "tuic":
		u := fmt.Sprintf("tuic://%s@%s:%d", p.UUID, p.Server, p.Port)
		q := url.Values{}
		if p.Password != "" {
			q.Set("password", p.Password)
		}
		if p.SNI != "" {
			q.Set("sni", p.SNI)
		}
		qs := q.Encode()
		if qs != "" {
			u += "?" + qs
		}
		return u + nameFragment
	case "socks5":
		u := fmt.Sprintf("socks5://%s:%d", p.Server, p.Port)
		if p.Username != "" && p.Password != "" {
			u = fmt.Sprintf("socks5://%s:%s@%s:%d", p.Username, p.Password, p.Server, p.Port)
		}
		return u + nameFragment
	case "http":
		u := fmt.Sprintf("http://%s:%d", p.Server, p.Port)
		if p.Username != "" && p.Password != "" {
			u = fmt.Sprintf("http://%s:%s@%s:%d", p.Username, p.Password, p.Server, p.Port)
		}
		return u + nameFragment
	default:
		return ""
	}
}