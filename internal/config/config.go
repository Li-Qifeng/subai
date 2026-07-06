package config

import (
	"fmt"
	"os"

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
// Fields set in a profile replace the corresponding root-level fields
// when the profile is active. Unset fields fall through to the root config.
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
	RefreshCron  string `yaml:"refresh-cron,omitempty"`  // Cron expression for auto-refresh
	Login        *Login `yaml:"login,omitempty"`          // Automated login config
}

// Login defines the configuration for automated panel login.
// Supported methods: "v2board"
type Login struct {
	Method   string `yaml:"method"`            // Login method: "v2board"
	URL      string `yaml:"url"`               // Panel base URL, e.g. "https://www.xfltd.org"
	Email    string `yaml:"email"`              // Login email
	Password string `yaml:"password"`           // Login password
}

// Rules defines filtering rules for proxy selection.
type Rules struct {
	Include []string `yaml:"include,omitempty"`
	Exclude []string `yaml:"exclude,omitempty"`
}

// Output defines the output target configuration.
type Output struct {
	Target string `yaml:"target"`          // clash, base64, singbox, mixed
	Path   string `yaml:"path,omitempty"`  // output file path
	Pretty bool   `yaml:"pretty"`          // pretty-print output
	Template template.Config `yaml:",inline"` // template config (merged inline)
}

// Server defines the optional HTTP server configuration.
type Server struct {
	Enabled bool   `yaml:"enabled"`
	Listen  string `yaml:"listen"` // e.g. ":8080"
	Token   string `yaml:"token,omitempty"`
}

// Load reads and parses a YAML config file.
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

// Save writes the config to a YAML file.
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// Validate checks config for common errors.
func (c *Config) Validate() []error {
	var errs []error
	for i, s := range c.Sources {
		if s.Name == "" {
			errs = append(errs, fmt.Errorf("source[%d]: name is required", i))
		}
		if s.URL == "" {
			errs = append(errs, fmt.Errorf("source[%d]: url is required", i))
		}
	}
	if c.Output.Target == "" {
		errs = append(errs, fmt.Errorf("output.target is required (clash, base64, singbox, mixed)"))
	}
	// Validate profiles
	for name, p := range c.Profiles {
		for i, s := range p.Sources {
			if s.Name == "" {
				errs = append(errs, fmt.Errorf("profile[%s].sources[%d]: name is required", name, i))
			}
			if s.URL == "" {
				errs = append(errs, fmt.Errorf("profile[%s].sources[%d]: url is required", name, i))
			}
		}
	}
	return errs
}

// Resolve returns the effective config for the given profile name.
// If profileName is empty, uses CurrentProfile. Falls back to root config
// when the profile is not found or has no profile system.
func (c *Config) Resolve(profileName string) *Config {
	if profileName == "" {
		profileName = c.CurrentProfile
	}
	if profileName == "" || c.Profiles == nil {
		return c // no profile active
	}
	p, ok := c.Profiles[profileName]
	if !ok {
		return c // unknown profile, return root
	}

	// Shallow copy, then override
	resolved := *c
	if len(p.Sources) > 0 {
		resolved.Sources = p.Sources
	}
	if p.Rules != nil {
		resolved.Rules = *p.Rules
	}
	if p.Output != nil {
		resolved.Output = *p.Output
	}
	return &resolved
}

// DefaultConfig returns a sensible default configuration.
func DefaultConfig() *Config {
	return &Config{
		Output: Output{
			Target: "clash",
			Pretty: true,
		},
	}
}