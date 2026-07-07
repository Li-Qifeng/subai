package template

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/user/subai/internal/parser"
)

// ProxyGroups are custom proxy groups (used when Template is empty or "custom")
type ProxyGroup struct {
	Name     string   `yaml:"name"`
	Type     string   `yaml:"type"`               // select, url-test, fallback, load-balance, relay
	Proxies  []string `yaml:"proxies,omitempty"`   // proxy names, "*" = all
	Filter   string   `yaml:"filter,omitempty"`    // regex to auto-select proxies
	URL      string   `yaml:"url,omitempty"`       // test URL for url-test/fallback
	Interval int      `yaml:"interval,omitempty"`  // test interval in seconds
}

// RuleSet defines a rule that maps traffic to a proxy group.
type RuleSet struct {
	Group        string `yaml:"group"`                  // target proxy group name
	URL          string `yaml:"url,omitempty"`           // URL to rule file (RULE-SET)
	Rule         string `yaml:"rule,omitempty"`           // inline rule like "GEOIP,CN,DIRECT" or "MATCH,Proxy"
	BuiltIn      string `yaml:"built_in,omitempty"`      // built-in reference like "geosite:category-ads-all"
	ProviderName string `yaml:"provider_name,omitempty"` // explicit rule-provider name (auto-gen if empty)
}

// RulePatch defines a custom rule inserted at a specific position.
// Position formats:
//
//	"top"               — insert at the beginning of the rules list
//	"bottom"            — insert at the end of the rules list
//	"before:<target>"   — insert before the rule matching target (URL/name substring)
//	"after:<target>"    — insert after the rule matching target
type RulePatch struct {
	ID       string `yaml:"id"`                // unique patch identifier
	Position string `yaml:"position"`           // "top", "bottom", "before:X", "after:X"
	Rule     string `yaml:"rule"`               // rule line to insert, e.g. "DOMAIN-SUFFIX,example.com,Proxy"
}

// Config is the template configuration embedded in subai.yaml.
type Config struct {
	// Built-in template name: "basic", "acl4ssr_full", "acl4ssr_lite", etc.
	Template string `yaml:"template,omitempty"`

	// FetchRules controls whether RULE-SET URLs are fetched and expanded
	// into inline rules during conversion. Default: false (RULE-SET references).
	// When true, rule content is downloaded from source URLs and embedded directly.
	FetchRules bool `yaml:"fetch_rules"`

	// RuleProviders enables Clash.Meta rule-providers output.
	// When true, generates rule-providers section + RULE-SET,name,group references.
	// Only applies when FetchRules is false. Default: false.
	RuleProviders bool `yaml:"rule_providers"`

	// ProviderInterval is the update interval for rule-providers in seconds.
	// Default: 86400 (24 hours).
	ProviderInterval int `yaml:"provider_interval,omitempty"`

	// ProviderProxy is the optional proxy to use for downloading rule files.
	ProviderProxy string `yaml:"provider_proxy,omitempty"`

	// Custom proxy groups (used when Template is empty or "custom")
	ProxyGroups []ProxyGroup `yaml:"proxy_groups,omitempty"`
	RuleSets    []RuleSet    `yaml:"rule_sets,omitempty"`

	// RulePatches are custom inline rules inserted at specific positions
	// in the generated rules list. Applied in config order after main rules are built.
	RulePatches []RulePatch `yaml:"rule_patches,omitempty"`
}

// BuildResult holds the generated proxy groups and rules.
type BuildResult struct {
	ProxyGroups   []map[string]interface{}
	Rules         []string
	RuleProviders []map[string]interface{} // rule-providers config (Clash.Meta)
}

// Builtin returns a built-in template config by name.
// It checks the local cache first (if synced), then falls back to the compiled-in template.
func Builtin(name string) (*Config, error) {
	// Try cached template first (from remote sync)
	if cfg, err := LoadCachedTemplate(name); err == nil {
		return cfg, nil
	}

	// Fall back to compiled-in builtins
	switch name {
	case "basic":
		return basicTemplate(), nil
	case "acl4ssr_full":
		return acl4ssrFullTemplate(), nil
	case "acl4ssr_lite":
		return acl4ssrLiteTemplate(), nil
	case "loyalsoldier":
		return loyalsoldierTemplate(), nil
	default:
		return nil, fmt.Errorf("unknown template: %q (available: basic, acl4ssr_full, acl4ssr_lite, loyalsoldier, or run 'subai template sync' to fetch more)", name)
	}
}

