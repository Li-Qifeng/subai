package login

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Result holds the output of a successful login.
type Result struct {
	SubscribeURL string `json:"subscribe_url"`
	Token        string `json:"token"`
	AuthData     string `json:"auth_data"`
	User         User   `json:"user"`
}

// User holds user profile information from the panel.
type User struct {
	Email          string `json:"email"`
	Plan           string `json:"plan"`
	TransferEnable int64  `json:"transfer_enable"`
	Used           int64  `json:"used"`
	ExpiredAt      any    `json:"expired_at"`
	Balance        int    `json:"balance"`
	UUID           string `json:"uuid"`
}

// Config defines the login parameters for a subscription source.
type Config struct {
	Method   string `yaml:"method"`            // "v2board"
	BaseURL  string `yaml:"url"`               // e.g. "https://www.xfltd.org"
	Email    string `yaml:"email"`              // login email
	Password string `yaml:"password"`           // login password
}

// V2Board runs the V2Board login flow via Python cloudscraper.
// It returns the subscribe URL, token, and user info.
func V2Board(baseURL, email, password string) (*Result, error) {
	// Build the input for the Python helper
	input := map[string]string{
		"method":   "v2board",
		"base_url": baseURL,
		"email":    email,
		"password": password,
	}
	inputJSON, _ := json.Marshal(input)

	// Find the helper script
	scriptPath := findScript()
	if scriptPath == "" {
		return nil, fmt.Errorf("scripts/subai_login.py not found. Ensure it's in the scripts/ directory")
	}

	// Execute Python helper
	cmd := exec.Command("python3", scriptPath)
	cmd.Stdin = strings.NewReader(string(inputJSON))
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Try to parse error from output
		var errResult struct {
			Ok    bool   `json:"ok"`
			Error string `json:"error"`
		}
		if json.Unmarshal(output, &errResult) == nil && errResult.Error != "" {
			return nil, fmt.Errorf("login failed: %s", errResult.Error)
		}
		return nil, fmt.Errorf("python helper error: %w\noutput: %s", err, string(output))
	}

	// Parse the result
	var rawResult struct {
		Ok           bool   `json:"ok"`
		SubscribeURL string `json:"subscribe_url"`
		Token        string `json:"token"`
		AuthData     string `json:"auth_data"`
		User         User   `json:"user"`
		Error        string `json:"error"`
	}
	if err := json.Unmarshal(output, &rawResult); err != nil {
		return nil, fmt.Errorf("parse login result: %w\noutput: %s", err, string(output))
	}

	if !rawResult.Ok {
		return nil, fmt.Errorf("%s\n\nTry manually obtaining the subscribe URL from the panel and add it to the config:\n  subai source add <name> <url>", rawResult.Error)
	}

	if rawResult.SubscribeURL == "" {
		return nil, fmt.Errorf("login succeeded but no subscribe URL found. Check the panel manually")
	}

	return &Result{
		SubscribeURL: rawResult.SubscribeURL,
		Token:        rawResult.Token,
		AuthData:     rawResult.AuthData,
		User:         rawResult.User,
	}, nil
}

// findScript locates the Python helper script.
// It checks relative to the binary and common install paths.
func findScript() string {
	paths := []string{
		"scripts/subai_login.py",
		"./scripts/subai_login.py",
		"../scripts/subai_login.py",
	}
	for _, p := range paths {
		if fileExists(p) {
			return p
		}
	}
	return ""
}

func fileExists(path string) bool {
	_, err := exec.Command("test", "-f", path).Output()
	return err == nil
}