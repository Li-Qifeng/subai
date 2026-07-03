package template

import (
	"testing"
)

func TestBuiltinWithCache(t *testing.T) {
	names := []string{"basic", "acl4ssr_online", "acl4ssr_online_full_netflix", "special_netease_unblock", "loyalsoldier"}
	for _, name := range names {
		cfg, err := Builtin(name)
		if err != nil {
			t.Errorf("Builtin(%q) unexpected error: %v", name, err)
			continue
		}
		t.Logf("✅ %s: %d groups, %d rules", name, len(cfg.ProxyGroups), len(cfg.RuleSets))
	}
}

func TestBuiltinUnknown(t *testing.T) {
	_, err := Builtin("nonexistent_template_xyz")
	if err == nil {
		t.Error("Builtin('nonexistent') should error")
	}
}

func TestAvailableCachedTemplates(t *testing.T) {
	names := AvailableCachedTemplates()
	if len(names) < 4 {
		t.Errorf("expected at least 4 templates, got %d: %v", len(names), names)
	}
	t.Logf("Total templates: %d", len(names))
	for _, n := range names {
		t.Logf("  - %s", n)
	}
}