// AvailableCachedTemplates returns all available template names including cached remote ones.
func AvailableCachedTemplates() []string {
	builtin := AvailableTemplates()

	cached, err := ListCachedTemplates()
	if err != nil || cached == nil {
		return builtin
	}

	// Merge, deduplicate
	seen := make(map[string]bool)
	var all []string
	for _, name := range builtin {
		seen[name] = true
		all = append(all, name)
	}
	for _, entry := range cached {
		if !seen[entry.Name] {
			seen[entry.Name] = true
			all = append(all, entry.Name)
		}
	}
	return all
}

// Build generates proxy groups and rules from a template config and proxy list.
// Three modes:
//   cfg.FetchRules=true     → fetch rule files, expand inline (existing)
//   cfg.RuleProviders=true  → generate rule-providers + RULE-SET,name,group
//   both false              → RULE-SET,url,group references (existing fallback)
func Build(cfg *Config, proxies []parser.Proxy) *BuildResult {
	result := &BuildResult{
		ProxyGroups: []map[string]interface{}{},
		Rules:       []string{},
		RuleProviders: []map[string]interface{}{},
	}

	allNames := make([]string, len(proxies))
	for i, p := range proxies {
		if p.Name != "" {
			allNames[i] = p.Name
		} else {
			allNames[i] = fmt.Sprintf("%s-%d", p.Server, p.Port)
		}
	}

	for _, pg := range cfg.ProxyGroups {
		group := map[string]interface{}{
			"name": pg.Name,
			"type": pg.Type,
		}

		// Determine members: static list or filter
		if len(pg.Proxies) > 0 {
			// Expand "*" to all proxy names
			var members []string
			for _, m := range pg.Proxies {
				if m == "*" {
					members = append(members, allNames...)
				} else if m == "[]DIRECT" {
					members = append(members, "DIRECT")
				} else if m == "[]REJECT" {
					members = append(members, "REJECT")
				} else {
					members = append(members, m)
				}
			}
			group["proxies"] = members
		} else if pg.Filter != "" {
			// Filter proxies by regex
			re, err := regexp.Compile("(?i)" + pg.Filter)
			if err == nil {
				var matched []string
				for _, name := range allNames {
					if re.MatchString(name) {
						matched = append(matched, name)
					}
				}
				if len(matched) > 0 {
					group["proxies"] = matched
				} else {
					group["proxies"] = allNames
				}
			}
		}

		if pg.URL != "" {
			group["url"] = pg.URL
		}
		if pg.Interval > 0 {
			group["interval"] = pg.Interval
		}

		result.ProxyGroups = append(result.ProxyGroups, group)
	}

	// Build rules: three modes
	if cfg.FetchRules {
		// Mode 1: fetch and expand all RULE-SET URLs into inline rules
		inlineRules, errs := ExpandRuleSets(cfg.RuleSets)
		for _, err := range errs {
			fmt.Fprintf(ruleLogWriter, "  ⚠️  %v\n", err)
		}
		result.Rules = append(result.Rules, inlineRules...)

		// Add non-URL rule sets (inline rules only, URL-based already expanded)
		for _, rs := range cfg.RuleSets {
			if rs.URL != "" {
				continue
			}
			rule := buildRuleLine(rs)
			if rule != "" {
				result.Rules = append(result.Rules, rule)
			}
		}
	} else if cfg.RuleProviders {
		// Mode 2: generate rule-providers + RULE-SET,provider,group
		for _, rs := range cfg.RuleSets {
			if rs.URL != "" {
				name := rs.ProviderName
				if name == "" {
					name = GenerateProviderName(rs.URL)
				}

				// Add RULE-SET,provider,group rule
				result.Rules = append(result.Rules, fmt.Sprintf("RULE-SET,%s,%s", name, rs.Group))

				// Build provider config
				interval := cfg.ProviderInterval
				if interval <= 0 {
					interval = 86400
				}
				provider := map[string]interface{}{
					"type":     "http",
					"behavior": InferBehavior(rs.URL),
					"url":      rs.URL,
					"interval": interval,
				}
				if cfg.ProviderProxy != "" {
					provider["proxy"] = cfg.ProviderProxy
				}
				// Nest under provider name
				result.RuleProviders = append(result.RuleProviders, map[string]interface{}{
					name: provider,
				})
			} else {
				// Inline / built-in rules (no provider URL)
				rule := buildRuleLine(rs)
				if rule != "" {
					result.Rules = append(result.Rules, rule)
				}
			}
		}
	} else {
		// Mode 3: RULE-SET,url,group references (legacy, no providers)
		for _, rs := range cfg.RuleSets {
			rule := buildRuleLine(rs)
			if rule != "" {
				result.Rules = append(result.Rules, rule)
			}
		}
	}

	// Apply rule patches (inline rule insertions)
	if len(cfg.RulePatches) > 0 {
		result.Rules = ApplyPatches(result.Rules, cfg.RulePatches)
	}

	return result
}

