package rulerepo

import (
	"sort"
	"strings"
)

// SearchOptions controls rule search and filtering.
type SearchOptions struct {
	// Keyword filters rules whose name, tags, description, or ID contain the keyword (case-insensitive).
	Keyword string
	// Repo filters by repository name (e.g., "blackmatrix7", "acl4ssr", "loyalsoldier").
	Repo string
	// Category filters by category (e.g., "AI", "Streaming", "Ads", "Proxy", "Direct").
	Category string
	// Behavior filters by behavior type ("domain", "ipcidr", "classical").
	Behavior string
	// Tag filters by a specific tag.
	Tag string
	// Limit caps the number of results (0 = no limit).
	Limit int
}

// SearchResult holds a matched rule with optional match info.
type SearchResult struct {
	Rule RuleMeta
	// Highlight is a short excerpt showing why the rule matched.
	Highlight string
}

// SearchRules searches the rule index with the given options.
// Results are sorted by relevance (exact name match > tag match > partial match).
func SearchRules(opts SearchOptions) []SearchResult {
	var results []SearchResult
	keyword := strings.ToLower(strings.TrimSpace(opts.Keyword))

	for _, rule := range AllRules() {
		if !matchRepo(rule, opts.Repo) {
			continue
		}
		if !matchCategory(rule, opts.Category) {
			continue
		}
		if !matchBehavior(rule, opts.Behavior) {
			continue
		}
		if !matchTag(rule, opts.Tag) {
			continue
		}

		if keyword != "" {
			highlight := matchKeyword(rule, keyword)
			if highlight == "" {
				continue
			}
			results = append(results, SearchResult{Rule: rule, Highlight: highlight})
		} else {
			results = append(results, SearchResult{Rule: rule, Highlight: ""})
		}
	}

	// Sort by relevance
	sort.Slice(results, func(i, j int) bool {
		ri, rj := results[i].Rule, results[j].Rule
		// Rules with explicit category sort first
		ci, cj := ri.Category, rj.Category
		if ci != cj {
			if ci == "" {
				return false
			}
			if cj == "" {
				return true
			}
		}
		// Then by name
		return ri.Name < rj.Name
	})

	if opts.Limit > 0 && len(results) > opts.Limit {
		results = results[:opts.Limit]
	}

	return results
}

// ListCategories returns all unique categories across all repos.
func ListCategories() []string {
	seen := make(map[string]bool)
	var cats []string
	for _, r := range AllRules() {
		if r.Category != "" && !seen[r.Category] {
			seen[r.Category] = true
			cats = append(cats, r.Category)
		}
	}
	sort.Strings(cats)
	return cats
}

// ListByCategory groups rules by category.
func ListByCategory() map[string][]RuleMeta {
	groups := make(map[string][]RuleMeta)
	for _, r := range AllRules() {
		cat := r.Category
		if cat == "" {
			cat = "Other"
		}
		groups[cat] = append(groups[cat], r)
	}
	// Sort each group by name
	for _, rules := range groups {
		sort.Slice(rules, func(i, j int) bool {
			return rules[i].Name < rules[j].Name
		})
	}
	return groups
}

// --- Matchers ---

func matchRepo(rule RuleMeta, repo string) bool {
	if repo == "" {
		return true
	}
	return strings.EqualFold(string(rule.Repo), repo)
}

func matchCategory(rule RuleMeta, cat string) bool {
	if cat == "" {
		return true
	}
	return strings.EqualFold(rule.Category, cat)
}

func matchBehavior(rule RuleMeta, behavior string) bool {
	if behavior == "" {
		return true
	}
	return strings.EqualFold(rule.Behavior, behavior)
}

func matchTag(rule RuleMeta, tag string) bool {
	if tag == "" {
		return true
	}
	tag = strings.ToLower(tag)
	for _, t := range rule.Tags {
		if strings.EqualFold(t, tag) {
			return true
		}
	}
	return false
}

func matchKeyword(rule RuleMeta, keyword string) string {
	// Exact name match (highest)
	if strings.EqualFold(rule.Name, keyword) {
		return "name match: " + rule.Name
	}
	// ID match
	if strings.Contains(strings.ToLower(rule.ID), keyword) {
		return "id: " + rule.ID
	}
	// Tag match
	for _, t := range rule.Tags {
		if strings.Contains(strings.ToLower(t), keyword) {
			return "tag: " + t
		}
	}
	// Description match
	if strings.Contains(strings.ToLower(rule.Description), keyword) {
		return "desc: " + rule.Description
	}
	// Name prefix/substring
	if strings.Contains(strings.ToLower(rule.Name), keyword) {
		return "name: " + rule.Name
	}
	return ""
}