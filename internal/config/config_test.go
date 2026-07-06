package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Output.Target != "clash" {
		t.Errorf("expected target=clash, got %q", cfg.Output.Target)
	}
	if !cfg.Output.Pretty {
		t.Error("expected pretty=true")
	}
	if cfg.Sources != nil {
		t.Errorf("expected nil sources, got %d", len(cfg.Sources))
	}
}

func TestLoadAndSave(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "subai.yaml")

	orig := &Config{
		Sources: []Source{
			{Name: "test", URL: "https://example.com/sub", UserAgent: "test-agent"},
		},
		Rules: Rules{
			Include: []string{"HK|Hong"},
			Exclude: []string{"过期"},
		},
		Output: Output{
			Target: "base64",
			Pretty: false,
		},
	}

	if err := orig.Save(cfgPath); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(loaded.Sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(loaded.Sources))
	}
	if loaded.Sources[0].Name != "test" {
		t.Errorf("expected name=test, got %q", loaded.Sources[0].Name)
	}
	if loaded.Sources[0].URL != "https://example.com/sub" {
		t.Errorf("expected url=https://example.com/sub, got %q", loaded.Sources[0].URL)
	}
	if loaded.Sources[0].UserAgent != "test-agent" {
		t.Errorf("expected ua=test-agent, got %q", loaded.Sources[0].UserAgent)
	}
	if loaded.Output.Target != "base64" {
		t.Errorf("expected target=base64, got %q", loaded.Output.Target)
	}
	if loaded.Output.Pretty {
		t.Error("expected pretty=false")
	}
	if len(loaded.Rules.Include) != 1 || loaded.Rules.Include[0] != "HK|Hong" {
		t.Errorf("include rules mismatch: %v", loaded.Rules.Include)
	}
	if len(loaded.Rules.Exclude) != 1 || loaded.Rules.Exclude[0] != "过期" {
		t.Errorf("exclude rules mismatch: %v", loaded.Rules.Exclude)
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/subai.yaml")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	badPath := filepath.Join(dir, "bad.yaml")
	os.WriteFile(badPath, []byte("invalid: yaml: [broken"), 0644)

	_, err := Load(badPath)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestValidate_Valid(t *testing.T) {
	cfg := &Config{
		Sources: []Source{
			{Name: "s1", URL: "https://example.com/sub"},
		},
		Output: Output{Target: "clash"},
	}
	errs := cfg.Validate()
	if len(errs) != 0 {
		t.Fatalf("expected 0 errors, got %d: %v", len(errs), errs)
	}
}

func TestValidate_MissingFields(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *Config
		wantErrs int
	}{
		{
			name:     "empty source name",
			cfg:      &Config{Sources: []Source{{URL: "https://example.com/sub"}}, Output: Output{Target: "clash"}},
			wantErrs: 1,
		},
		{
			name:     "empty source url",
			cfg:      &Config{Sources: []Source{{Name: "s1"}}, Output: Output{Target: "clash"}},
			wantErrs: 1,
		},
		{
			name:     "empty target",
			cfg:      &Config{Sources: []Source{{Name: "s1", URL: "https://example.com/sub"}}, Output: Output{}},
			wantErrs: 1,
		},
		{
			name:     "multiple errors",
			cfg:      &Config{Sources: []Source{{Name: "", URL: ""}}, Output: Output{}},
			wantErrs: 3, // source name + source url + target
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := tt.cfg.Validate()
			if len(errs) != tt.wantErrs {
				t.Errorf("expected %d errors, got %d: %v", tt.wantErrs, len(errs), errs)
			}
		})
	}
}

func TestLoad_RealConfig(t *testing.T) {
	// Use the testdata config to verify real-world loading
	cfg, err := Load("../../testdata/subai.yaml")
	if err != nil {
		t.Fatalf("Load testdata config failed: %v", err)
	}
	if len(cfg.Sources) != 1 {
		t.Errorf("expected 1 source, got %d", len(cfg.Sources))
	}
	if cfg.Sources[0].Name != "Free-Test" {
		t.Errorf("expected name=Free-Test, got %q", cfg.Sources[0].Name)
	}
	if cfg.Output.Target != "clash" {
		t.Errorf("expected target=clash, got %q", cfg.Output.Target)
	}
	if !cfg.Output.Pretty {
		t.Error("expected pretty=true")
	}
	if cfg.Server.Enabled != false {
		t.Error("expected server.enabled=false")
	}
}

func TestSaveAndReloadRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "roundtrip.yaml")

	cfg := &Config{
		Sources: []Source{
			{
				Name:      "test-airport",
				URL:       "https://sub.example.com/api/v1",
				UserAgent: "ClashMeta/1.18",
				Cookie:    "session=abc123",
				Login: &Login{
					Method:   "v2board",
					URL:      "https://panel.example.com",
					Email:    "user@example.com",
					Password: "secret123",
				},
			},
		},
		Rules: Rules{
			Include: []string{"🇭🇰|HK|Hong", "🇯🇵|JP|Japan"},
			Exclude: []string{"过期", "剩余流量"},
		},
		Output: Output{
			Target: "clash",
			Pretty: true,
		},
		Server: Server{
			Enabled: true,
			Listen:  ":9090",
			Token:   "mytoken",
		},
	}

	if err := cfg.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded.Sources) != 1 {
		t.Fatalf("sources: expected 1, got %d", len(loaded.Sources))
	}
	src := loaded.Sources[0]
	if src.Name != "test-airport" || src.URL != "https://sub.example.com/api/v1" {
		t.Errorf("source mismatch: %+v", src)
	}
	if src.Login == nil || src.Login.Method != "v2board" || src.Login.Email != "user@example.com" {
		t.Errorf("login mismatch: %+v", src.Login)
	}
	if !loaded.Server.Enabled || loaded.Server.Listen != ":9090" || loaded.Server.Token != "mytoken" {
		t.Errorf("server config mismatch: %+v", loaded.Server)
	}
}

func TestSourceOmitEmpty(t *testing.T) {
	// Verify that empty optional fields are omitted on marshal
	cfg := &Config{
		Sources: []Source{
			{Name: "test", URL: "https://example.com/sub"},
		},
		Output: Output{Target: "clash"},
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "omitempty.yaml")
	if err := cfg.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, _ := os.ReadFile(path)
	content := string(data)

	if contains(content, "cookie:") {
		t.Error("expected cookie to be omitted")
	}
	if contains(content, "login:") {
		t.Error("expected login to be omitted")
	}
	if contains(content, "server:") {
		t.Error("expected server to be omitted")
	}
}

// --- Profile tests ---

func TestProfile_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profile.yaml")

	cfg := &Config{
		Sources: []Source{
			{Name: "default-source", URL: "https://default.example.com/sub"},
		},
		Output: Output{Target: "clash", Pretty: true},
		CurrentProfile: "mobile",
		Profiles: map[string]Profile{
			"mobile": {
				Sources: []Source{
					{Name: "mobile-source", URL: "https://mobile.example.com/sub"},
				},
				Output: &Output{Target: "clash", Pretty: false},
			},
			"home": {
				Output: &Output{Target: "base64"},
			},
		},
	}

	if err := cfg.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.CurrentProfile != "mobile" {
		t.Errorf("expected current_profile=mobile, got %q", loaded.CurrentProfile)
	}
	if len(loaded.Profiles) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(loaded.Profiles))
	}
	if _, ok := loaded.Profiles["mobile"]; !ok {
		t.Error("expected 'mobile' profile")
	}
	if _, ok := loaded.Profiles["home"]; !ok {
		t.Error("expected 'home' profile")
	}
}

func TestResolve_NoProfile(t *testing.T) {
	cfg := &Config{
		Sources: []Source{{Name: "s1", URL: "https://example.com/sub"}},
		Output:  Output{Target: "clash"},
	}

	resolved := cfg.Resolve("")
	if resolved != cfg {
		t.Error("Resolve('') should return the same config when no profile is set")
	}
	if len(resolved.Sources) != 1 {
		t.Errorf("expected 1 source, got %d", len(resolved.Sources))
	}
}