// ruleLogWriter is used for logging rule fetch warnings.
// Defaults to a no-op writer; can be replaced for testing.
var ruleLogWriter io.Writer = &noopWriter{}

// SetLogWriter sets the writer for rule fetch logs. Pass os.Stderr for CLI output.
func SetLogWriter(w io.Writer) {
	ruleLogWriter = w
}

// buildRuleLine generates a single rule line from a RuleSet.
func buildRuleLine(rs RuleSet) string {
	if rs.URL != "" {
		return fmt.Sprintf("RULE-SET,%s,%s", rs.URL, rs.Group)
	}
	if rs.BuiltIn != "" {
		return fmt.Sprintf("RULE-SET,%s,%s", rs.BuiltIn, rs.Group)
	}
	if rs.Rule != "" {
		return rs.Rule
	}
	return ""
}

// GenerateProviderName auto-generates a rule-provider name from a URL.
// Extracts meaningful keywords from the URL path.
func GenerateProviderName(rawURL string) string {
	// Remove query string
	if idx := strings.Index(rawURL, "?"); idx >= 0 {
		rawURL = rawURL[:idx]
	}
	// Split by / and take the last meaningful segment
	parts := strings.Split(rawURL, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		part := strings.TrimSpace(parts[i])
		if part == "" {
			continue
		}
		// Remove file extension
		if idx := strings.Index(part, "."); idx >= 0 {
			part = part[:idx]
		}
		// Remove version/commit-like segments
		if len(part) < 3 || strings.Contains(part, "@") {
			continue
		}
		// Convert to lowercase snake_case
		name := strings.ToLower(part)
		name = strings.NewReplacer("-", "_", " ", "_").Replace(name)
		// Prepend project name if available
		return name
	}
	// Fallback: use last path segment
	return "ruleset"
}

// InferBehavior determines the rule-provider behavior from a URL.
// Uses heuristics based on common rule file naming conventions.
func InferBehavior(rawURL string) string {
	lower := strings.ToLower(rawURL)
	// IP/CIDR files
	if strings.Contains(lower, "ip") || strings.Contains(lower, "cidr") {
		return "ipcidr"
	}
	// Domain-only files
	if strings.Contains(lower, "domain") {
		return "domain"
	}
	// Default: classical (safe for mixed rule files)
	return "classical"
}

// ApplyPatches inserts rule patches into the rules slice in place.
// Patches are applied in order: "top" → "before:X" → "after:X" → "bottom".
func ApplyPatches(rules []string, patches []RulePatch) []string {
	if len(patches) == 0 {
		return rules
	}

	// Separate patches by type
	var tops, bottoms []RulePatch
	type positionedPatch struct {
		patch    RulePatch
		isBefore bool // false = after
	}
	var positioned []positionedPatch

	for _, p := range patches {
		switch {
		case p.Position == "top" || p.Position == "beginning":
			tops = append(tops, p)
		case p.Position == "bottom" || p.Position == "end":
			bottoms = append(bottoms, p)
		case strings.HasPrefix(p.Position, "before:") || strings.HasPrefix(p.Position, "before "):
			target := strings.TrimPrefix(p.Position, "before:")
			target = strings.TrimPrefix(target, "before ")
			target = strings.TrimSpace(target)
			positioned = append(positioned, positionedPatch{patch: p, isBefore: true})
			_ = target // evaluated at apply time
		case strings.HasPrefix(p.Position, "after:") || strings.HasPrefix(p.Position, "after "):
			target := strings.TrimPrefix(p.Position, "after:")
			target = strings.TrimPrefix(target, "after ")
			target = strings.TrimSpace(target)
			positioned = append(positioned, positionedPatch{patch: p, isBefore: false})
			_ = target
		default:
			fmt.Fprintf(ruleLogWriter, "  ⚠️  RulePatch %q: unknown position %q (use top/bottom/before:X/after:X)\n", p.ID, p.Position)
		}
	}

	result := make([]string, 0, len(rules)+len(patches))

	// 1. Tops
	for _, p := range tops {
		if p.Rule != "" {
			result = append(result, p.Rule)
		}
	}

	// 2. Main rules with positioned patches
	used := make([]bool, len(positioned))
	for _, rule := range rules {
		// Check if any "before:X" patches target this rule
		for i, pp := range positioned {
			if pp.isBefore && !used[i] && matchesPatchTarget(rule, patchTarget(pp.patch)) {
				result = append(result, pp.patch.Rule)
				used[i] = true
			}
		}

		result = append(result, rule)

		// Check if any "after:X" patches target this rule
		for i, pp := range positioned {
			if !pp.isBefore && !used[i] && matchesPatchTarget(rule, patchTarget(pp.patch)) {
				result = append(result, pp.patch.Rule)
				used[i] = true
			}
		}
	}

	// 3. Unmatched positioned patches go to bottom
	for i, pp := range positioned {
		if !used[i] {
			if pp.patch.Rule != "" {
				result = append(result, pp.patch.Rule)
			}
		}
	}

	// 4. Bottoms
	for _, p := range bottoms {
		if p.Rule != "" {
			result = append(result, p.Rule)
		}
	}

	return result
}

