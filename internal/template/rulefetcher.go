package template

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var ruleClient = &http.Client{
	Timeout: 15 * time.Second,
}

// FetchRuleSet downloads a rule file from URL and returns its lines (rules).
// Skips comments (#) and empty lines. Returns error if fetch fails.
func FetchRuleSet(url string) ([]string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "subai/1.0")
	req.Header.Set("Accept", "text/plain")

	resp, err := ruleClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s: HTTP %d", url, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", url, err)
	}

	var rules []string
	prevLinePayload := false
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		// Handle YAML payload format (Loyalsoldier, blackmatrix7 style):
		//   payload:
		//     - 'DOMAIN-SUFFIX,example.com'
		//     - '+.example.com'
		if line == "payload:" {
			prevLinePayload = true
			continue
		}
		if prevLinePayload || strings.HasPrefix(line, "- ") {
			prevLinePayload = false
			content := strings.TrimPrefix(line, "- ")
			content = strings.Trim(content, "'\"") // Remove surrounding quotes
			if content != "" {
				// Convert '+.example.com' (Loyalsoldier domain pattern) to Clash format
				if strings.HasPrefix(content, "+.") {
					content = "DOMAIN-SUFFIX," + strings.TrimPrefix(content, "+.")
				}
				rules = append(rules, content)
			}
			continue
		}
		rules = append(rules, line)
	}
	return rules, nil
}

// ExpandRuleSets fetches all RULE-SET URLs and returns inline rules.
// Each rule gets tagged with the corresponding group name.
func ExpandRuleSets(ruleSets []RuleSet) ([]string, []error) {
	var rules []string
	var errs []error

	for _, rs := range ruleSets {
		if rs.URL == "" {
			continue // skip inline/built-in rules
		}

		fetched, err := FetchRuleSet(rs.URL)
		if err != nil {
			errs = append(errs, fmt.Errorf("rule-set %q (%s): %w", rs.Group, rs.URL, err))
			continue
		}

		for _, line := range fetched {
			// Skip RULE-SET directives from upstream files (they reference more files)
			if strings.HasPrefix(line, "RULE-SET,") {
				continue
			}
			// Append group to each rule, avoiding double-append
			rule := line
			if !strings.HasSuffix(line, ","+rs.Group) {
				rule = line + "," + rs.Group
			}
			// Remove trailing comma if the rule already ends with a group
			// (some rule formats like GEOIP,CN already have implicit actions)
			rules = append(rules, rule)
		}
	}

	return rules, errs
}

// RuleURL returns a reliable CDN URL for a raw GitHub rule file.
// Uses jsDelivr CDN which is faster and more reliable than raw.githubusercontent.com.
func RuleURL(rawGithubURL string) string {
	// Convert https://raw.githubusercontent.com/user/repo/branch/path
	// to https://cdn.jsdelivr.net/gh/user/repo@branch/path
	if strings.Contains(rawGithubURL, "raw.githubusercontent.com") {
		parts := strings.SplitN(rawGithubURL, "/", 7)
		if len(parts) >= 7 {
			// parts[0..2] = https: + "" + raw.githubusercontent.com
			// parts[3] = user, parts[4] = repo, parts[5] = branch, parts[6+] = path
			return fmt.Sprintf("https://cdn.jsdelivr.net/gh/%s/%s@%s/%s",
				parts[3], parts[4], parts[5], strings.Join(parts[6:], "/"))
		}
	}
	return rawGithubURL
}

// AvailableRuleProjects returns a list of known rule projects with descriptions.
func AvailableRuleProjects() []map[string]string {
	return []map[string]string{
		{
			"name":        "ACL4SSR",
			"url":         "https://github.com/ACL4SSR/ACL4SSR",
			"description": "Most comprehensive Clash rules, 18.5k+ stars",
			"branch":      "master",
		},
		{
			"name":        "blackmatrix7",
			"url":         "https://github.com/blackmatrix7/ios_rule_script",
			"description": "Comprehensive rule sets for Clash, 20k+ stars",
			"branch":      "master",
		},
		{
			"name":        "Loyalsoldier",
			"url":         "https://github.com/Loyalsoldier/clash-rules",
			"description": "Simplified, well-maintained Clash rules, 10k+ stars",
			"branch":      "release",
		},
		{
			"name":        "MetaCubeX",
			"url":         "https://github.com/MetaCubeX/meta-rules",
			"description": "Official Mihomo/Clash.Meta rules",
			"branch":      "main",
		},
	}
}