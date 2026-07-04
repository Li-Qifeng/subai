package rule

import (
	"testing"
)

func TestNew(t *testing.T) {
	eng := New(nil)
	if eng == nil {
		t.Fatal("New(nil) returned nil")
	}
	if len(eng.rules) != 0 {
		t.Errorf("expected 0 rules, got %d", len(eng.rules))
	}
}

func TestAddRule(t *testing.T) {
	eng := New(nil)
	eng.AddRule(Rule{Action: ActionInclude, Pattern: "HK|Hong"})
	if len(eng.rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(eng.rules))
	}
	// Adding should invalidate cache
	eng.compiled = []*compiledRule{{action: ActionInclude, pattern: nil}}
	eng.AddRule(Rule{Action: ActionExclude, Pattern: "过期"})
	if eng.compiled != nil {
		t.Error("expected compiled cache to be cleared")
	}
}

func TestRules(t *testing.T) {
	eng := New([]Rule{
		{Action: ActionInclude, Pattern: "HK"},
		{Action: ActionExclude, Pattern: "过期"},
	})
	rules := eng.Rules()
	if len(rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(rules))
	}
}

func TestApply_IncludeOnly(t *testing.T) {
	eng := New([]Rule{
		{Action: ActionInclude, Pattern: "HK|Hong"},
	})

	proxies := []Proxy{
		{Name: "🇭🇰 HK Node 01"},
		{Name: "🇯🇵 JP Node 01"},
		{Name: "🇭🇰 Hong Kong 01"},
		{Name: "🇸🇬 SG Node 01"},
	}

	result := eng.Apply(proxies)
	if len(result) != 2 {
		t.Fatalf("expected 2 proxies (HK+Hong), got %d", len(result))
	}
}

func TestApply_ExcludeOnly(t *testing.T) {
	eng := New([]Rule{
		{Action: ActionExclude, Pattern: "过期|剩余"},
	})

	proxies := []Proxy{
		{Name: "🇭🇰 HK Node 01"},
		{Name: "🇯🇵 JP 过期 Node"},
		{Name: "🇸🇬 SG 剩余流量 50%"},
		{Name: "🇺🇸 US Node 01"},
	}

	result := eng.Apply(proxies)
	if len(result) != 2 {
		t.Fatalf("expected 2 proxies (excluded 2), got %d", len(result))
	}
}

func TestApply_IncludeAndExclude(t *testing.T) {
	eng := New([]Rule{
		{Action: ActionInclude, Pattern: "HK|JP|US"},
		{Action: ActionExclude, Pattern: "过期|剩余"},
	})

	proxies := []Proxy{
		{Name: "🇭🇰 HK Node 01"},
		{Name: "🇭🇰 HK 过期 Node"},
		{Name: "🇯🇵 JP Node 01"},
		{Name: "🇯🇵 JP 剩余流量"},
		{Name: "🇸🇬 SG Node 01"},
		{Name: "🇺🇸 US Node 01"},
	}

	result := eng.Apply(proxies)
	if len(result) != 3 {
		t.Fatalf("expected 3 proxies (HK+JP+US minus expired), got %d", len(result))
	}
}

func TestApply_NoIncludeAllPass(t *testing.T) {
	eng := New([]Rule{
		{Action: ActionExclude, Pattern: "过期"},
	})

	proxies := []Proxy{
		{Name: "HK Node"},
		{Name: "JP Node"},
		{Name: "SG Node"},
	}

	result := eng.Apply(proxies)
	if len(result) != 3 {
		t.Fatalf("expected all 3 to pass, got %d", len(result))
	}
}

func TestApply_EmptyProxies(t *testing.T) {
	eng := New([]Rule{
		{Action: ActionInclude, Pattern: "HK"},
	})
	result := eng.Apply(nil)
	if len(result) != 0 {
		t.Errorf("expected 0 results, got %d", len(result))
	}
}

func TestApply_NoRules(t *testing.T) {
	eng := New(nil)
	proxies := []Proxy{
		{Name: "HK Node"},
		{Name: "JP Node"},
	}
	result := eng.Apply(proxies)
	if len(result) != 2 {
		t.Fatalf("expected all 2 proxies, got %d", len(result))
	}
}

func TestApply_CaseInsensitive(t *testing.T) {
	eng := New([]Rule{
		{Action: ActionInclude, Pattern: "hk"},
	})

	proxies := []Proxy{
		{Name: "🇭🇰 HK Node"},
		{Name: "🇯🇵 JP Node"},
	}

	result := eng.Apply(proxies)
	if len(result) != 1 {
		t.Fatalf("expected 1 proxy (case-insensitive match), got %d", len(result))
	}
}

func TestValidate_Valid(t *testing.T) {
	eng := New([]Rule{
		{Action: ActionInclude, Pattern: "HK|Hong"},
		{Action: ActionExclude, Pattern: "过期"},
	})
	errs := eng.Validate()
	if len(errs) != 0 {
		t.Fatalf("expected 0 errors, got %d: %v", len(errs), errs)
	}
}

