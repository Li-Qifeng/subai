package rulerepo

import (
	"strings"
	"testing"
)

// --- Core package tests ---

func TestAllRepos(t *testing.T) {
	repos := AllRepos()
	if len(repos) != 3 {
		t.Fatalf("expected 3 repos, got %d", len(repos))
	}

	names := make(map[RepoName]bool)
	for _, r := range repos {
		names[r.Name] = true
	}
	if !names[RepoBlackmatrix7] {
		t.Error("missing blackmatrix7 repo")
	}
	if !names[RepoACL4SSR] {
		t.Error("missing acl4ssr repo")
	}
	if !names[RepoLoyalsoldier] {
		t.Error("missing loyalsoldier repo")
	}
}

func TestAllRules(t *testing.T) {
	rules := AllRules()
	if len(rules) < 50 {
		t.Fatalf("expected at least 50 rules across all repos, got %d", len(rules))
	}

	// Check no duplicate IDs
	seen := make(map[string]bool)
	for _, r := range rules {
		if seen[r.ID] {
			t.Errorf("duplicate rule ID: %s", r.ID)
		}
		seen[r.ID] = true
		if r.URL == "" {
			t.Errorf("rule %s has empty URL", r.ID)
		}
		if r.Behavior == "" {
			t.Errorf("rule %s has empty behavior", r.ID)
		}
	}
}

func TestFindRule(t *testing.T) {
	tests := []struct {
		id   string
		want bool
	}{
		{"blackmatrix7/OpenAI", true},
		{"blackmatrix7/Netflix", true},
		{"acl4ssr/BanAD", true},
		{"loyalsoldier/proxy", true},
		{"blackmatrix7/Nonexistent", false},
		{"", false},
	}

	for _, tt := range tests {
		got := FindRule(tt.id)
		if tt.want && got == nil {
			t.Errorf("FindRule(%q) = nil, want match", tt.id)
		}
		if !tt.want && got != nil {
			t.Errorf("FindRule(%q) = %v, want nil", tt.id, got.Name)
		}
	}
}

func TestFindRule_CaseInsensitive(t *testing.T) {
	variants := []string{
		"blackmatrix7/openai",
		"BLACKMATRIX7/OpenAI",
		"Blackmatrix7/OpenAI",
	}
	for _, v := range variants {
		if got := FindRule(v); got == nil {
			t.Errorf("FindRule(%q) = nil, want match (case-insensitive)", v)
		}
	}
}

func TestListRepos(t *testing.T) {
	repos := ListRepos()
	if len(repos) != 3 {
		t.Fatalf("expected 3 repo descriptions, got %d", len(repos))
	}
	for _, r := range repos {
		if !strings.Contains(r, " — ") {
			t.Errorf("repo description missing ' — ' separator: %q", r)
		}
	}
}

