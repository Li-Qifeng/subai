package server

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/user/subai/internal/config"
	"github.com/user/subai/internal/converter"
	"github.com/user/subai/internal/fetcher"
	"github.com/user/subai/internal/parser"
)

// Version is the build version, set via ldflags at build time.
var Version = "dev"

// Server is the lightweight HTTP server for subai.
type Server struct {
	listen          string
	token           string
	configPath      string
	mu              sync.RWMutex

	autoRefresh     bool
	refreshInterval time.Duration
	webhookURLs     []string
	outputPath      string
	stopCh          chan struct{}
}

// New creates a new Server with the given settings.
func New(listen, token, configPath string) *Server {
	return &Server{
		listen:     listen,
		token:      token,
		configPath: configPath,
		stopCh:     make(chan struct{}),
	}
}

// WithAutoRefresh enables periodic config refresh and webhook notifications.
func (s *Server) WithAutoRefresh(interval time.Duration, webhookURLs []string, outputPath string) *Server {
	s.autoRefresh = true
	s.refreshInterval = interval
	s.webhookURLs = webhookURLs
	s.outputPath = outputPath
	return s
}

// Start begins listening and serving HTTP requests. Blocks until the server
// exits.
func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/sub", s.authMiddleware(s.handleSub))
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/version", s.handleVersion)
	mux.HandleFunc("/refresh", s.authMiddleware(s.handleRefresh))

	addr := s.listen
	if addr == "" {
		addr = ":8080"
	}

	if s.autoRefresh {
		go s.refreshLoop()
	}

	log.Printf("subai server listening on %s", addr)
	if s.autoRefresh {
		log.Printf("  auto-refresh: every %s", s.refreshInterval)
		if len(s.webhookURLs) > 0 {
			log.Printf("  webhooks: %d configured", len(s.webhookURLs))
		}
		if s.outputPath != "" {
			log.Printf("  output: %s", s.outputPath)
		}
	}
	return http.ListenAndServe(addr, mux)
}

// ---------------------------------------------------------------------------
// Auto-refresh loop
// ---------------------------------------------------------------------------

// refreshLoop runs periodic config refreshes.
func (s *Server) refreshLoop() {
	ticker := time.NewTicker(s.refreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.refreshAndNotify()
		case <-s.stopCh:
			return
		}
	}
}

// refreshAndNotify fetches all sources, generates config, writes output,
// and sends webhook notifications.
func (s *Server) refreshAndNotify() {
	log.Printf("auto-refresh: starting refresh cycle")

	cfg, err := config.Load(s.configPath)
	if err != nil {
		log.Printf("auto-refresh: load config: %v", err)
		return
	}
	cfg = cfg.Resolve("") // use current profile

	// Fetch and parse all sources
	var allProxies parser.ProxyList
	for _, src := range cfg.Sources {
		body, err := fetcher.Fetch(src.URL, src.Cookie, src.UserAgent)
		if err != nil {
			log.Printf("auto-refresh: fetch source %q: %v", src.Name, err)
			continue
		}
		proxies, err := parser.ParseAuto(body)
		if err != nil {
			log.Printf("auto-refresh: parse source %q: %v", src.Name, err)
			continue
		}
		allProxies = append(allProxies, proxies...)
	}

	if len(allProxies) == 0 {
		log.Printf("auto-refresh: no proxies found, skipping")
		return
	}

	allProxies = filterProxies(allProxies, cfg.Rules)

	// Convert to Clash format
	var data []byte
	if cfg.Output.Target == "base64" {
		data = s.renderBase64Bytes(allProxies)
	} else {
		data = s.renderClashBytes(cfg, allProxies)
	}

	if len(data) == 0 {
		log.Printf("auto-refresh: generated empty config, skipping")
		return
	}

	// Write to output file if configured
	if s.outputPath != "" {
		if err := writeFile(s.outputPath, data); err != nil {
			log.Printf("auto-refresh: write output: %v", err)
		} else {
			log.Printf("auto-refresh: wrote %d bytes to %s", len(data), s.outputPath)
		}
	}

	// Send webhooks
	if len(s.webhookURLs) > 0 {
		s.sendWebhooks(s.outputPath)
	}

	log.Printf("auto-refresh: completed (%d proxies, %d bytes)", len(allProxies), len(data))
}