// patchTarget extracts the target substring from a patch's Position field.
// "before:OpenAI" → "OpenAI", "after:apple" → "apple"
func patchTarget(p RulePatch) string {
	t := p.Position
	for _, prefix := range []string{"before:", "before ", "after:", "after "} {
		if strings.HasPrefix(t, prefix) {
			return strings.ToLower(strings.TrimSpace(t[len(prefix):]))
		}
	}
	return ""
}

// matchesPatchTarget checks if a rule line matches a patch target.
func matchesPatchTarget(ruleLine, target string) bool {
	if target == "" {
		return false
	}
	lower := strings.ToLower(ruleLine)
	return strings.Contains(lower, target)
}

type noopWriter struct{}

func (w *noopWriter) Write(p []byte) (int, error) { return len(p), nil }

// Validate checks that the template config is valid.
func Validate(cfg *Config) []error {
	var errs []error
	for i, pg := range cfg.ProxyGroups {
		if pg.Name == "" {
			errs = append(errs, fmt.Errorf("proxy_groups[%d]: name is required", i))
		}
		switch pg.Type {
		case "select", "url-test", "fallback", "load-balance", "relay":
			// valid
		default:
			errs = append(errs, fmt.Errorf("proxy_groups[%d]: invalid type %q", i, pg.Type))
		}
		if pg.Type == "url-test" || pg.Type == "fallback" {
			if pg.URL == "" {
				errs = append(errs, fmt.Errorf("proxy_groups[%d]: url required for %s", i, pg.Type))
			}
		}
		if pg.Filter != "" {
			if _, err := regexp.Compile("(?i)" + pg.Filter); err != nil {
				errs = append(errs, fmt.Errorf("proxy_groups[%d]: invalid filter regex: %w", i, err))
			}
		}
	}
	for i, rs := range cfg.RuleSets {
		if rs.Group == "" {
			errs = append(errs, fmt.Errorf("rule_sets[%d]: group is required", i))
		}
		if rs.URL == "" && rs.Rule == "" && rs.BuiltIn == "" {
			errs = append(errs, fmt.Errorf("rule_sets[%d]: one of url/rule/built_in is required", i))
		}
	}
	return errs
}

// --- Built-in templates ---

func basicTemplate() *Config {
	return &Config{
		Template: "basic",
		ProxyGroups: []ProxyGroup{
			{Name: "Proxy", Type: "select", Proxies: []string{"*"}},
			{Name: "Auto", Type: "url-test", Proxies: []string{"*"}, URL: "http://www.gstatic.com/generate_204", Interval: 300},
			{Name: "Fallback", Type: "fallback", Proxies: []string{"*"}, URL: "http://www.gstatic.com/generate_204", Interval: 300},
			{Name: "Final", Type: "select", Proxies: []string{"Proxy", "Auto", "DIRECT"}},
		},
		RuleSets: []RuleSet{
			{Rule: "MATCH,Final"},
		},
	}
}