func TestResolveRepoURL(t *testing.T) {
	tests := []struct {
		ref     string
		wantURL string
		wantErr bool
	}{
		{"blackmatrix7:OpenAI", "https://raw.githubusercontent.com/blackmatrix7/ios_rule_script/master/rule/Clash/OpenAI/OpenAI.yaml", false},
		{"blackmatrix7/OpenAI", "https://raw.githubusercontent.com/blackmatrix7/ios_rule_script/master/rule/Clash/OpenAI/OpenAI.yaml", false},
		{"acl4ssr:BanAD", "https://cdn.jsdelivr.net/gh/ACL4SSR/ACL4SSR@master/Clash/BanAD.list", false},
		{"loyalsoldier:proxy", "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/proxy.txt", false},
		{"blackmatrix7/Nonexistent", "", true},
		{"unknown:foo", "", true},
		{"invalid-ref", "", true},
	}

	for _, tt := range tests {
		got, err := ResolveRepoURL(tt.ref)
		if tt.wantErr {
			if err == nil {
				t.Errorf("ResolveRepoURL(%q) = %q, want error", tt.ref, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ResolveRepoURL(%q) unexpected error: %v", tt.ref, err)
			continue
		}
		if got != tt.wantURL {
			t.Errorf("ResolveRepoURL(%q) = %q, want %q", tt.ref, got, tt.wantURL)
		}
	}
}

func TestInferBehavior(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://example.com/ChinaIp.list", "ipcidr"},
		{"https://example.com/telegramcidr.txt", "ipcidr"},
		{"https://example.com/cncidr.txt", "ipcidr"},
		{"https://example.com/ChinaDomain.list", "domain"},
		{"https://example.com/BanAD.list", "classical"},
		{"https://example.com/proxy.txt", "classical"},
		{"https://example.com/mixed.yaml", "classical"},
	}
	for _, tt := range tests {
		if got := InferBehavior(tt.url); got != tt.want {
			t.Errorf("InferBehavior(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
}

// --- Search tests ---

func TestSearchRules_Keyword(t *testing.T) {
	results := SearchRules(SearchOptions{Keyword: "chatgpt"})
	if len(results) == 0 {
		t.Fatal("SearchRules('chatgpt') returned 0 results, expected at least 1")
	}
	found := false
	for _, r := range results {
		if r.Rule.ID == "blackmatrix7/OpenAI" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("SearchRules('chatgpt') should include blackmatrix7/OpenAI, got: %v", resultIDs(results))
	}
}

func TestSearchRules_Repo(t *testing.T) {
	results := SearchRules(SearchOptions{Repo: "loyalsoldier"})
	if len(results) == 0 {
		t.Fatal("SearchRules(repo=loyalsoldier) returned 0 results")
	}
	for _, r := range results {
		if r.Rule.Repo != RepoLoyalsoldier {
			t.Errorf("expected all results from loyalsoldier, got %s/%s", r.Rule.Repo, r.Rule.Name)
		}
	}
}

func TestSearchRules_Category(t *testing.T) {
	results := SearchRules(SearchOptions{Category: "AI"})
	if len(results) == 0 {
		t.Fatal("SearchRules(category=AI) returned 0 results")
	}
	for _, r := range results {
		if r.Rule.Category != "AI" {
			t.Errorf("expected all results category=AI, got %s for %s", r.Rule.Category, r.Rule.ID)
		}
	}
}

func TestSearchRules_Behavior(t *testing.T) {
	results := SearchRules(SearchOptions{Behavior: "ipcidr"})
	if len(results) == 0 {
		t.Fatal("SearchRules(behavior=ipcidr) returned 0 results")
	}
	for _, r := range results {
		if r.Rule.Behavior != "ipcidr" {
			t.Errorf("expected all results behavior=ipcidr, got %s for %s", r.Rule.Behavior, r.Rule.ID)
		}
	}
}

func TestSearchRules_Tag(t *testing.T) {
	results := SearchRules(SearchOptions{Tag: "netflix"})
	if len(results) == 0 {
		t.Fatal("SearchRules(tag=netflix) returned 0 results")
	}
	hasNetflix := false
	for _, r := range results {
		if strings.Contains(r.Rule.Name, "Netflix") || strings.Contains(r.Rule.Name, "netflix") {
			hasNetflix = true
			break
		}
	}
	if !hasNetflix {
		t.Errorf("SearchRules(tag=netflix) should include Netflix rules, got: %v", resultIDs(results))
	}
}

func TestSearchRules_Combined(t *testing.T) {
	results := SearchRules(SearchOptions{Repo: "blackmatrix7", Category: "Streaming", Keyword: "netflix"})
	if len(results) == 0 {
		t.Fatal("combined filter returned 0 results")
	}
	for _, r := range results {
		if r.Rule.Repo != RepoBlackmatrix7 {
			t.Errorf("repo mismatch: %s", r.Rule.Repo)
		}
		if r.Rule.Category != "Streaming" {
			t.Errorf("category mismatch: %s for %s", r.Rule.Category, r.Rule.ID)
		}
	}
}

func TestSearchRules_Limit(t *testing.T) {
	all := SearchRules(SearchOptions{Keyword: "google"})
	limited := SearchRules(SearchOptions{Keyword: "google", Limit: 3})
	if len(limited) > 3 {
		t.Errorf("limited results should be <= 3, got %d", len(limited))
	}
	if len(all) < len(limited) {
		t.Errorf("limited (%d) should be <= unlimited (%d)", len(limited), len(all))
	}
}

func TestSearchRules_NoMatch(t *testing.T) {
	results := SearchRules(SearchOptions{Keyword: "xyznonexistent12345"})
	if len(results) != 0 {
		t.Errorf("expected 0 results for nonsense keyword, got %d", len(results))
	}
}

func TestListCategories(t *testing.T) {
	cats := ListCategories()
	if len(cats) == 0 {
		t.Fatal("ListCategories() returned empty")
	}
	// Verify sorted
	for i := 1; i < len(cats); i++ {
		if cats[i-1] > cats[i] {
			t.Errorf("categories not sorted: %s > %s", cats[i-1], cats[i])
		}
	}
	// Check common categories exist
	expected := []string{"Ads", "AI", "Apple", "Dev", "Direct", "Gaming", "Microsoft", "Proxy", "Social", "Streaming"}
	for _, exp := range expected {
		found := false
		for _, c := range cats {
			if strings.EqualFold(c, exp) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected category %q not found in %v", exp, cats)
		}
	}
}

func TestListByCategory(t *testing.T) {
	groups := ListByCategory()
	if len(groups) == 0 {
		t.Fatal("ListByCategory() returned empty")
	}
	// Each category should have at least 1 rule
	for cat, rules := range groups {
		if len(rules) == 0 {
			t.Errorf("category %q has 0 rules", cat)
		}
		// Rules should be sorted by name
		for i := 1; i < len(rules); i++ {
			if rules[i-1].Name > rules[i].Name {
				t.Errorf("rules in category %q not sorted: %s > %s", cat, rules[i-1].Name, rules[i].Name)
			}
		}
	}
}

// --- Helpers ---

func resultIDs(results []SearchResult) []string {
	var ids []string
	for _, r := range results {
		ids = append(ids, r.Rule.ID)
	}
	return ids
}

// Verify blackmatrix7 has the most rules (it's the largest repo)
func TestBlackmatrix7IsLargest(t *testing.T) {
	repos := AllRepos()
	var counts []int
	for _, r := range repos {
		counts = append(counts, len(r.Rules))
	}
	// blackmatrix7 should be first and largest
	if counts[0] <= counts[1] || counts[0] <= counts[2] {
		t.Errorf("blackmatrix7 should be the largest repo, counts: %v", counts)
	}
}