// sendWebhooks sends HTTP requests to all configured webhook URLs.
func (s *Server) sendWebhooks(configPath string) {
	client := &http.Client{Timeout: 10 * time.Second}

	for _, rawURL := range s.webhookURLs {
		url := strings.TrimSpace(rawURL)
		if url == "" {
			continue
		}

		go func(u string) {
			var resp *http.Response
			var err error

			// Clash API-style: PUT /configs?force=true with path as JSON body
			if strings.Contains(u, "/configs") {
				body, _ := json.Marshal(map[string]string{"path": configPath})
				req, reqErr := http.NewRequest(http.MethodPut, u, bytes.NewReader(body))
				if reqErr != nil {
					log.Printf("webhook: create request: %v", reqErr)
					return
				}
				req.Header.Set("Content-Type", "application/json")
				resp, err = client.Do(req)
			} else {
				// Default: POST with JSON notification
				payload, _ := json.Marshal(map[string]interface{}{
					"event":   "config_updated",
					"time":    time.Now().Unix(),
					"message": "subai auto-refresh completed",
					"path":    configPath,
				})
				resp, err = client.Post(u, "application/json", bytes.NewReader(payload))
			}

			if err != nil {
				log.Printf("webhook: %s — %v", u, err)
				return
			}
			resp.Body.Close()
			if resp.StatusCode >= 300 {
				log.Printf("webhook: %s — %d", u, resp.StatusCode)
			}
		}(url)
	}
}

// ---------------------------------------------------------------------------
// Middleware
// ---------------------------------------------------------------------------

// authMiddleware wraps a handler, requiring a valid ?token=xxx parameter.
func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.token != "" && r.URL.Query().Get("token") != s.token {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

// handleSub generates a subscription on-the-fly. It reads the config, fetches
// all sources, parses the proxies, and converts to the requested target format.
//
// Query params:
//
//	token  - auth token (if server token is set)
//	target - output format: "clash" (default) or "base64"
func (s *Server) handleSub(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	target := r.URL.Query().Get("target")
	if target == "" {
		target = "clash"
	}

	// 1. Load config
	cfg, err := config.Load(s.configPath)
	if err != nil {
		log.Printf("sub: load config: %v", err)
		http.Error(w, fmt.Sprintf(`{"error":"load config: %v"}`, err), http.StatusInternalServerError)
		return
	}

	// Resolve profile from ?profile= query param
	profileName := r.URL.Query().Get("profile")
	cfg = cfg.Resolve(profileName)

	// 2. Fetch and parse each source
	var allProxies parser.ProxyList
	for _, src := range cfg.Sources {
		body, err := fetcher.Fetch(src.URL, src.Cookie, src.UserAgent)
		if err != nil {
			log.Printf("sub: fetch source %q: %v", src.Name, err)
			continue // skip failing sources
		}

		proxies, err := parser.ParseAuto(body)
		if err != nil {
			log.Printf("sub: parse source %q: %v", src.Name, err)
			continue
		}
		allProxies = append(allProxies, proxies...)
	}

	// 3. Apply filter rules
	allProxies = filterProxies(allProxies, cfg.Rules)

	// 4. Render to requested target format
	switch target {
	case "base64":
		s.renderBase64(w, allProxies)
	default:
		s.renderClash(w, cfg, allProxies)
	}
}

// handleHealth returns a simple health-check JSON response.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status":"ok"}`)
}

// handleVersion returns version information as JSON.
func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"version": Version,
	})
}

