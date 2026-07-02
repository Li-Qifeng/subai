package parser

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// clashConfig is the structure of a Clash YAML configuration file.
type clashConfig struct {
	Port               int                `yaml:"port,omitempty"`
	SocksPort          int                `yaml:"socks-port,omitempty"`
	RedirPort          int                `yaml:"redir-port,omitempty"`
	AllowLan           bool               `yaml:"allow-lan,omitempty"`
	Mode               string             `yaml:"mode,omitempty"`
	LogLevel           string             `yaml:"log-level,omitempty"`
	ExternalController string             `yaml:"external-controller,omitempty"`
	Proxies            []Proxy            `yaml:"proxies,omitempty"`
	ProxyGroups        []clashProxyGroup  `yaml:"proxy-groups,omitempty"`
	Rules              []string           `yaml:"rules,omitempty"`
	ProxyProviders     map[string]clashProvider `yaml:"proxy-providers,omitempty"`
	DNS                *clashDNS          `yaml:"dns,omitempty"`
	CFWConnections     string             `yaml:"cfw-connections,omitempty"`
}

type clashProxyGroup struct {
	Name     string        `yaml:"name"`
	Type     string        `yaml:"type"`
	Proxies  []string      `yaml:"proxies,omitempty"`
	URL      string        `yaml:"url,omitempty"`
	Interval int           `yaml:"interval,omitempty"`
	Use      []string      `yaml:"use,omitempty"` // proxy-provider references
}

type clashProvider struct {
	Type     string `yaml:"type"`
	URL      string `yaml:"url,omitempty"`
	Path     string `yaml:"path,omitempty"`
	Interval int    `yaml:"interval,omitempty"`
	HealthCheck *clashHealthCheck `yaml:"health-check,omitempty"`
}

type clashHealthCheck struct {
	Enable   bool   `yaml:"enable"`
	URL      string `yaml:"url"`
	Interval int    `yaml:"interval"`
}

type clashDNS struct {
	Enable         bool              `yaml:"enable"`
	IPv6           bool              `yaml:"ipv6"`
	Nameserver     []string          `yaml:"nameserver"`
	Fallback       []string          `yaml:"fallback,omitempty"`
	EnhancedMode   string            `yaml:"enhanced-mode,omitempty"`
	FakeIPRange    string            `yaml:"fake-ip-range,omitempty"`
	NameserverPolicy map[string]string `yaml:"nameserver-policy,omitempty"`
}

// ParseClashYAML parses a Clash-compatible YAML file and extracts proxies.
func ParseClashYAML(data []byte) (ProxyList, error) {
	var cfg clashConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse clash yaml: %w", err)
	}
	if cfg.Proxies == nil {
		return ProxyList{}, nil
	}
	return cfg.Proxies, nil
}

// ParseClashProxyList parses a Clash proxies list (from proxy-provider format).
func ParseClashProxyList(data []byte) (ProxyList, error) {
	var proxies ProxyList
	if err := yaml.Unmarshal(data, &proxies); err != nil {
		return nil, fmt.Errorf("parse clash proxy list: %w", err)
	}
	return proxies, nil
}

// LoadClashFromFile loads and parses a Clash YAML file.
func LoadClashFromFile(path string) (ProxyList, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read clash file: %w", err)
	}
	return ParseClashYAML(data)
}

// ParseAuto detects the input format and parses accordingly.
func ParseAuto(data []byte) (ProxyList, error) {
	str := string(data)

	// Try Clash YAML first (has "proxies:" key)
	if containsYAMLProxyList(str) {
		proxies, err := ParseClashYAML(data)
		if err == nil && len(proxies) > 0 {
			return proxies, nil
		}
		// Try proxy-provider format (just a list)
		proxies, err = ParseClashProxyList(data)
		if err == nil && len(proxies) > 0 {
			return proxies, nil
		}
	}

	// Try base64 subscription format
	uris, err := ParseSubscription(data)
	if err == nil && len(uris) > 0 {
		var result ProxyList
		for _, uri := range uris {
			p, err := ParseURI(uri)
			if err != nil {
				continue // skip unparsable
			}
			result = append(result, *p)
		}
		if len(result) > 0 {
			return result, nil
		}
	}

	// Try as plain URI (single node)
	p, err := ParseURI(str)
	if err == nil {
		return ProxyList{*p}, nil
	}

	return nil, fmt.Errorf("unable to detect input format")
}

// containsYAMLProxyList does a quick check if content looks like Clash YAML.
func containsYAMLProxyList(s string) bool {
	return containsLine(s, "proxies:") || containsLine(s, "- name:")
}

func containsLine(s, substr string) bool {
	for _, line := range strings.Split(s, "\n") {
		if strings.Contains(line, substr) {
			return true
		}
	}
	return false
}