func acl4ssrFullTemplate() *Config {
	return &Config{
		Template: "acl4ssr_full",
		ProxyGroups: []ProxyGroup{
			{Name: "🚀 节点选择", Type: "select", Proxies: []string{"*"}},
			{Name: "♻️ 自动选择", Type: "url-test", Proxies: []string{"*"}, URL: "http://www.gstatic.com/generate_204", Interval: 300},
			{Name: "📺 Netflix", Type: "select", Filter: "🇭🇰|🇸🇬|NETFLIX|Netflix|nf|NF"},
			{Name: "📽 YouTube", Type: "select", Filter: "YouTube|YT|🇭🇰|🇯🇵|🇸🇬|🇺🇸"},
			{Name: "🌐 谷歌服务", Type: "select", Filter: "Google|谷歌|🇭🇰|🇸🇬|🇺🇸"},
			{Name: "🍎 苹果服务", Type: "select", Filter: "Apple|苹果|🇭🇰|🇸🇬|🇺🇸"},
			{Name: "🐦 Twitter", Type: "select", Filter: "Twitter|推特|🇭🇰|🇯🇵|🇸🇬|🇺🇸"},
			{Name: "📲 Telegram", Type: "select", Filter: "Telegram|电报|🇭🇰|🇸🇬|🇯🇵"},
			{Name: "🎮 游戏平台", Type: "select", Filter: "Steam|Epic|Game|游戏|🇭🇰|🇯🇵|🇸🇬"},
			{Name: "🤖 OpenAI", Type: "select", Filter: "OpenAI|ChatGPT|🇺🇸|🇯🇵"},
			{Name: "🇨🇳 国内流量", Type: "select", Proxies: []string{"DIRECT", "🚀 节点选择"}},
			{Name: "🚫 广告拦截", Type: "select", Proxies: []string{"REJECT", "DIRECT"}},
			{Name: "🐟 漏网之鱼", Type: "select", Proxies: []string{"🚀 节点选择", "♻️ 自动选择", "DIRECT"}},
		},
		RuleSets: []RuleSet{
			{URL: "https://cdn.jsdelivr.net/gh/ACL4SSR/ACL4SSR@master/Clash/BanAD.list", Group: "🚫 广告拦截"},
			{URL: "https://cdn.jsdelivr.net/gh/ACL4SSR/ACL4SSR@master/Clash/BanProgramAD.list", Group: "🚫 广告拦截"},
			{URL: "https://cdn.jsdelivr.net/gh/ACL4SSR/ACL4SSR@master/Clash/BanEasyList.list", Group: "🚫 广告拦截"},
			{URL: "https://cdn.jsdelivr.net/gh/ACL4SSR/ACL4SSR@master/Clash/BanEasyListChina.list", Group: "🚫 广告拦截"},
			{URL: "https://cdn.jsdelivr.net/gh/ACL4SSR/ACL4SSR@master/Clash/Netflix.list", Group: "📺 Netflix"},
			{URL: "https://cdn.jsdelivr.net/gh/ACL4SSR/ACL4SSR@master/Clash/YouTube.list", Group: "📽 YouTube"},
			{URL: "https://cdn.jsdelivr.net/gh/ACL4SSR/ACL4SSR@master/Clash/Google.list", Group: "🌐 谷歌服务"},
			{URL: "https://cdn.jsdelivr.net/gh/ACL4SSR/ACL4SSR@master/Clash/Apple.list", Group: "🍎 苹果服务"},
			{URL: "https://cdn.jsdelivr.net/gh/ACL4SSR/ACL4SSR@master/Clash/Twitter.list", Group: "🐦 Twitter"},
			{URL: "https://cdn.jsdelivr.net/gh/ACL4SSR/ACL4SSR@master/Clash/Telegram.list", Group: "📲 Telegram"},
			{URL: "https://cdn.jsdelivr.net/gh/ACL4SSR/ACL4SSR@master/Clash/Steam.list", Group: "🎮 游戏平台"},
			{URL: "https://cdn.jsdelivr.net/gh/ACL4SSR/ACL4SSR@master/Clash/OpenAi.list", Group: "🤖 OpenAI"},
			{URL: "https://cdn.jsdelivr.net/gh/ACL4SSR/ACL4SSR@master/Clash/Microsoft.list", Group: "🍎 苹果服务"},
			{URL: "https://cdn.jsdelivr.net/gh/ACL4SSR/ACL4SSR@master/Clash/ChinaDomain.list", Group: "🇨🇳 国内流量"},
			{URL: "https://cdn.jsdelivr.net/gh/ACL4SSR/ACL4SSR@master/Clash/ChinaIp.list", Group: "🇨🇳 国内流量"},
			{URL: "https://cdn.jsdelivr.net/gh/ACL4SSR/ACL4SSR@master/Clash/LocalAreaNetwork.list", Group: "🇨🇳 国内流量"},
			{Rule: "MATCH,🐟 漏网之鱼"},
		},
	}
}

