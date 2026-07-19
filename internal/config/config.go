package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/user/subai/internal/template"
	"gopkg.in/yaml.v3"
)

// Config is the top-level configuration for subai.
type Config struct {
	Sources        []Source          `yaml:"sources"`
	Rules          Rules             `yaml:"rules"`
	Output         Output            `yaml:"output"`
	Server         Server            `yaml:"server,omitempty"`
	CurrentProfile string            `yaml:"current_profile,omitempty"` // active profile name
	Profiles       map[string]Profile `yaml:"profiles,omitempty"`       // named profiles
}

// Profile defines a named configuration profile that overrides the base config.
type Profile struct {
	Sources []Source `yaml:"sources,omitempty"`
	Rules   *Rules   `yaml:"rules,omitempty"`  // pointer: nil = not set, non-nil = override
	Output  *Output  `yaml:"output,omitempty"` // pointer: nil = not set, non-nil = override
}

// Source defines a subscription source.
type Source struct {
	Name         string `yaml:"name"`
	URL          string `yaml:"url"`
	Cookie       string `yaml:"cookie,omitempty"`
	UserAgent    string `yaml:"user-agent,omitempty"`
	RefreshCron  string `yaml:"refresh-cron,omitempty"`
	Login        *Login `yaml:"login,omitempty"`
}

// Login defines the configuration for automated panel login.
type Login struct {
	Method   string `yaml:"method"`
	URL      string `yaml:"url"`
	Email    string `yaml:"email"`
	Password string `yaml:"password"`
}

// Rules defines filtering rules for proxy selection.
type Rules struct {
	Include []string `yaml:"include,omitempty"`
	Exclude []string `yaml:"exclude,omitempty"`
}

// Output defines the output target configuration.
type Output struct {
	Target   string           `yaml:"target"`
	Path     string           `yaml:"path,omitempty"`
	Pretty   bool             `yaml:"pretty"`
	Template template.Config  `yaml:",inline"`
}

// Server defines the optional HTTP server configuration.
type Server struct {
	Enabled bool   `yaml:"enabled"`
	Listen  string `yaml:"listen"`
	Token   string `yaml:"token,omitempty"`
}

var validTargets = map[string]bool{"clash": true, "base64": true, "singbox": true, "mixed": true}

// Load reads and parses a YAML config file WITHOUT validation.
// Use this for management commands (source add, rule add, login, etc.)
// that need to read and modify the config even if it's incomplete.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}

// LoadAndValidate reads, parses, AND validates a config file.
// Use this for commands that actually use the config (convert, validate, serve).
func LoadAndValidate(path string) (*Config, error) {
	cfg, err := Load(path)
	if err != nil {
		return nil, err
	}
	if errs := cfg.Validate(); len(errs) > 0 {
		return nil, fmt.Errorf("invalid config: %v", errs)
	}
	return cfg, nil
}

// Save writes the config to a YAML file atomically.
func (c *Config) Save(path string) error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	dir := filepath.Dir(path)
	tmpPath := filepath.Join(dir, "."+filepath.Base(path)+".tmp")
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write temp: %w", err)
	}
	return os.Rename(tmpPath, path)
}

// Validate checks config for common errors.
func (c *Config) Validate() []error {
	if c == nil {
		return []error{fmt.Errorf("config is nil")}
	}
	var errs []error
	for i, s := range c.Sources {
		if s.Name == "" {
			errs = append(errs, fmt.Errorf("source[%d]: name is required", i))
		}
		if s.URL == "" {
			errs = append(errs, fmt.Errorf("source[%d]: url is required", i))
		}
		if s.Login != nil {
			errs = append(errs, validateLogin(s.Login, fmt.Sprintf("source[%d].login", i))...)
		}
	}
	if c.Output.Target == "" {
		errs = append(errs, fmt.Errorf("output.target is required (clash, base64, singbox, mixed)"))
	} else if !validTargets[c.Output.Target] {
		errs = append(errs, fmt.Errorf("output.target: %q is not valid (must be clash, base64, singbox, or mixed)", c.Output.Target))
	}
	if c.CurrentProfile != "" && c.Profiles != nil {
		if _, ok := c.Profiles[c.CurrentProfile]; !ok {
			errs = append(errs, fmt.Errorf("current_profile %q not found in profiles", c.CurrentProfile))
		}
	}
	for name, p := range c.Profiles {
		for i, s := range p.Sources {
			if s.Name == "" {
				errs = append(errs, fmt.Errorf("profile[%s].sources[%d]: name is required", name, i))
			}
			if s.URL == "" {
				errs = append(errs, fmt.Errorf("profile[%s].sources[%d]: url is required", name, i))
			}
			if s.Login != nil {
				errs = append(errs, validateLogin(s.Login, fmt.Sprintf("profile[%s].sources[%d].login", name, i))...)
			}
		}
		if p.Output != nil {
			if p.Output.Target != "" && !validTargets[p.Output.Target] {
				errs = append(errs, fmt.Errorf("profile[%s].output.target: %q is not valid", name, p.Output.Target))
			}
		}
	}
	if c.Server.Enabled && c.Server.Listen == "" {
		errs = append(errs, fmt.Errorf("server.listen is required when server.enabled is true"))
	}
	return errs
}

func validateLogin(l *Login, prefix string) []error {
	var errs []error
	if l.Method == "" {
		errs = append(errs, fmt.Errorf("%s.method is required", prefix))
	} else if l.Method != "v2board" {
		errs = append(errs, fmt.Errorf("%s.method: %q is not supported (only v2board)", prefix, l.Method))
	}
	if l.URL == "" {
		errs = append(errs, fmt.Errorf("%s.url is required", prefix))
	}
	if l.Email == "" {
		errs = append(errs, fmt.Errorf("%s.email is required", prefix))
	}
	if l.Password == "" {
		errs = append(errs, fmt.Errorf("%s.password is required", prefix))
	}
	return errs
}

// Resolve returns the effective config for the given profile name.
func (c *Config) Resolve(profileName string) *Config {
	if c == nil {
		return nil
	}
	if profileName == "" {
		profileName = c.CurrentProfile
	}
	if profileName == "" || c.Profiles == nil {
		return c
	}
	p, ok := c.Profiles[profileName]
	if !ok {
		return c
	}

	resolved := *c
	if len(p.Sources) > 0 {
		resolved.Sources = p.Sources
	} else if c.Sources != nil {
		resolved.Sources = make([]Source, len(c.Sources))
		copy(resolved.Sources, c.Sources)
	}
	if p.Rules != nil {
		resolved.Rules = *p.Rules
	}
	if p.Output != nil {
		merged := resolved.Output
		if p.Output.Target != "" {
			merged.Target = p.Output.Target
		}
		if p.Output.Path != "" {
			merged.Path = p.Output.Path
		}
		merged.Pretty = p.Output.Pretty
		if p.Output.Template.Template != "" {
			merged.Template.Template = p.Output.Template.Template
		}
		if len(p.Output.Template.ProxyGroups) > 0 {
			merged.Template.ProxyGroups = p.Output.Template.ProxyGroups
		}
		if len(p.Output.Template.RuleSets) > 0 {
			merged.Template.RuleSets = p.Output.Template.RuleSets
		}
		if len(p.Output.Template.RulePatches) > 0 {
			merged.Template.RulePatches = p.Output.Template.RulePatches
		}
		resolved.Output = merged
	}
	return &resolved
}

// DefaultConfig returns a sensible default configuration.
func DefaultConfig() *Config {
	return &Config{
		Output: Output{
			Target: "clash",
			Pretty: true,
			Template: template.Config{
				Template: "basic",
			},
		},
	}
}