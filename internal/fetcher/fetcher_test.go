package fetcher

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetch_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "custom-agent" {
			t.Errorf("expected User-Agent=custom-agent, got %q", r.Header.Get("User-Agent"))
		}
		if r.Header.Get("Cookie") != "session=abc" {
			t.Errorf("expected Cookie=session=abc, got %q", r.Header.Get("Cookie"))
		}
		w.Write([]byte("hello world"))
	}))
	defer ts.Close()

	body, err := Fetch(ts.URL, "session=abc", "custom-agent")
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	if string(body) != "hello world" {
		t.Errorf("expected 'hello world', got %q", string(body))
	}
}

func TestFetch_DefaultUserAgent(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("expected non-empty User-Agent")
		}
		w.Write([]byte("ok"))
	}))
	defer ts.Close()

	body, err := Fetch(ts.URL, "", "")
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	if string(body) != "ok" {
		t.Errorf("expected 'ok', got %q", string(body))
	}
}

func TestFetch_HTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	_, err := Fetch(ts.URL, "", "")
	if err == nil {
		t.Fatal("expected error for HTTP 404")
	}
}

func TestFetch_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	_, err := Fetch(ts.URL, "", "")
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestFetch_InvalidURL(t *testing.T) {
	_, err := Fetch("://invalid-url", "", "")
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestSession(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("session-data"))
	}))
	defer ts.Close()

	sess := NewSession(ts.URL, "init-cookie", "init-ua")
	if sess.URL != ts.URL {
		t.Errorf("url mismatch")
	}
	if sess.Cookie != "init-cookie" {
		t.Errorf("cookie mismatch")
	}

	body, err := sess.Fetch()
	if err != nil {
		t.Fatalf("Session.Fetch failed: %v", err)
	}
	if string(body) != "session-data" {
		t.Errorf("expected 'session-data', got %q", string(body))
	}

	sess.UpdateCookie("new-cookie")
	if sess.Cookie != "new-cookie" {
		t.Errorf("expected cookie=new-cookie, got %q", sess.Cookie)
	}

	// Fetch again with new cookie
	ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") != "new-cookie" {
			t.Errorf("expected Cookie=new-cookie, got %q", r.Header.Get("Cookie"))
		}
		w.Write([]byte("updated"))
	}))
	defer ts2.Close()

	sess.URL = ts2.URL
	body2, err := sess.Fetch()
	if err != nil {
		t.Fatalf("Session.Fetch after update failed: %v", err)
	}
	if string(body2) != "updated" {
		t.Errorf("expected 'updated', got %q", string(body2))
	}
}

func TestFetch_Redirect(t *testing.T) {
	var redirectCount int
	var redirectSrv *httptest.Server
	redirectSrv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if redirectCount < 2 {
			redirectCount++
			http.Redirect(w, r, redirectSrv.URL+"/final", http.StatusFound)
			return
		}
		w.Write([]byte("redirected"))
	}))
	defer redirectSrv.Close()

	body, err := Fetch(redirectSrv.URL, "", "")
	if err != nil {
		t.Fatalf("Fetch with redirects failed: %v", err)
	}
	if string(body) != "redirected" {
		t.Errorf("expected 'redirected', got %q", string(body))
	}
}

func TestFetch_TooManyRedirects(t *testing.T) {
	var redirectLoopSrv *httptest.Server
	redirectLoopSrv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, redirectLoopSrv.URL, http.StatusFound)
	}))
	defer redirectLoopSrv.Close()

	_, err := Fetch(redirectLoopSrv.URL, "", "")
	if err == nil {
		t.Fatal("expected error for too many redirects")
	}
}