func acl4ssrLiteTemplate() *Config {
	return &Config{
		Template: "acl4ssr_lite",
		ProxyGroups: []ProxyGroup{
			{Name: "🚀 节点选择", Type: "select", Proxies: []string{"*"}},
			{Name: "♻️ 自动选择", Type: "url-test", Proxies: []string{"*"}, URL: "http://www.gstatic.com/generate_204", Interval: 300},
			{Name: "🎯 全球直连", Type: "select", Proxies: []string{"DIRECT", "🚀 节点选择"}},
			{Name: "🛑 全球拦截", Type: "select", Proxies: []string{"REJECT", "DIRECT"}},
			{Name: "🐟 漏网之鱼", Type: "select", Proxies: []string{"🚀 节点选择", "♻️ 自动选择", "DIRECT"}},
		},
		RuleSets: []RuleSet{
			{URL: "https://cdn.jsdelivr.net/gh/ACL4SSR/ACL4SSR@master/Clash/BanAD.list", Group: "🛑 全球拦截"},
			{URL: "https://cdn.jsdelivr.net/gh/ACL4SSR/ACL4SSR@master/Clash/BanEasyList.list", Group: "🛑 全球拦截"},
			{URL: "https://cdn.jsdelivr.net/gh/ACL4SSR/ACL4SSR@master/Clash/ProxyGFWlist.list", Group: "🚀 节点选择"},
			{URL: "https://cdn.jsdelivr.net/gh/ACL4SSR/ACL4SSR@master/Clash/ChinaDomain.list", Group: "🎯 全球直连"},
			{URL: "https://cdn.jsdelivr.net/gh/ACL4SSR/ACL4SSR@master/Clash/ChinaIp.list", Group: "🎯 全球直连"},
			{URL: "https://cdn.jsdelivr.net/gh/ACL4SSR/ACL4SSR@master/Clash/LocalAreaNetwork.list", Group: "🎯 全球直连"},
			{Rule: "MATCH,🐟 漏网之鱼"},
		},
	}
}

// loyalsoldierTemplate references Loyalsoldier/clash-rules (10k+ stars).
// Well-maintained, simplified rules optimized for Clash.Meta.
// Uses jsDelivr CDN for reliable access.
func loyalsoldierTemplate() *Config {
	return &Config{
		Template: "loyalsoldier",
		ProxyGroups: []ProxyGroup{
			{Name: "🚀 节点选择", Type: "select", Proxies: []string{"*"}},
			{Name: "♻️ 自动选择", Type: "url-test", Proxies: []string{"*"}, URL: "http://www.gstatic.com/generate_204", Interval: 300},
			{Name: "🎯 全球直连", Type: "select", Proxies: []string{"DIRECT", "🚀 节点选择"}},
			{Name: "🍎 苹果服务", Type: "select", Filter: "Apple|iCloud|🇭🇰|🇸🇬|🇺🇸"},
			{Name: "📲 Telegram", Type: "select", Filter: "Telegram|🇭🇰|🇸🇬|🇯🇵"},
			{Name: "🐟 漏网之鱼", Type: "select", Proxies: []string{"🚀 节点选择", "♻️ 自动选择", "DIRECT"}},
		},
		RuleSets: []RuleSet{
			// Direct connection
			{URL: "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/direct.txt", Group: "🎯 全球直连"},
			{URL: "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/private.txt", Group: "🎯 全球直连"},
			{URL: "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/apple.txt", Group: "🍎 苹果服务"},
			// Proxy
			{URL: "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/proxy.txt", Group: "🚀 节点选择"},
			{URL: "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/gfw.txt", Group: "🚀 节点选择"},
			{URL: "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/telegramcidr.txt", Group: "📲 Telegram"},
			// Reject
			{URL: "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/reject.txt", Group: "🎯 全球直连"},
			{Rule: "MATCH,🐟 漏网之鱼"},
		},
	}
}

// AvailableTemplates returns a list of built-in template names with descriptions.
func AvailableTemplates() []string {
	return []string{"basic", "acl4ssr_full", "acl4ssr_lite", "loyalsoldier"}
}

// HasTemplate returns true if the given name is a valid built-in or cached template.
func HasTemplate(name string) bool {
	for _, t := range AvailableCachedTemplates() {
		if t == name {
			return true
		}
	}
	return false
}

// SanitizeGroupName makes a name safe for use as a group name in YAML.
func SanitizeGroupName(name string) string {
	// Emoji-safe: Clash handles emoji in names fine
	return strings.TrimSpace(name)
}