func TestResolve_WithProfile(t *testing.T) {
	cfg := &Config{
		Sources: []Source{{Name: "default", URL: "https://default.example.com/sub"}},
		Output:  Output{Target: "clash", Pretty: true},
		Profiles: map[string]Profile{
			"mobile": {
				Sources: []Source{{Name: "mobile", URL: "https://mobile.example.com/sub"}},
				Output:  &Output{Target: "clash", Pretty: false},
			},
		},
	}

	resolved := cfg.Resolve("mobile")
	if resolved == cfg {
		t.Fatal("Resolve should return a new config, not the original")
	}

	// Sources should be overridden
	if len(resolved.Sources) != 1 || resolved.Sources[0].Name != "mobile" {
		t.Errorf("expected mobile source, got %+v", resolved.Sources)
	}

	// Output should be overridden
	if resolved.Output.Pretty != false {
		t.Error("expected pretty=false from profile")
	}
}

func TestResolve_Profile_PartialOverride(t *testing.T) {
	// Profile only overrides output, sources remain from root
	cfg := &Config{
		Sources: []Source{{Name: "default", URL: "https://default.example.com/sub"}},
		Rules:   Rules{Include: []string{"HK"}},
		Output:  Output{Target: "clash", Pretty: true},
		Profiles: map[string]Profile{
			"no-sources": {
				Output: &Output{Target: "base64", Pretty: false},
			},
		},
	}

	resolved := cfg.Resolve("no-sources")

	// Sources should remain from root
	if len(resolved.Sources) != 1 || resolved.Sources[0].Name != "default" {
		t.Errorf("expected sources from root, got %+v", resolved.Sources)
	}

	// Rules should remain from root (profile didn't set Rules)
	if len(resolved.Rules.Include) != 1 || resolved.Rules.Include[0] != "HK" {
		t.Errorf("expected rules from root, got %+v", resolved.Rules)
	}

	// Output should be from profile
	if resolved.Output.Target != "base64" {
		t.Errorf("expected target=base64 from profile, got %q", resolved.Output.Target)
	}
}

func TestResolve_UnknownProfile(t *testing.T) {
	cfg := &Config{
		Sources: []Source{{Name: "s1", URL: "https://example.com/sub"}},
		Output:  Output{Target: "clash"},
		Profiles: map[string]Profile{
			"known": {},
		},
	}

	resolved := cfg.Resolve("unknown")
	if resolved != cfg {
		t.Error("Resolve with unknown profile should return the original config")
	}
}

func TestResolve_ProfileWithRules(t *testing.T) {
	cfg := &Config{
		Sources: []Source{{Name: "s1", URL: "https://example.com/sub"}},
		Rules:   Rules{Include: []string{"HK"}},
		Output:  Output{Target: "clash"},
		Profiles: map[string]Profile{
			"strict": {
				Rules: &Rules{Include: []string{"🇭🇰"}, Exclude: []string{"过期"}},
			},
		},
	}

	resolved := cfg.Resolve("strict")
	if len(resolved.Rules.Include) != 1 || resolved.Rules.Include[0] != "🇭🇰" {
		t.Errorf("expected rules from profile, got %+v", resolved.Rules)
	}
}

func TestCurrentProfile_UsedByDefault(t *testing.T) {
	cfg := &Config{
		Sources: []Source{{Name: "default", URL: "https://default.example.com/sub"}},
		Output:  Output{Target: "clash"},
		CurrentProfile: "mobile",
		Profiles: map[string]Profile{
			"mobile": {
				Sources: []Source{{Name: "mobile-src", URL: "https://mobile.example.com/sub"}},
			},
		},
	}

	resolved := cfg.Resolve("") // empty → uses CurrentProfile
	if len(resolved.Sources) != 1 || resolved.Sources[0].Name != "mobile-src" {
		t.Errorf("expected mobile sources from CurrentProfile, got %+v", resolved.Sources)
	}
}

func TestValidate_ProfileErrors(t *testing.T) {
	cfg := &Config{
		Output: Output{Target: "clash"},
		Profiles: map[string]Profile{
			"bad": {
				Sources: []Source{
					{Name: "", URL: ""}, // missing both name and url
				},
			},
		},
	}

	errs := cfg.Validate()
	// Should have 2 profile source errors (name + url)
	if len(errs) < 2 {
		t.Fatalf("expected at least 2 errors, got %d: %v", len(errs), errs)
	}
	hasProfileErr := false
	for _, e := range errs {
		if contains(e.Error(), "profile[bad]") {
			hasProfileErr = true
			break
		}
	}
	if !hasProfileErr {
		t.Errorf("expected profile validation error, got: %v", errs)
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}