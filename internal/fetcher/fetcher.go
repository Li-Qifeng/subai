package fetcher

import (
	"fmt"
	"io"
	"net/http"
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
func Fetch(url, cookie, userAgent string) ([]byte, error) {
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
		req.Header.Set("User-Agent", "subai/1.0")
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
		return nil, fmt.Errorf("read response: %w", err)
	}

	return body, nil
}

// FetchWithSession fetches with cookie management, supporting refresh.
type Session struct {
	URL       string
	Cookie    string
	UserAgent string
}

func NewSession(url, cookie, ua string) *Session {
	return &Session{URL: url, Cookie: cookie, UserAgent: ua}
}

func (s *Session) Fetch() ([]byte, error) {
	return Fetch(s.URL, s.Cookie, s.UserAgent)
}

func (s *Session) UpdateCookie(cookie string) {
	s.Cookie = cookie
}