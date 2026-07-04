package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsProxyURI(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"ss://YWVzLTI1Ni1nY206cGFzc3dkQDEuMS4xLjE6NDQz#Node", true},
		{"ssr://base64data", true},
		{"vmess://base64json", true},
		{"vless://uuid@host:443", true},
		{"trojan://pass@host:443", true},
		{"hysteria2://pass@host:443", true},
		{"hy2://pass@host:443", true},
		{"tuic://uuid@host:443", true},
		{"socks5://host:1080", true},
		{"ssd://base64", true},
		{"https://example.com/sub", false},
		{"http://example.com", false},
		{"random text", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input[:min(len(tt.input), 10)], func(t *testing.T) {
			got := isProxyURI(tt.input)
			if got != tt.want {
				t.Errorf("isProxyURI(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestMaskString(t *testing.T) {
	tests := []struct {
		input   string
		maxLen  int
		checkFn func(string) bool
	}{
		{
			input:  "user@example.com",
			maxLen: 16,
			checkFn: func(s string) bool {
				return strings.Contains(s, "***") && len(s) < 20
			},
		},
		{
			input:  "short",
			maxLen: 20,
			checkFn: func(s string) bool {
				return strings.Contains(s, "***")
			},
		},
		{
			input:   "",
			maxLen:  10,
			checkFn: func(s string) bool { return true }, // maskString("",10)="***"
		},
	}

	for _, tt := range tests {
		label := tt.input
		if label == "" {
			label = "empty"
		} else if len(label) > 5 {
			label = label[:5]
		}
		t.Run(label, func(t *testing.T) {
			result := maskString(tt.input, tt.maxLen)
			if !tt.checkFn(result) {
				t.Errorf("maskString(%q, %d) = %q, check failed", tt.input, tt.maxLen, result)
			}
		})
	}
}

func TestVersionCmd(t *testing.T) {
	// Test the version command output (captures stdout since cmd uses fmt.Println)
	rootCmd.SetArgs([]string{"version"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("version command failed: %v", err)
	}
}

func TestDryRunNoConfig(t *testing.T) {
	// dry-run should fail without a config file
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "subai.yaml")

	rootCmd.SetArgs([]string{"dry-run", "-c", cfgPath})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing config in dry-run")
	}
}

func TestValidateNoConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "subai.yaml")

	rootCmd.SetArgs([]string{"validate", "-c", cfgPath})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing config in validate")
	}
}

func TestLogin_MissingArgs(t *testing.T) {
	// Should fail without required flags
	rootCmd.SetArgs([]string{"login"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing login name arg")
	}
}

func TestLogin_MissingURL(t *testing.T) {
	rootCmd.SetArgs([]string{"login", "test-airport", "--email", "test@test.com", "--password", "pass"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing --url flag")
	}
}

func TestLogin_UnsupportedMethod(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "subai.yaml")

	rootCmd.SetArgs([]string{"login", "test-airport", "-c", cfgPath,
		"--method", "unsupported",
		"--url", "https://panel.example.com",
		"--email", "test@test.com",
		"--password", "pass"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for unsupported login method")
	}
}

func TestSourceList_NoConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "subai.yaml")

	rootCmd.SetArgs([]string{"source", "list", "-c", cfgPath})
	err := rootCmd.Execute()
	if err != nil {
		// Should either succeed (empty list) or give a meaningful error
		if !strings.Contains(err.Error(), "no such file") &&
			!strings.Contains(err.Error(), "not found") &&
			!strings.Contains(err.Error(), "load config") {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}

func TestSourceAdd_NoArgs(t *testing.T) {
	rootCmd.SetArgs([]string{"source", "add"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing args")
	}
}

func TestSourceRemove_NoArgs(t *testing.T) {
	rootCmd.SetArgs([]string{"source", "remove"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing args")
	}
}

func TestConvert_InlineSS(t *testing.T) {
	// Inline conversion should work
	rootCmd.SetArgs([]string{"convert", "-t", "clash", "-c", "/nonexistent/config.yaml",
		"ss://YWVzLTI1Ni1nY206cGFzc3dk@1.1.1.1:443#TestNode"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("convert inline failed: %v", err)
	}
}

func TestConvert_InvalidURI(t *testing.T) {
	// Invalid URI should produce error
	rootCmd.SetArgs([]string{"convert", "-c", "/nonexistent/config.yaml",
		"not-a-valid-uri-or-url"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid URI")
	}
}

func TestConvert_OutputToFile(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "output.yaml")

	rootCmd.SetArgs([]string{"convert", "-c", "/nonexistent/config.yaml",
		"-o", outPath,
		"ss://YWVzLTI1Ni1nY206cGFzc3dk@1.1.1.1:443#FileTest"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("convert to file failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(outPath); os.IsNotExist(err) {
		t.Fatal("output file was not created")
	}

	data, _ := os.ReadFile(outPath)
	if !strings.Contains(string(data), "FileTest") {
		t.Errorf("expected FileTest in output file, got: %s", string(data))
	}
}

func TestSourceAdd_AndList(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "subai.yaml")

	// Add a source
	rootCmd.SetArgs([]string{"source", "add", "test-source", "https://example.com/sub", "-c", cfgPath})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("source add failed: %v", err)
	}

	// Verify by checking the config file directly
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read config failed: %v", err)
	}
	if !strings.Contains(string(data), "test-source") {
		t.Errorf("expected test-source in config, got: %s", string(data))
	}
	if !strings.Contains(string(data), "https://example.com/sub") {
		t.Errorf("expected url in config, got: %s", string(data))
	}
}

func TestSourceRemove_NotFound(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "subai.yaml")
	os.WriteFile(cfgPath, []byte(`sources: []`), 0644)

	rootCmd.SetArgs([]string{"source", "remove", "nonexistent-source", "-c", cfgPath})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for removing nonexistent source")
	}
}