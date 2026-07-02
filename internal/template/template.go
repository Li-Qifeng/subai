package template

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/user/subai/internal/parser"
)

// ProxyGroup defines a Clash proxy group.
type ProxyGroup struct {
	Name     string   `yaml:"name"`
	Type     string   `yaml:"type"`
	Proxies  []string `yaml:"proxies,omitempty"` // static proxy list
	Filter   string   `yaml:"filter,omitempty"`  // regex to match proxy names (used at build time)
	URL      string   `yaml:"url,omitempty"`      // health check URL for url-test/fallback
	Interval int      `yaml:"interval,omitempty"`  // health check interval
}

// RuleSet defines a rule that maps traffic to a proxy group.
type RuleSet struct {
	Group    string `yaml:"group"`               // target proxy group name
	URL      string `yaml:"url,omitempty"`        // URL to rule file (RULE-SET)
	Rule     string `yaml:"rule,omitempty"`        // inline rule like "GEOIP,CN,DIRECT" or "MATCH,Proxy"
	BuiltIn  string `yaml:"built_in,omitempty"`   // built-in reference like "geosite:category-ads-all"
}

// Config is the template configuration embedded in subai.yaml.
type Config struct {
	// Built-in template name: "basic", "acl4ssr_full"
	Template string `yaml:"template,omitempty"`

	// Custom proxy groups (used when Template is empty or "custom")
	ProxyGroups []ProxyGroup `yaml:"proxy_groups,omitempty"`
	RuleSets    []RuleSet    `yaml:"rule_sets,omitempty"`
}

// BuildResult holds the generated proxy groups and rules.
type BuildResult struct {
	ProxyGroups []map[string]interface{}
	Rules       []string
}

// Builtin returns a built-in template config by name.
func Builtin(name string) (*Config, error) {
	switch name {
	case "basic":
		return basicTemplate(), nil
	case "acl4ssr_full":
		return acl4ssrFullTemplate(), nil
	case "acl4ssr_lite":
		return acl4ssrLiteTemplate(), nil
	default:
		return nil, fmt.Errorf("unknown built-in template: %q (available: basic, acl4ssr_full, acl4ssr_lite)", name)
	}
}

// Build generates proxy groups and rules from a template config and proxy list.
func Build(cfg *Config, proxies []parser.Proxy) *BuildResult {
	result := &BuildResult{
		ProxyGroups: []map[string]interface{}{},
		Rules:       []string{},
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

	for _, rs := range cfg.RuleSets {
		rule := ""
		if rs.URL != "" {
			rule = fmt.Sprintf("RULE-SET,%s,%s", rs.URL, rs.Group)
		} else if rs.BuiltIn != "" {
			rule = fmt.Sprintf("RULE-SET,%s,%s", rs.BuiltIn, rs.Group)
		} else if rs.Rule != "" {
			rule = rs.Rule
		}
		if rule != "" {
			result.Rules = append(result.Rules, rule)
		}
	}

	return result
}

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
			{URL: "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/BanAD.list", Group: "🚫 广告拦截"},
			{URL: "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/BanProgramAD.list", Group: "🚫 广告拦截"},
			{URL: "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/BanEasyList.list", Group: "🚫 广告拦截"},
			{URL: "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/BanEasyListChina.list", Group: "🚫 广告拦截"},
			{URL: "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/Netflix.list", Group: "📺 Netflix"},
			{URL: "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/YouTube.list", Group: "📽 YouTube"},
			{URL: "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/Google.list", Group: "🌐 谷歌服务"},
			{URL: "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/Apple.list", Group: "🍎 苹果服务"},
			{URL: "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/Twitter.list", Group: "🐦 Twitter"},
			{URL: "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/Telegram.list", Group: "📲 Telegram"},
			{URL: "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/Steam.list", Group: "🎮 游戏平台"},
			{URL: "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/OpenAi.list", Group: "🤖 OpenAI"},
			{URL: "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/Microsoft.list", Group: "🍎 苹果服务"},
			{URL: "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/ChinaDomain.list", Group: "🇨🇳 国内流量"},
			{URL: "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/ChinaIp.list", Group: "🇨🇳 国内流量"},
			{URL: "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/LocalAreaNetwork.list", Group: "🇨🇳 国内流量"},
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
			{URL: "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/BanAD.list", Group: "🛑 全球拦截"},
			{URL: "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/BanEasyList.list", Group: "🛑 全球拦截"},
			{URL: "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/ProxyGFWlist.list", Group: "🚀 节点选择"},
			{URL: "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/ChinaDomain.list", Group: "🎯 全球直连"},
			{URL: "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/ChinaIp.list", Group: "🎯 全球直连"},
			{URL: "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/LocalAreaNetwork.list", Group: "🎯 全球直连"},
			{Rule: "MATCH,🐟 漏网之鱼"},
		},
	}
}

// AvailableTemplates returns a list of built-in template names with descriptions.
func AvailableTemplates() []string {
	return []string{"basic", "acl4ssr_full", "acl4ssr_lite"}
}

// HasTemplate returns true if the given name is a valid built-in template.
func HasTemplate(name string) bool {
	for _, t := range AvailableTemplates() {
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