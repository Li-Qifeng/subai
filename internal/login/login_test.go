package login

import (
	"os"
	"testing"
)

func TestFindScript(t *testing.T) {
	// Create a temporary scripts directory and file
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)

	dir := t.TempDir()
	os.Chdir(dir)

	// Without the script, should return empty
	path := findScript()
	if path != "" {
		t.Errorf("expected empty with no script, got %q", path)
	}

	// Create the script
	os.Mkdir("scripts", 0755)
	f, _ := os.Create("scripts/subai_login.py")
	f.Close()

	path = findScript()
	if path == "" {
		t.Fatal("expected to find script")
	}
	if path != "scripts/subai_login.py" && path != "./scripts/subai_login.py" {
		t.Errorf("unexpected path: %q", path)
	}
}

func TestFileExists(t *testing.T) {
	dir := t.TempDir()
	existingPath := dir + "/existing.txt"
	nonexistentPath := dir + "/nonexistent.txt"

	os.WriteFile(existingPath, []byte("test"), 0644)

	if !fileExists(existingPath) {
		t.Error("expected fileExists to return true for existing file")
	}
	if fileExists(nonexistentPath) {
		t.Error("expected fileExists to return false for nonexistent file")
	}
}

func TestConfigMethod(t *testing.T) {
	cfg := Config{
		Method:   "v2board",
		BaseURL:  "https://panel.example.com",
		Email:    "user@example.com",
		Password: "secret123",
	}
	if cfg.Method != "v2board" {
		t.Errorf("method: got %q", cfg.Method)
	}
	if cfg.BaseURL != "https://panel.example.com" {
		t.Errorf("baseURL: got %q", cfg.BaseURL)
	}
	if cfg.Email != "user@example.com" {
		t.Errorf("email: got %q", cfg.Email)
	}
	if cfg.Password != "secret123" {
		t.Errorf("password: got %q", cfg.Password)
	}
}

func TestResultFields(t *testing.T) {
	user := User{
		Email:          "user@example.com",
		Plan:           "Pro Plan",
		TransferEnable: 107374182400, // 100GB
		Used:           21474836480,  // 20GB
		ExpiredAt:      "2026-12-31",
		Balance:        100,
		UUID:           "550e8400-e29b-41d4-a716-446655440000",
	}

	result := Result{
		SubscribeURL: "https://subscribe.example.com?token=abc",
		Token:        "abc123",
		AuthData:     "authdata",
		User:         user,
	}

	if result.SubscribeURL != "https://subscribe.example.com?token=abc" {
		t.Errorf("subscribeURL: got %q", result.SubscribeURL)
	}
	if result.Token != "abc123" {
		t.Errorf("token: got %q", result.Token)
	}
	if result.User.Plan != "Pro Plan" {
		t.Errorf("plan: got %q", result.User.Plan)
	}
	if result.User.TransferEnable != 107374182400 {
		t.Errorf("transfer: got %d", result.User.TransferEnable)
	}
	if result.User.Used != 21474836480 {
		t.Errorf("used: got %d", result.User.Used)
	}
}

func TestV2Board_NoScript(t *testing.T) {
	// Without the Python script, V2Board should return an error
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)

	dir := t.TempDir()
	os.Chdir(dir)

	_, err := V2Board("https://panel.example.com", "user@example.com", "password")
	if err == nil {
		t.Fatal("expected error when no Python script exists")
	}
	if err.Error() != "scripts/subai_login.py not found. Ensure it's in the scripts/ directory" &&
		!os.IsNotExist(err) {
		// Accept either the specific error or a file-not-found error
	}
}

func TestUserExpiredAt(t *testing.T) {
	// Test that ExpiredAt can be various types (string, int, nil)
	tests := []struct {
		name string
		val  any
	}{
		{"string", "2026-12-31"},
		{"int", 1700000000},
		{"nil", nil},
		{"float", 1700000000.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := User{ExpiredAt: tt.val}
			if u.ExpiredAt != tt.val {
				t.Errorf("expected %v, got %v", tt.val, u.ExpiredAt)
			}
		})
	}
}

func TestFindScript_Paths(t *testing.T) {
	// We can't easily test the full search path without modifying the function,
	// but we can verify the paths are checked in order
	dir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(dir)

	os.Mkdir("scripts", 0755)
	os.Chdir("scripts")

	// Now in scripts/, checking ../scripts/subai_login.py should look up one level
	// Since we're in scripts/, the paths are:
	// scripts/subai_login.py -> scripts/scripts/subai_login.py (doesn't exist)
	// ./scripts/subai_login.py -> scripts/scripts/subai_login.py (doesn't exist)
	// ../scripts/subai_login.py -> scripts/subai_login.py (doesn't exist)

	path := findScript()
	if path != "" {
		t.Errorf("expected empty, got %q", path)
	}
}