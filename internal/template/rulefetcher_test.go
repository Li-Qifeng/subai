package template

import (
	"fmt"
	"os"
	"testing"

	"github.com/user/subai/internal/parser"
)

func TestFetchRuleSet(t *testing.T) {
	rules, err := FetchRuleSet("https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/reject.txt")
	if err != nil {
		t.Fatalf("FetchRuleSet failed: %v", err)
	}
	if len(rules) == 0 {
		t.Fatal("FetchRuleSet returned 0 rules")
	}
	fmt.Fprintf(os.Stderr, "  ✅ Fetched %d rules from reject.txt\n", len(rules))
	fmt.Fprintf(os.Stderr, "  First rule: %s\n", rules[0])
}

func TestExpandRuleSets(t *testing.T) {
	ruleSets := []RuleSet{
		{URL: "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/reject.txt", Group: "Reject"},
		{URL: "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/private.txt", Group: "Direct"},
	}
	expanded, errs := ExpandRuleSets(ruleSets)
	for _, e := range errs {
		t.Errorf("ExpandRuleSets error: %v", e)
	}
	if len(expanded) == 0 {
		t.Fatal("ExpandRuleSets returned 0 rules")
	}
	fmt.Fprintf(os.Stderr, "  ✅ Expanded %d rules from %d rule sets\n", len(expanded), len(ruleSets))
	for i := 0; i < 3 && i < len(expanded); i++ {
		fmt.Fprintf(os.Stderr, "  [%d] %s\n", i, expanded[i])
	}
}

func TestFetchRulesBuild(t *testing.T) {
	cfg := loyalsoldierTemplate()
	cfg.FetchRules = true

	proxies := []parser.Proxy{
		{Name: "🇭🇰 HK Node 01", Type: "ss", Server: "1.1.1.1", Port: 443, Cipher: "aes-256-gcm", Password: "test"},
		{Name: "🇯🇵 JP Node 01", Type: "ss", Server: "2.2.2.2", Port: 443, Cipher: "aes-256-gcm", Password: "test"},
	}

	SetLogWriter(os.Stderr)
	result := Build(cfg, proxies)

	if len(result.ProxyGroups) == 0 {
		t.Fatal("Build returned 0 proxy groups")
	}
	if len(result.Rules) == 0 {
		t.Fatal("Build returned 0 rules")
	}
	fmt.Fprintf(os.Stderr, "  ✅ Build: %d proxy groups, %d rules\n", len(result.ProxyGroups), len(result.Rules))
	fmt.Fprintf(os.Stderr, "  Proxy groups:")
	for _, g := range result.ProxyGroups {
		fmt.Fprintf(os.Stderr, " %s", g["name"])
	}
	fmt.Fprintf(os.Stderr, "\n")
}