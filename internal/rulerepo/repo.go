// Package rulerepo provides a built-in index of rule set repositories
// (blackmatrix7, ACL4SSR, Loyalsoldier) for Clash/Mihomo rule-providers.
//
// Rules are identified by unique IDs like "blackmatrix7/OpenAI" and can be
// searched, filtered, and added to the subai config.
package rulerepo

import (
	"fmt"
	"strings"
)

// RepoName identifies a rule source repository.
type RepoName string

const (
	RepoBlackmatrix7 RepoName = "blackmatrix7"
	RepoACL4SSR      RepoName = "acl4ssr"
	RepoLoyalsoldier RepoName = "loyalsoldier"
)

// Repo describes a rule source repository with its metadata and rules.
type Repo struct {
	Name        RepoName   `json:"name"`
	DisplayName string     `json:"display_name"`
	Description string     `json:"description"`
	BaseURL     string     `json:"base_url"`
	Rules       []RuleMeta `json:"rules"`
}

// RuleMeta describes a single rule set available in a repository.
type RuleMeta struct {
	ID          string     `json:"id"`                     // unique ID: "blackmatrix7/OpenAI"
	Name        string     `json:"name"`                   // "OpenAI"
	Repo        RepoName   `json:"repo"`                   // "blackmatrix7"
	URL         string     `json:"url"`                    // full download URL
	Behavior    string     `json:"behavior"`                // "domain", "ipcidr", "classical"
	Category    string     `json:"category,omitempty"`      // "AI", "Streaming", "Ads", etc.
	Tags        []string   `json:"tags,omitempty"`          // searchable keywords
	Description string     `json:"description,omitempty"`
}

// AllRepos returns all registered rule repositories.
func AllRepos() []Repo {
	repos := []Repo{blackmatrix7Repo(), acl4ssrRepo(), loyalsoldierRepo()}
	for i := range repos {
		for j := range repos[i].Rules {
			repos[i].Rules[j].Repo = repos[i].Name
		}
	}
	return repos
}

// AllRules returns a flat list of all rules from all repos.
func AllRules() []RuleMeta {
	var rules []RuleMeta
	for _, repo := range AllRepos() {
		rules = append(rules, repo.Rules...)
	}
	return rules
}

// FindRule looks up a rule by its full ID (e.g., "blackmatrix7/OpenAI").
// Returns nil if not found.
func FindRule(id string) *RuleMeta {
	for _, r := range AllRules() {
		if strings.EqualFold(r.ID, id) {
			return &r
		}
	}
	return nil
}

// ListRepos returns available repository names.
func ListRepos() []string {
	return []string{
		"blackmatrix7  — blackmatrix7/ios_rule_script (600+ rules, most comprehensive)",
		"acl4ssr      — ACL4SSR/ACL4SSR (29 rule files, classic)",
		"loyalsoldier — Loyalsoldier/clash-rules (14 rule files, Clash Premium optimized)",
	}
}

// ResolveRepoURL resolves a repo:name reference to a full rule URL.
// Format: "repo:name" or "blackmatrix7/OpenAI".
func ResolveRepoURL(ref string) (string, error) {
	ref = strings.TrimSpace(ref)

	// Try as full ID first
	if rule := FindRule(ref); rule != nil {
		return rule.URL, nil
	}

	// Try as "repo:name" or "repo/name"
	separator := ":"
	if !strings.Contains(ref, ":") {
		separator = "/"
	}
	parts := strings.SplitN(ref, separator, 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid rule reference %q; use format repo:name or repo/name", ref)
	}

	repoName := strings.ToLower(strings.TrimSpace(parts[0]))
	ruleName := strings.TrimSpace(parts[1])

	for _, repo := range AllRepos() {
		if strings.EqualFold(string(repo.Name), repoName) {
			for _, rule := range repo.Rules {
				if strings.EqualFold(rule.Name, ruleName) {
					return rule.URL, nil
				}
			}
			return "", fmt.Errorf("rule %q not found in repo %s", ruleName, repoName)
		}
	}

	return "", fmt.Errorf("unknown repo %q; available: blackmatrix7, acl4ssr, loyalsoldier", repoName)
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