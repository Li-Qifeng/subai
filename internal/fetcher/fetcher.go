package fetcher

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var defaultClient = &http.Client{
	Timeout: 30 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) > 5 {
			return fmt.Errorf("too many redirects")
		}
		return nil
	},
}

// Fetch downloads content from a URL with optional cookie and user-agent.
// Falls back to cloudscraper (Python) if the HTTP client fails or gets 403.
func Fetch(url, cookie, userAgent string) ([]byte, error) {
	// Try Go HTTP client first
	body, err := fetchGo(url, cookie, userAgent)
	if err == nil {
		return body, nil
	}

	// Check if it's a timeout, connection, or 403 issue — try cloudscraper fallback
	errStr := err.Error()
	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "connection") || strings.Contains(errStr, "EOF") || strings.Contains(errStr, "HTTP 403") {
		return fetchCloudscraper(url)
	}

	return nil, err
}

func fetchGo(url, cookie, userAgent string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	} else {
		req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15")
	}

	resp, err := defaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s: HTTP %d", url, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	return body, nil
}

func fetchCloudscraper(url string) ([]byte, error) {
	cmd := exec.Command("python3", "-c", `
import cloudscraper, sys
scraper = cloudscraper.create_scraper(delay=10)
scraper.headers.update({"User-Agent": "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15"})
try:
    r = scraper.get(sys.argv[1], timeout=30)
    sys.stdout.buffer.write(r.content)
except Exception as e:
    sys.stderr.write("ERROR: " + str(e) + "\n")
    sys.exit(1)
`, url)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("cloudscraper: %v", err)
	}
	return output, nil
}

// Session is a simple fetch session with cookie and user-agent tracking.
// Used by the login flow to maintain state across multiple requests.
type Session struct {
	URL       string
	Cookie    string
	UserAgent string
	client    *http.Client
}

// NewSession creates a new fetch session.
func NewSession(url, cookie, userAgent string) *Session {
	return &Session{
		URL:       url,
		Cookie:    cookie,
		UserAgent: userAgent,
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

// Fetch makes a request with the session's cookie and user-agent.
func (s *Session) Fetch() ([]byte, error) {
	return Fetch(s.URL, s.Cookie, s.UserAgent)
}

// UpdateCookie updates the session's cookie.
func (s *Session) UpdateCookie(cookie string) {
	s.Cookie = cookie
}

// setClient sets a custom HTTP client (used for testing with httptest).
func (s *Session) setClient(client *http.Client) {
	s.client = client
}

// PersistentCache is a disk-based cache for subscription fetch results.
// It survives process restarts, so stale nodes are preserved when a source
// is temporarily unreachable.
type PersistentCache struct {
	dir string
}

// NewPersistentCache creates a cache in the given directory.
func NewPersistentCache(dir string) *PersistentCache {
	return &PersistentCache{dir: dir}
}

// Get returns cached data for a source name, or nil if not cached.
func (c *PersistentCache) Get(name string) []byte {
	if c == nil || c.dir == "" {
		return nil
	}
	path := filepath.Join(c.dir, ".cache_"+name+".raw")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return data
}

// Set caches data for a source name.
func (c *PersistentCache) Set(name string, data []byte) {
	if c == nil || c.dir == "" {
		return
	}
	path := filepath.Join(c.dir, ".cache_"+name+".raw")
	if err := os.WriteFile(path, data, 0644); err != nil {
		log.Printf("cache: write %q: %v", name, err)
	}
}

// FetchWithCache fetches a URL, falling back to persistent cache on failure.
// If the fetch succeeds, the result is cached for future use.
func (c *PersistentCache) FetchWithCache(name, url, cookie, userAgent string) ([]byte, error) {
	body, err := Fetch(url, cookie, userAgent)
	if err == nil {
		// Success: cache and return
		c.Set(name, body)
		return body, nil
	}
	// Fetch failed: try cache
	if cached := c.Get(name); len(cached) > 0 {
		return cached, nil
	}
	return nil, err
}

// FetchRaw is like Fetch but returns the raw HTTP response body and status code.
// Used by the login flow to inspect the response.
func FetchRaw(url, cookie, userAgent string) ([]byte, int, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("create request: %w", err)
	}
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}
	resp, err := defaultClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read body: %w", err)
	}
	return body, resp.StatusCode, nil
}

// FetchJSON fetches a URL and parses the response as JSON.
func FetchJSON(url, cookie, userAgent string, target interface{}) error {
	body, err := Fetch(url, cookie, userAgent)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, target)
}