// handleRefresh triggers a config refresh by reloading the config file.
func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Reload the config to verify it is valid
	_, err := config.Load(s.configPath)
	if err != nil {
		log.Printf("refresh: reload config: %v", err)
		http.Error(w, fmt.Sprintf(`{"error":"refresh failed: %v"}`, err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status":"ok","message":"config refreshed"}`)
}

// ---------------------------------------------------------------------------
// Output renderers (HTTP response)
// ---------------------------------------------------------------------------

// renderClash renders the proxy list as a Clash-compatible YAML configuration.
func (s *Server) renderClash(w http.ResponseWriter, cfg *config.Config, proxies parser.ProxyList) {
	data := s.renderClashBytes(cfg, proxies)
	if data == nil {
		http.Error(w, `{"error":"render clash"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\"sub.yaml\"")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// renderClashBytes generates Clash YAML bytes without writing to a response.
func (s *Server) renderClashBytes(cfg *config.Config, proxies parser.ProxyList) []byte {
	// Use template-based converter if a template is configured
	if cfg.Output.Template.Template != "" || len(cfg.Output.Template.ProxyGroups) > 0 {
		eng := converter.NewWithTemplate(&cfg.Output.Template)
		if data, err := eng.Convert(proxies, "clash"); err == nil && len(data) > 0 {
			return data
		}
	}

	// Fallback to simple inline renderer
	out := map[string]interface{}{
		"port":                7890,
		"socks-port":          7891,
		"allow-lan":           false,
		"mode":                "Rule",
		"log-level":           "info",
		"external-controller": "127.0.0.1:9090",
		"proxies":             proxies,
	}

	if len(proxies) > 0 {
		names := make([]string, len(proxies))
		for i, p := range proxies {
			names[i] = p.Name
		}
		out["proxy-groups"] = []map[string]interface{}{
			{
				"name":    "Proxy",
				"type":    "select",
				"proxies": names,
			},
		}
	}

	data, err := yaml.Marshal(out)
	if err != nil {
		log.Printf("render clash bytes: %v", err)
		return nil
	}
	return data
}

// renderBase64 renders the proxy list as a base64-encoded subscription.
func (s *Server) renderBase64(w http.ResponseWriter, proxies parser.ProxyList) {
	data := s.renderBase64Bytes(proxies)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\"sub.txt\"")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// renderBase64Bytes generates base64 subscription bytes without writing to a response.
func (s *Server) renderBase64Bytes(proxies parser.ProxyList) []byte {
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
	return []byte(base64.StdEncoding.EncodeToString(buf.Bytes()))
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// filterProxies applies include/exclude rules to the proxy list.
func filterProxies(proxies parser.ProxyList, rules config.Rules) parser.ProxyList {
	if len(rules.Include) == 0 && len(rules.Exclude) == 0 {
		return proxies
	}

	var filtered parser.ProxyList
	for _, p := range proxies {
		if !matchAny(p.Name, rules.Exclude) && (len(rules.Include) == 0 || matchAny(p.Name, rules.Include)) {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// matchAny returns true if name contains any of the substrings.
func matchAny(name string, patterns []string) bool {
	for _, pat := range patterns {
		if strings.Contains(name, pat) {
			return true
		}
	}
	return false
}

// writeFile is a variable so tests can mock it.
var writeFile = func(path string, data []byte) error {
	// Use the os package via a minimal wrapper
	return fmt.Errorf("writeFile not implemented (use real os.WriteFile)")
}

func init() {
	// In the real server, writeFile is set to os.WriteFile by the main package.
	// Tests override it with a mock.
}

// SetWriteFile allows tests to inject a mock file writer.
func SetWriteFile(fn func(string, []byte) error) {
	writeFile = fn
}

// ResetWriteFile restores the default writeFile (which panics in tests).
func ResetWriteFile() {
	writeFile = func(path string, data []byte) error {
		return fmt.Errorf("writeFile not implemented")
	}
}

// proxyToURI converts a Proxy back to a subscription URI string. This is a
// best-effort reconstruction used for base64 output. Not every proxy type
// can be perfectly round-tripped.
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
		jsonBytes, _ := json.Marshal(vmess)
		return "vmess://" + base64.RawURLEncoding.EncodeToString(jsonBytes)
	case "vless":
		u := fmt.Sprintf("vless://%s@%s:%d", p.UUID, p.Server, p.Port)
		params := []string{"encryption=" + p.Encryption, "type=" + p.Network}
		if p.Security != "" {
			params = append(params, "security="+p.Security)
		}
		if p.SNI != "" {
			params = append(params, "sni="+p.SNI)
		}
		if p.Flow != "" {
			params = append(params, "flow="+p.Flow)
		}
		u += "?" + strings.Join(params, "&")
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
		u += "?" + strings.Join(params, "&")
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