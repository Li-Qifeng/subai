package template

import (
	"fmt"
	"strconv"
	"strings"
)

// parseINITemplate converts Aethersailor INI format content to a Config struct.
// The INI format has:
//   [custom] section with ruleset= directives
//   custom_proxy_group=name`type`args...
//   enable_rule_generator=true
//   overwrite_original_rules=true
func parseINITemplate(body []byte, name string) (*Config, error) {
	lines := strings.Split(string(body), "\n")
	cfg := &Config{
		Template: name,
	}

	inCustom := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, ";") {
			continue
		}

		switch {
		case line == "[custom]":
			inCustom = true
		case strings.HasPrefix(line, "["):
			inCustom = false
		case inCustom && strings.HasPrefix(line, "ruleset="):
			rs, err := parseRuleSet(line)
			if err != nil {
				return nil, fmt.Errorf("parse ruleset: %w", err)
			}
			if rs != nil {
				cfg.RuleSets = append(cfg.RuleSets, *rs)
			}
		case inCustom && strings.HasPrefix(line, "custom_proxy_group="):
			pg, err := parseCustomProxyGroup(line)
			if err != nil {
				return nil, fmt.Errorf("parse proxy group: %w", err)
			}
			if pg != nil {
				cfg.ProxyGroups = append(cfg.ProxyGroups, *pg)
			}
		case inCustom && line == "enable_rule_generator=true":
			cfg.FetchRules = true
		case inCustom && line == "overwrite_original_rules=true":
			// This is the default behavior for our converter
		}
	}

	return cfg, nil
}

// parseRuleSet parses a ruleset= line.
// Format: ruleset=group_name,type:url,interval
// Or: ruleset=group_name,[]GEOSITE,domain
// Or: ruleset=group_name,[]GEOIP,ip,no-resolve
func parseRuleSet(line string) (*RuleSet, error) {
	// Strip "ruleset=" prefix
	content := strings.TrimPrefix(line, "ruleset=")
	parts := strings.SplitN(content, ",", 3)
	if len(parts) < 2 {
		return nil, nil
	}

	rs := &RuleSet{
		Group: parts[0],
	}

	ruleType := parts[1]
	ruleValue := ""
	if len(parts) > 2 {
		ruleValue = parts[2]
		// Strip trailing interval (e.g. "28800")
		if idx := strings.Index(ruleValue, ","); idx >= 0 {
			ruleValue = ruleValue[:idx]
		}
	}

	switch {
	case strings.HasPrefix(ruleType, "[]GEOSITE"):
		rs.BuiltIn = "geosite:" + ruleValue
	case strings.HasPrefix(ruleType, "[]GEOIP"):
		rs.BuiltIn = "geoip:" + ruleValue
	case strings.HasPrefix(ruleType, "[]FINAL"):
		rs.Rule = "MATCH," + rs.Group
	case strings.HasPrefix(ruleType, "https://") || strings.HasPrefix(ruleType, "http://"):
		// Raw URL (ACL4SSR format)
		rs.URL = ruleType
		// ruleValue might be a comma-separated interval, ignore
	case strings.HasPrefix(ruleType, "clash-domain:"):
		rs.URL = strings.TrimPrefix(ruleType, "clash-domain:")
	case strings.HasPrefix(ruleType, "clash-classic:"):
		rs.URL = strings.TrimPrefix(ruleType, "clash-classic:")
	default:
		// Unknown type, skip
		return nil, nil
	}

	return rs, nil
}

// parseCustomProxyGroup parses a custom_proxy_group= line.
// Format: name`type`args...
// For select: name`select`[]group1`[]group2`...`.*
// For url-test: name`url-test`filter`url`interval`tolerance
// For fallback: name`fallback`filter`url`interval`tolerance
// For load-balance: name`load-balance`filter`url`interval
func parseCustomProxyGroup(line string) (*ProxyGroup, error) {
	content := strings.TrimPrefix(line, "custom_proxy_group=")
	parts := strings.Split(content, "`")
	if len(parts) < 3 {
		return nil, fmt.Errorf("too few parts: %q", line)
	}

	pg := &ProxyGroup{
		Name: parts[0],
		Type: parts[1],
	}

	args := parts[2:]

	switch pg.Type {
	case "select":
		// Members are all args that start with []
		// Last arg might be .* (all proxies fallback)
		for _, arg := range args {
			arg = strings.TrimSpace(arg)
			if arg == "" {
				continue
			}
			if strings.HasPrefix(arg, "[]") {
				// []DIRECT → DIRECT, []group → group
				member := strings.TrimPrefix(arg, "[]")
				pg.Proxies = append(pg.Proxies, member)
			} else if arg == ".*" {
				// All proxies - handled by enrichment
				// No need to add as member
			}
		}
	case "url-test", "fallback":
		// Format: name`type`filter`url`interval`tolerance
		if len(args) >= 1 {
			pg.Filter = strings.TrimSpace(args[0])
		}
		if len(args) >= 2 {
			pg.URL = strings.TrimSpace(args[1])
		}
		if len(args) >= 3 {
			// Interval might be like "300,,50" (interval,tolerance,unused)
			intervalStr := strings.TrimSpace(args[2])
			if idx := strings.Index(intervalStr, ","); idx >= 0 {
				intervalStr = intervalStr[:idx]
			}
			if interval, err := parseInt(intervalStr); err == nil {
				pg.Interval = interval
			}
		}
	case "load-balance":
		// Format: name`load-balance`filter`url`interval
		if len(args) >= 1 {
			pg.Filter = strings.TrimSpace(args[0])
		}
		if len(args) >= 2 {
			pg.URL = strings.TrimSpace(args[1])
		}
		if len(args) >= 3 {
			if interval, err := parseInt(strings.TrimSpace(args[2])); err == nil {
				pg.Interval = interval
			}
		}
	}

	return pg, nil
}

// parseInt parses an integer from a string, handling empty strings.
func parseInt(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty string")
	}
	return strconv.Atoi(s)
}