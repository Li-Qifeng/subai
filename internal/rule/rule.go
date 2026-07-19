package rule

import (
	"fmt"
	"log"
	"regexp"
	"strings"
)

// Action represents a rule action.
type Action string

const (
	ActionInclude Action = "include"
	ActionExclude Action = "exclude"
)

// Rule defines a single filtering rule.
type Rule struct {
	Action  Action `yaml:"action"`
	Pattern string `yaml:"pattern"`
}

// Engine applies rules to a proxy list using regex matching.
type Engine struct {
	rules    []Rule
	compiled []*compiledRule
}

type compiledRule struct {
	action  Action
	pattern *regexp.Regexp
}

// New creates a rule engine from a list of rules.
func New(rules []Rule) *Engine {
	return &Engine{rules: rules}
}

// AddRule adds a rule to the engine (recompiles on Apply/Validate).
func (e *Engine) AddRule(r Rule) {
	e.rules = append(e.rules, r)
	e.compiled = nil
}

// Apply filters proxies based on rules.
// - Include rules: only keep proxies whose Name matches ANY include pattern.
//   If no include rules, don't filter by include.
// - Exclude rules: remove proxies whose Name matches ANY exclude pattern.
//   Exclude always applies.
func (e *Engine) Apply(proxies []Proxy) []Proxy {
	if err := e.compile(); err != nil {
		log.Printf("rule engine: compile failed, returning all proxies unfiltered: %v", err)
		return proxies
	}

	var includePatterns, excludePatterns []*regexp.Regexp
	for _, cr := range e.compiled {
		switch cr.action {
		case ActionInclude:
			includePatterns = append(includePatterns, cr.pattern)
		case ActionExclude:
			excludePatterns = append(excludePatterns, cr.pattern)
		}
	}

	var result []Proxy
	for _, p := range proxies {
		// Check exclude first
		excluded := false
		for _, pat := range excludePatterns {
			if pat.MatchString(p.Name) {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}

		// Check include (if any include rules exist)
		if len(includePatterns) > 0 {
			included := false
			for _, pat := range includePatterns {
				if pat.MatchString(p.Name) {
					included = true
					break
				}
			}
			if !included {
				continue
			}
		}

		result = append(result, p)
	}

	return result
}

// Validate checks that all rule patterns are valid regex.
func (e *Engine) Validate() []error {
	var errs []error
	for i, r := range e.rules {
		if r.Action != ActionInclude && r.Action != ActionExclude {
			errs = append(errs, fmt.Errorf("rule[%d]: invalid action %q (must be include/exclude)", i, r.Action))
		}
		if r.Pattern == "" {
			errs = append(errs, fmt.Errorf("rule[%d]: pattern is required", i))
			continue
		}
		if _, err := regexp.Compile("(?i)" + r.Pattern); err != nil {
			errs = append(errs, fmt.Errorf("rule[%d]: invalid regex %q: %w", i, r.Pattern, err))
		}
	}
	return errs
}

// Rules returns the underlying rule list.
func (e *Engine) Rules() []Rule {
	return e.rules
}

func (e *Engine) compile() error {
	if e.compiled != nil {
		return nil
	}
	for _, r := range e.rules {
		pat, err := regexp.Compile("(?i)" + r.Pattern)
		if err != nil {
			return fmt.Errorf("compile rule %q: %w", r.Pattern, err)
		}
		e.compiled = append(e.compiled, &compiledRule{
			action:  r.Action,
			pattern: pat,
		})
	}
	return nil
}

// Proxy is a minimal interface for rule matching.
// Allows users of this package to pass any type with a Name field.
type Proxy struct {
	Name string
}

// FromFullProxy converts from parser.Proxy to rule.Proxy.
// This function is called by the CLI layer, not imported here.
func FromName(name string) Proxy {
	return Proxy{Name: name}
}

// MatchAny checks if name matches any of the patterns (substring, case-insensitive).
// This is a simpler version used by the server for quick matching.
func MatchAny(name string, patterns []string) bool {
	lower := strings.ToLower(name)
	for _, pat := range patterns {
		if strings.Contains(lower, strings.ToLower(pat)) {
			return true
		}
	}
	return false
}