func TestValidate_InvalidAction(t *testing.T) {
	eng := New([]Rule{
		{Action: "invalid", Pattern: "test"},
	})
	errs := eng.Validate()
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}
}

func TestValidate_EmptyPattern(t *testing.T) {
	eng := New([]Rule{
		{Action: ActionInclude, Pattern: ""},
	})
	errs := eng.Validate()
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}
}

func TestValidate_InvalidRegex(t *testing.T) {
	eng := New([]Rule{
		{Action: ActionInclude, Pattern: "[invalid"},
	})
	errs := eng.Validate()
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}
}

func TestMatchAny(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		patterns []string
		want     bool
	}{
		{"match simple", "🇭🇰 HK Node", []string{"HK", "Hong"}, true},
		{"match exact", "HK Node", []string{"HK"}, true},
		{"no match", "🇯🇵 JP Node", []string{"HK", "Hong"}, false},
		{"case insensitive", "hk node", []string{"HK"}, true},
		{"empty patterns", "HK Node", nil, false},
		{"empty name", "", []string{"HK"}, false},
		{"multi patterns", "SG Node", []string{"HK", "JP", "SG"}, true},
		{"partial match", "Singapore-01", []string{"Singapore"}, true},
		{"chinese match", "香港 01", []string{"香港"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchAny(tt.input, tt.patterns)
			if got != tt.want {
				t.Errorf("MatchAny(%q, %v) = %v, want %v", tt.input, tt.patterns, got, tt.want)
			}
		})
	}
}

func TestFromName(t *testing.T) {
	p := FromName("Test Node")
	if p.Name != "Test Node" {
		t.Errorf("name: got %q", p.Name)
	}
}

func TestApply_RegexWithSpecialChars(t *testing.T) {
	eng := New([]Rule{
		{Action: ActionInclude, Pattern: `\d+`},
	})

	proxies := []Proxy{
		{Name: "Node 01"},
		{Name: "Node A"},
		{Name: "Node 123"},
	}

	result := eng.Apply(proxies)
	if len(result) != 2 {
		t.Fatalf("expected 2 proxies (with digits), got %d", len(result))
	}
}

func TestApply_ExcludeAll(t *testing.T) {
	eng := New([]Rule{
		{Action: ActionExclude, Pattern: ".*"},
	})

	proxies := []Proxy{
		{Name: "Node 01"},
		{Name: "Node 02"},
	}

	result := eng.Apply(proxies)
	if len(result) != 0 {
		t.Fatalf("expected 0 proxies (all excluded), got %d", len(result))
	}
}

func TestApply_IncludeAll(t *testing.T) {
	eng := New([]Rule{
		{Action: ActionInclude, Pattern: ".*"},
	})

	proxies := []Proxy{
		{Name: "Node 01"},
		{Name: "Node 02"},
	}

	result := eng.Apply(proxies)
	if len(result) != 2 {
		t.Fatalf("expected 2 proxies (all included), got %d", len(result))
	}
}

func TestValidate_EmptyRules(t *testing.T) {
	eng := New(nil)
	errs := eng.Validate()
	if len(errs) != 0 {
		t.Fatalf("expected 0 errors for empty rules, got %d", len(errs))
	}
}

func TestApply_InvalidRegex(t *testing.T) {
	// Invalid regex should cause compile to fail, and Apply returns all proxies
	eng := New([]Rule{
		{Action: ActionInclude, Pattern: "[invalid"},
	})

	proxies := []Proxy{
		{Name: "Node 01"},
		{Name: "Node 02"},
	}

	result := eng.Apply(proxies)
	if len(result) != 2 {
		t.Fatalf("expected 2 proxies (all returned on error), got %d", len(result))
	}
}

func TestMultipleExclude(t *testing.T) {
	eng := New([]Rule{
		{Action: ActionExclude, Pattern: "过期"},
		{Action: ActionExclude, Pattern: "剩余"},
		{Action: ActionExclude, Pattern: "测试"},
	})

	proxies := []Proxy{
		{Name: "HK Node"},
		{Name: "HK 过期"},
		{Name: "JP 剩余"},
		{Name: "SG 测试"},
		{Name: "US Node"},
	}

	result := eng.Apply(proxies)
	if len(result) != 2 {
		t.Fatalf("expected 2 proxies (3 excluded), got %d", len(result))
	}
}

func TestMultipleInclude(t *testing.T) {
	eng := New([]Rule{
		{Action: ActionInclude, Pattern: "HK"},
		{Action: ActionInclude, Pattern: "JP"},
		{Action: ActionInclude, Pattern: "US"},
	})

	proxies := []Proxy{
		{Name: "HK Node"},
		{Name: "JP Node"},
		{Name: "SG Node"},
		{Name: "US Node"},
		{Name: "UK Node"},
	}

	result := eng.Apply(proxies)
	if len(result) != 3 {
		t.Fatalf("expected 3 proxies (HK+JP+US), got %d", len(result))
	}
}