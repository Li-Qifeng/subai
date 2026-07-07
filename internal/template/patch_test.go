package template

import (
	"reflect"
	"testing"
)

func TestApplyPatches_NoPatches(t *testing.T) {
	rules := []string{"MATCH,Proxy", "GEOIP,CN,DIRECT"}
	result := ApplyPatches(rules, nil)
	if !reflect.DeepEqual(result, rules) {
		t.Errorf("expected no change, got %v", result)
	}
}

func TestApplyPatches_Top(t *testing.T) {
	rules := []string{"MATCH,Proxy"}
	patches := []RulePatch{
		{ID: "p1", Position: "top", Rule: "DOMAIN-SUFFIX,example.com,Proxy"},
	}
	result := ApplyPatches(rules, patches)
	expected := []string{"DOMAIN-SUFFIX,example.com,Proxy", "MATCH,Proxy"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestApplyPatches_Bottom(t *testing.T) {
	rules := []string{"MATCH,Proxy"}
	patches := []RulePatch{
		{ID: "p1", Position: "bottom", Rule: "DOMAIN-SUFFIX,example.com,Direct"},
	}
	result := ApplyPatches(rules, patches)
	expected := []string{"MATCH,Proxy", "DOMAIN-SUFFIX,example.com,Direct"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestApplyPatches_Before(t *testing.T) {
	rules := []string{
		"RULE-SET,proxy,🚀 节点选择",
		"MATCH,🐟 漏网之鱼",
	}
	patches := []RulePatch{
		{ID: "p1", Position: "before:MATCH", Rule: "GEOIP,CN,DIRECT"},
	}
	result := ApplyPatches(rules, patches)
	expected := []string{
		"RULE-SET,proxy,🚀 节点选择",
		"GEOIP,CN,DIRECT",
		"MATCH,🐟 漏网之鱼",
	}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestApplyPatches_After(t *testing.T) {
	rules := []string{
		"RULE-SET,proxy,🚀 节点选择",
		"MATCH,🐟 漏网之鱼",
	}
	patches := []RulePatch{
		{ID: "p1", Position: "after:RULE-SET", Rule: "DOMAIN-SUFFIX,example.com,Proxy"},
	}
	result := ApplyPatches(rules, patches)
	expected := []string{
		"RULE-SET,proxy,🚀 节点选择",
		"DOMAIN-SUFFIX,example.com,Proxy",
		"MATCH,🐟 漏网之鱼",
	}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestApplyPatches_Multiple(t *testing.T) {
	rules := []string{
		"RULE-SET,openai,🤖 AI",
		"RULE-SET,netflix,📺 Netflix",
		"MATCH,🐟 漏网之鱼",
	}
	patches := []RulePatch{
		{ID: "p1", Position: "top", Rule: "DOMAIN-SUFFIX,google.com,Proxy"},
		{ID: "p2", Position: "before:MATCH", Rule: "GEOIP,CN,DIRECT"},
		{ID: "p3", Position: "bottom", Rule: "DOMAIN,test.com,DIRECT"},
	}
	result := ApplyPatches(rules, patches)
	expected := []string{
		"DOMAIN-SUFFIX,google.com,Proxy",
		"RULE-SET,openai,🤖 AI",
		"RULE-SET,netflix,📺 Netflix",
		"GEOIP,CN,DIRECT",
		"MATCH,🐟 漏网之鱼",
		"DOMAIN,test.com,DIRECT",
	}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestApplyPatches_UnmatchedBefore(t *testing.T) {
	rules := []string{"MATCH,Proxy"}
	patches := []RulePatch{
		{ID: "p1", Position: "before:NONEXISTENT", Rule: "DOMAIN,test.com,Proxy"},
	}
	result := ApplyPatches(rules, patches)
	// Unmatched before: should go to bottom
	expected := []string{"MATCH,Proxy", "DOMAIN,test.com,Proxy"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestApplyPatches_EmptyRule(t *testing.T) {
	rules := []string{"MATCH,Proxy"}
	patches := []RulePatch{
		{ID: "p1", Position: "bottom", Rule: ""},
	}
	result := ApplyPatches(rules, patches)
	expected := []string{"MATCH,Proxy"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestApplyPatches_UnknownPosition(t *testing.T) {
	rules := []string{"MATCH,Proxy"}
	patches := []RulePatch{
		{ID: "p1", Position: "invalid", Rule: "DOMAIN,test.com,Proxy"},
	}
	// Should not crash, just log a warning. The patch is silently dropped.
	result := ApplyPatches(rules, patches)
	expected := []string{"MATCH,Proxy"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestPatchTarget(t *testing.T) {
	tests := []struct {
		position string
		expected string
	}{
		{"before:MATCH", "match"},
		{"after:RULE-SET", "rule-set"},
		{"before OpenAI", "openai"},
		{"after netflix", "netflix"},
		{"top", ""},
		{"bottom", ""},
		{"invalid", ""},
	}
	for _, tt := range tests {
		result := patchTarget(RulePatch{Position: tt.position})
		if result != tt.expected {
			t.Errorf("patchTarget(%q) = %q, want %q", tt.position, result, tt.expected)
		}
	}
}

func TestMatchesPatchTarget(t *testing.T) {
	tests := []struct {
		ruleLine string
		target   string
		matches  bool
	}{
		{"RULE-SET,openai,🤖 AI", "openai", true},
		{"MATCH,🐟 漏网之鱼", "match", true},
		{"GEOIP,CN,DIRECT", "match", false},
		{"", "test", false},
		{"RULE-SET,proxy,Proxy", "", false},
	}
	for _, tt := range tests {
		result := matchesPatchTarget(tt.ruleLine, tt.target)
		if result != tt.matches {
			t.Errorf("matchesPatchTarget(%q, %q) = %v, want %v", tt.ruleLine, tt.target, result, tt.matches)
		}
	}
}