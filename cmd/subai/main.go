package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/user/subai/internal/config"
	"github.com/user/subai/internal/converter"
	"github.com/user/subai/internal/fetcher"
	"github.com/user/subai/internal/login"
	"github.com/user/subai/internal/parser"
	"github.com/user/subai/internal/rule"
	"github.com/user/subai/internal/rulerepo"
	"github.com/user/subai/internal/server"
	"github.com/user/subai/internal/template"
)

var (
	cfgFile    string
	outputFile string
	target     string
	verbose    bool
)

var rootCmd = &cobra.Command{
	Use:   "subai",
	Short: "AI-managed subscription converter",
	Long: `subai is a high-performance, lightweight subscription conversion tool designed for AI management.
It converts between various proxy subscription formats with deterministic rule-based filtering.

CLI commands for AI control:
  subai convert       Convert subscriptions (from config or inline)
  subai validate      Validate configuration and rules
  subai source        Manage subscription sources
  subai serve         Start HTTP server for client subscriptions
  subai version       Show version`,
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "subai.yaml", "Config file path")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")

	// Convert command
	convertCmd := &cobra.Command{
		Use:   "convert [url...]",
		Short: "Convert subscriptions to target format",
		Long: `Convert one or more subscription URLs to the specified format.
If no URLs are provided, sources from config file are used.
Output is written to stdout unless --output is specified.

Examples:
  subai convert -t clash "https://example.com/sub?token=xxx"
  subai convert -t base64 -o out.txt "ss://..." "vmess://..."
  subai convert -c subai.yaml`,
		RunE: runConvert,
	}
	convertCmd.Flags().StringVarP(&target, "target", "t", "clash", "Output format (clash, base64, mixed)")
	convertCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file (default: stdout)")
	convertCmd.Flags().String("profile", "", "Use named profile (overrides current_profile)")
	rootCmd.AddCommand(convertCmd)

	// Validate command
	validateCmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration and rules",
		Long:  `Check the config file for errors and validate all rule patterns.`,
		RunE:  runValidate,
	}
	validateCmd.Flags().String("profile", "", "Use named profile (overrides current_profile)")
	rootCmd.AddCommand(validateCmd)

	// Source management commands
	rootCmd.AddCommand(newSourceCmd())

	// Serve command
	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Start HTTP server for client subscriptions",
		Long:  `Start a lightweight HTTP server that serves converted subscriptions to clients.`,
		RunE:  runServe,
	}
	serveCmd.Flags().String("listen", ":8080", "Listen address")
	serveCmd.Flags().String("token", "", "Auth token for API access")
	rootCmd.AddCommand(serveCmd)

	// Login command
	loginCmd := &cobra.Command{
		Use:   "login <name>",
		Short: "Auto-login to panel and save subscribe URL",
		Long: `Automatically log in to a proxy panel (V2Board) to obtain the
subscription URL. Requires Python with cloudscraper installed.

  pip3 install cloudscraper

The login result is saved as a new source in the config file.

Examples:
  subai login my-airport --url https://www.xfltd.org \
    --email user@example.com --password "xxx"
  subai login my-airport -c subai.yaml \
    --method v2board --url https://panel.com --email u@e.com --password "p"
`,
		Args: cobra.ExactArgs(1),
		RunE: runLogin,
	}
	loginCmd.Flags().String("method", "v2board", "Login method (v2board)")
	loginCmd.Flags().String("url", "", "Panel base URL (e.g. https://www.xfltd.org)")
	loginCmd.Flags().String("email", "", "Login email")
	loginCmd.Flags().String("password", "", "Login password")
	rootCmd.AddCommand(loginCmd)

	// Template command
	templateCmd := &cobra.Command{
		Use:   "template",
		Short: "List, sync, and manage templates",
		Long: `Manage templates for subscription conversion.

Built-in templates (always available):
  basic, acl4ssr_full, acl4ssr_lite, loyalsoldier

Remote templates (synced from GitHub, 20+ variants):
  Run 'subai template sync' to fetch the latest templates.
  Run 'subai template list' to see all available templates.

Templates are cached locally and auto-refresh when stale.`,
	}

	templateCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all available templates (built-in + cached remote)",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Built-in templates
			fmt.Println("━━ Built-in templates ━━")
			for _, name := range template.AvailableTemplates() {
				desc := ""
				switch name {
				case "basic":
					desc = "Simple select/url-test/fallback groups"
				case "acl4ssr_full":
					desc = "Full ACL4SSR rules (13 proxy groups, 16 rule sets)"
				case "acl4ssr_lite":
					desc = "Lite ACL4SSR rules (5 groups, 7 rule sets)"
				case "loyalsoldier":
					desc = "Loyalsoldier/clash-rules (6 groups, 8 rule sets)"
				}
				fmt.Printf("  %-32s %s\n", name, desc)
			}

			// Cached remote templates
			cached, err := template.ListCachedTemplates()
			if err != nil || cached == nil || len(cached) == 0 {
				fmt.Println()
				fmt.Println("No remote templates cached. Run 'subai template sync' to fetch.")
				return nil
			}

			// Group by category
			byCat := make(map[string][]template.TemplateIndexEntry)
			for _, entry := range cached {
				if entry.Category == "" {
					entry.Category = "other"
				}
				byCat[entry.Category] = append(byCat[entry.Category], entry)
			}

			fmt.Println()
			fmt.Println("━━ Remote templates (synced) ━━")
			for cat, entries := range byCat {
				fmt.Printf("  [%s]\n", cat)
				for _, e := range entries {
					fmt.Printf("  %-32s %s\n", e.Name, e.Description)
				}
				fmt.Println()
			}

			fmt.Println("Use 'subai convert -t clash' or set 'template: <name>' in config.")
			return nil
		},
	})

	templateCmd.AddCommand(&cobra.Command{
		Use:   "sync [url]",
		Short: "Sync latest templates from remote repository",
		Long: `Fetch the latest template list and files from the remote template repository.

By default, templates are fetched from:
  https://raw.githubusercontent.com/Li-Qifeng/subai/main/templates

A custom URL can be provided as an argument.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			remoteURL := template.DefaultRemoteURL
			if len(args) > 0 {
				remoteURL = args[0]
			}
			template.SetLogWriter(os.Stderr)
			if err := template.SyncTemplates(remoteURL); err != nil {
				return fmt.Errorf("sync failed: %w", err)
			}
			return nil
		},
	})
	rootCmd.AddCommand(templateCmd)

	// Rule management commands
	rootCmd.AddCommand(newRuleCmd())

	// Profile management commands
	rootCmd.AddCommand(newProfileCmd())

	// Dry-run convert
	dryRunCmd := &cobra.Command{
		Use:   "dry-run",
		Short: "Preview conversion result without writing output",
		Long:  `Similar to convert but prints a summary instead of full output. Useful for AI to verify changes.`,
		RunE:  runDryRun,
	}
	dryRunCmd.Flags().StringVarP(&target, "target", "t", "clash", "Output format")
	rootCmd.AddCommand(dryRunCmd)

	// Version command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Show version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("subai v0.1.0")
		},
	})
}

func runConvert(cmd *cobra.Command, args []string) error {
	var proxies parser.ProxyList
	// Preload config if file exists (for template settings even in inline mode)
	var cfg *config.Config
	if _, err := os.Stat(cfgFile); err == nil {
		cfg, _ = config.Load(cfgFile)
	}

	if len(args) > 0 {
		// Inline mode: detect proxy URIs vs subscription URLs
		for _, arg := range args {
			if isProxyURI(arg) {
				// Direct proxy URI
				p, err := parser.ParseURI(arg)
				if err != nil {
					log.Printf("parse proxy %q: %v", arg, err)
					continue
				}
				proxies = append(proxies, *p)
			} else {
				// Subscription URL - fetch first
				body, err := fetcher.Fetch(arg, "", "")
				if err != nil {
					return fmt.Errorf("fetch %s: %w", arg, err)
				}
				parsed, err := parser.ParseAuto(body)
				if err != nil {
					return fmt.Errorf("parse %s: %w", arg, err)
				}
				proxies = append(proxies, parsed...)
			}
		}
	} else {
		// Config mode
		var err error
		cfg, err = config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("load config %s: %w", cfgFile, err)
		}

		// Resolve profile
		profileName, _ := cmd.Flags().GetString("profile")
		cfg = cfg.Resolve(profileName)

		for _, src := range cfg.Sources {
			body, err := fetcher.Fetch(src.URL, src.Cookie, src.UserAgent)
			if err != nil {
				log.Printf("fetch source %q: %v", src.Name, err)
				continue
			}
			parsed, err := parser.ParseAuto(body)
			if err != nil {
				log.Printf("parse source %q: %v", src.Name, err)
				continue
			}
			proxies = append(proxies, parsed...)
		}

		// Apply rules
		if len(cfg.Rules.Include) > 0 || len(cfg.Rules.Exclude) > 0 {
			var ruleProxies []rule.Proxy
			for _, p := range proxies {
				ruleProxies = append(ruleProxies, rule.FromName(p.Name))
			}
			var rules []rule.Rule
			for _, inc := range cfg.Rules.Include {
				rules = append(rules, rule.Rule{Action: rule.ActionInclude, Pattern: inc})
			}
			for _, exc := range cfg.Rules.Exclude {
				rules = append(rules, rule.Rule{Action: rule.ActionExclude, Pattern: exc})
			}
			engine := rule.New(rules)
			filtered := engine.Apply(ruleProxies)

			// Map back
			nameSet := make(map[string]bool)
			for _, rp := range filtered {
				nameSet[rp.Name] = true
			}
			var filteredProxies parser.ProxyList
			for _, p := range proxies {
				if nameSet[p.Name] {
					filteredProxies = append(filteredProxies, p)
				}
			}
			proxies = filteredProxies
		}
	}

	if len(proxies) == 0 {
		return fmt.Errorf("no proxies found")
	}

	var eng *converter.Engine
	if cfg != nil && (cfg.Output.Template.Template != "" || len(cfg.Output.Template.ProxyGroups) > 0) {
		eng = converter.NewWithTemplate(&cfg.Output.Template)
	} else {
		eng = converter.New()
	}
	data, err := eng.Convert(proxies, target)
	if err != nil {
		return fmt.Errorf("convert: %w", err)
	}

	if outputFile != "" {
		if err := os.WriteFile(outputFile, data, 0644); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Written %d bytes to %s (%d proxies)\n", len(data), outputFile, len(proxies))
	} else {
		os.Stdout.Write(data)
	}

	return nil
}

func runValidate(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("config error: %w", err)
	}

	profileName, _ := cmd.Flags().GetString("profile")
	cfg = cfg.Resolve(profileName)

	errs := cfg.Validate()
	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "  ❌ %v\n", e)
		}
		return fmt.Errorf("%d config error(s)", len(errs))
	}
	fmt.Fprintf(os.Stderr, "  ✅ Config structure valid\n")

	// Validate rules
	var rules []rule.Rule
	for _, inc := range cfg.Rules.Include {
		rules = append(rules, rule.Rule{Action: rule.ActionInclude, Pattern: inc})
	}
	for _, exc := range cfg.Rules.Exclude {
		rules = append(rules, rule.Rule{Action: rule.ActionExclude, Pattern: exc})
	}

	if len(rules) > 0 {
		eng := rule.New(rules)
		if ruleErrs := eng.Validate(); len(ruleErrs) > 0 {
			for _, e := range ruleErrs {
				fmt.Fprintf(os.Stderr, "  ❌ Rule error: %v\n", e)
			}
			return fmt.Errorf("%d rule error(s)", len(ruleErrs))
		}
		fmt.Fprintf(os.Stderr, "  ✅ Rules valid (%d rules)\n", len(rules))
	}

	fmt.Fprintf(os.Stderr, "  ✅ Sources: %d configured\n", len(cfg.Sources))
	return nil
}

func runDryRun(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	var totalProxies int
	for _, src := range cfg.Sources {
		body, err := fetcher.Fetch(src.URL, src.Cookie, src.UserAgent)
		if err != nil {
			log.Printf("fetch %q: %v", src.Name, err)
			continue
		}
		proxies, err := parser.ParseAuto(body)
		if err != nil {
			log.Printf("parse %q: %v", src.Name, err)
			continue
		}
		totalProxies += len(proxies)
		fmt.Fprintf(os.Stderr, "  📦 %s: %d proxies\n", src.Name, len(proxies))
	}

	out := map[string]interface{}{
		"sources":       len(cfg.Sources),
		"total_proxies": totalProxies,
		"target":        target,
		"output_file":   outputFile,
	}

	if len(cfg.Rules.Include) > 0 {
		out["include_rules"] = cfg.Rules.Include
	}
	if len(cfg.Rules.Exclude) > 0 {
		out["exclude_rules"] = cfg.Rules.Exclude
	}

	data, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(data))
	return nil
}

func runServe(cmd *cobra.Command, args []string) error {
	listen, _ := cmd.Flags().GetString("listen")
	token, _ := cmd.Flags().GetString("token")

	absPath, err := filepath.Abs(cfgFile)
	if err != nil {
		return fmt.Errorf("config path: %w", err)
	}

	srv := server.New(listen, token, absPath)
	return srv.Start()
}

func runLogin(cmd *cobra.Command, args []string) error {
	name := args[0]
	method, _ := cmd.Flags().GetString("method")
	baseURL, _ := cmd.Flags().GetString("url")
	email, _ := cmd.Flags().GetString("email")
	password, _ := cmd.Flags().GetString("password")

	if baseURL == "" {
		return fmt.Errorf("--url is required (panel base URL)")
	}
	if email == "" {
		return fmt.Errorf("--email is required")
	}
	if password == "" {
		return fmt.Errorf("--password is required")
	}

	fmt.Fprintf(os.Stderr, "  🔐 Logging into %s as %s...\n", baseURL, email)

	var result *login.Result
	var err error
	switch method {
	case "v2board":
		result, err = login.V2Board(baseURL, email, password)
	default:
		return fmt.Errorf("unsupported login method: %s (supported: v2board)", method)
	}
	if err != nil {
		return fmt.Errorf("login failed: %w\n\n  Tip: Try running 'subai login' from a machine with a clean IP\n  Or manually add the subscribe URL: subai source add %s <url>", err, name)
	}

	fmt.Fprintf(os.Stderr, "  ✅ Login successful!\n")
	fmt.Fprintf(os.Stderr, "  📧 Email: %s\n", result.User.Email)
	if result.User.Plan != "" {
		fmt.Fprintf(os.Stderr, "  📦 Plan: %s\n", result.User.Plan)
	}
	if result.User.TransferEnable > 0 {
		used := result.User.Used
		total := result.User.TransferEnable
		pct := float64(used) / float64(total) * 100
		fmt.Fprintf(os.Stderr, "  📊 Traffic: %.2f GB / %.2f GB (%.1f%%)\n",
			float64(used)/1073741824, float64(total)/1073741824, pct)
	}
	fmt.Fprintf(os.Stderr, "  🔗 Subscribe URL: %s\n", result.SubscribeURL)

	// Save to config
	cfg, err := config.Load(cfgFile)
	if err != nil {
		cfg = config.DefaultConfig()
	}

	// Update or add source
	found := false
	for i := range cfg.Sources {
		if cfg.Sources[i].Name == name {
			cfg.Sources[i].URL = result.SubscribeURL
			cfg.Sources[i].UserAgent = "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15"
			cfg.Sources[i].Login = &config.Login{
				Method:   method,
				URL:      baseURL,
				Email:    email,
				Password: password,
			}
			found = true
			fmt.Fprintf(os.Stderr, "  📝 Updated source %q in config\n", name)
			break
		}
	}
	if !found {
		cfg.Sources = append(cfg.Sources, config.Source{
			Name:      name,
			URL:       result.SubscribeURL,
			UserAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15",
			Login: &config.Login{
				Method:   method,
				URL:      baseURL,
				Email:    email,
				Password: password,
			},
		})
		fmt.Fprintf(os.Stderr, "  📝 Added source %q to config\n", name)
	}

	if err := cfg.Save(cfgFile); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Fprintf(os.Stderr, "  ✅ Config saved to %s\n", cfgFile)
	return nil
}

func newSourceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "source",
		Short: "Manage subscription sources",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List configured sources",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			if len(cfg.Sources) == 0 {
				fmt.Println("No sources configured.")
				return nil
			}
			for i, src := range cfg.Sources {
				fmt.Printf("%d. %s\n", i+1, src.Name)
				fmt.Printf("   URL: %s\n", src.URL)
				if src.Cookie != "" {
					fmt.Printf("   Cookie: %s\n", maskString(src.Cookie, 20))
				}
				if src.UserAgent != "" {
					fmt.Printf("   UA: %s\n", src.UserAgent)
				}
				if src.Login != nil {
					fmt.Printf("   Login: %s @ %s (method: %s)\n",
						maskString(src.Login.Email, 16), src.Login.URL, src.Login.Method)
				}
				fmt.Println()
			}
			return nil
		},
	})

	addCmd := &cobra.Command{
		Use:   "add <name> <url>",
		Short: "Add a subscription source",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgFile)
			if err != nil {
				cfg = config.DefaultConfig()
			}
			cookie, _ := cmd.Flags().GetString("cookie")
			ua, _ := cmd.Flags().GetString("user-agent")
			cfg.Sources = append(cfg.Sources, config.Source{
				Name:      args[0],
				URL:       args[1],
				Cookie:    cookie,
				UserAgent: ua,
			})
			if err := cfg.Save(cfgFile); err != nil {
				return fmt.Errorf("save config: %w", err)
			}
			fmt.Printf("Added source %q\n", args[0])
			return nil
		},
	}
	addCmd.Flags().String("cookie", "", "Cookie for authentication")
	addCmd.Flags().String("user-agent", "", "User-Agent header")
	cmd.AddCommand(addCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a subscription source",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			found := false
			var updated []config.Source
			for _, src := range cfg.Sources {
				if src.Name == args[0] {
					found = true
					continue
				}
				updated = append(updated, src)
			}
			if !found {
				return fmt.Errorf("source %q not found", args[0])
			}
			cfg.Sources = updated
			if err := cfg.Save(cfgFile); err != nil {
				return fmt.Errorf("save config: %w", err)
			}
			fmt.Printf("Removed source %q\n", args[0])
			return nil
		},
	})

	return cmd
}

func maskString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s[:len(s)/3] + "***" + s[len(s)-len(s)/3:]
	}
	return s[:maxLen/3] + "..." + s[len(s)-maxLen/3:]
}

// isProxyURI checks if a string looks like a proxy URI rather than an HTTP URL.
func isProxyURI(s string) bool {
	prefixes := []string{
		"ss://", "ssr://", "vmess://", "vless://",
		"trojan://", "hysteria2://", "hy2://", "tuic://",
		"socks5://", "ssd://",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

// newRuleCmd creates the "subai rule" command tree for rule management.
func newRuleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rule",
		Short: "Manage rule sets from built-in rule repositories",
		Long: `Manage rule sets from built-in rule repositories.

Available rule sources:
  blackmatrix7  — blackmatrix7/ios_rule_script (600+ rules, most comprehensive)
  acl4ssr       — ACL4SSR/ACL4SSR (29 rule files, classic)
  loyalsoldier  — Loyalsoldier/clash-rules (14 rule files, Clash Premium optimized)

Use 'subai rule list' to see all available rules.
Use 'subai rule add <id> --group <group>' to add a rule to your config.`,
	}

	// List command
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all available rule sets from built-in repositories",
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, _ := cmd.Flags().GetString("repo")
			category, _ := cmd.Flags().GetString("category")
			behavior, _ := cmd.Flags().GetString("behavior")

			opts := rulerepo.SearchOptions{
				Repo:     repo,
				Category: category,
				Behavior: behavior,
			}

			results := rulerepo.SearchRules(opts)
			if len(results) == 0 {
				fmt.Println("No rules found matching the filters.")
				return nil
			}

			// Group by repo + category
			type groupKey struct{ repo, cat string }
			groups := make(map[groupKey][]rulerepo.RuleMeta)
			var keys []groupKey
			for _, r := range results {
				k := groupKey{repo: string(r.Rule.Repo), cat: r.Rule.Category}
				if _, ok := groups[k]; !ok {
					keys = append(keys, k)
				}
				groups[k] = append(groups[k], r.Rule)
			}

			for _, k := range keys {
				cat := k.cat
				if cat == "" {
					cat = "Other"
				}
				fmt.Printf("\n  \033[1m%s\033[0m  [\033[36m%s\033[0m]\n", k.repo, cat)
				for _, r := range groups[k] {
					fmt.Printf("    \033[33m%-36s\033[0m %s\n", r.ID, r.Description)
				}
			}
			fmt.Printf("\n  Total: %d rules\n", len(results))
			return nil
		},
	}
	listCmd.Flags().String("repo", "", "Filter by repo (blackmatrix7, acl4ssr, loyalsoldier)")
	listCmd.Flags().String("category", "", "Filter by category (AI, Streaming, Ads, Proxy, Direct, ...)")
	listCmd.Flags().String("behavior", "", "Filter by behavior (domain, ipcidr, classical)")
	cmd.AddCommand(listCmd)

	// Search command
	searchCmd := &cobra.Command{
		Use:   "search <keyword>",
		Short: "Search rule sets by keyword",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			results := rulerepo.SearchRules(rulerepo.SearchOptions{Keyword: args[0]})
			if len(results) == 0 {
				fmt.Printf("No rules found matching %q\n", args[0])
				return nil
			}
			fmt.Printf("  Found %d rules matching %q:\n\n", len(results), args[0])
			for _, r := range results {
				hl := ""
				if r.Highlight != "" {
					hl = " \033[90m(" + r.Highlight + ")\033[0m"
				}
				cat := r.Rule.Category
				if cat == "" {
					cat = "Other"
				}
				fmt.Printf("  \033[33m%-36s\033[0m [\033[36m%s\033[0m] [\033[35m%s\033[0m] %s%s\n",
					r.Rule.ID, cat, r.Rule.Behavior, r.Rule.Description, hl)
			}
			return nil
		},
	}
	cmd.AddCommand(searchCmd)

	// Add command
	addCmd := &cobra.Command{
		Use:   "add <id>",
		Short: "Add a rule set to the config",
		Long: `Add a rule set from a built-in repository to the config file.

The <id> is the rule identifier, e.g. "blackmatrix7/OpenAI" or "loyalsoldier/proxy".
Use --group to specify the target proxy group (default: "🚀 节点选择").
Use --provider-name to override the auto-generated rule-provider name.

Examples:
  subai rule add blackmatrix7/OpenAI --group "🤖 AI"
  subai rule add loyalsoldier/proxy --group "🚀 节点选择"
  subai rule add acl4ssr/BanAD --group "🚫 广告拦截"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			group, _ := cmd.Flags().GetString("group")
			providerName, _ := cmd.Flags().GetString("provider-name")

			// Resolve the rule
			rule := rulerepo.FindRule(id)
			if rule == nil {
				// Try as repo:name format
				url, err := rulerepo.ResolveRepoURL(id)
				if err != nil {
					return fmt.Errorf("unknown rule %q; use 'subai rule search' to find rules", id)
				}
				// Create a minimal rule meta for the found URL
				rule = &rulerepo.RuleMeta{ID: id, Name: id, URL: url, Behavior: rulerepo.InferBehavior(url)}
			}

			// Load config
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			// Check for duplicates
			for _, rs := range cfg.Output.Template.RuleSets {
				if rs.URL == rule.URL {
					return fmt.Errorf("rule %q already exists in config (URL: %s)", id, rule.URL)
				}
			}

			// Build the rule set entry
			rs := template.RuleSet{
				URL:          rule.URL,
				Group:        group,
				ProviderName: providerName,
			}
			cfg.Output.Template.RuleSets = append(cfg.Output.Template.RuleSets, rs)

			// Save
			if err := cfg.Save(cfgFile); err != nil {
				return fmt.Errorf("save config: %w", err)
			}

			fmt.Printf("  ✅ Added rule \033[33m%s\033[0m → group \033[36m%s\033[0m\n", id, group)
			fmt.Printf("     URL: %s\n", rule.URL)
			fmt.Printf("     Behavior: %s\n", rule.Behavior)
			return nil
		},
	}
	addCmd.Flags().String("group", "🚀 节点选择", "Target proxy group name")
	addCmd.Flags().String("provider-name", "", "Override rule-provider name (auto-generated from URL if empty)")
	cmd.AddCommand(addCmd)

	// Remove command
	removeCmd := &cobra.Command{
		Use:   "remove <id>",
		Short: "Remove a rule set from the config",
		Long: `Remove a rule set from the config file by its ID or URL.

The <id> can be a rule identifier (e.g. "blackmatrix7/OpenAI") or
a substring of the rule URL. Use 'subai rule list' to see configured rules.

Examples:
  subai rule remove blackmatrix7/OpenAI
  subai rule remove OpenAI        (matches by name/substring)
  subai rule remove "BanAD"       (matches by URL substring)`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.ToLower(args[0])

			// Load config
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			// Find matching rules
			var matched []int
			for i, rs := range cfg.Output.Template.RuleSets {
				lowerURL := strings.ToLower(rs.URL)
				lowerName := strings.ToLower(rs.ProviderName)
				if strings.Contains(lowerURL, query) || strings.Contains(lowerName, query) {
					matched = append(matched, i)
				}
			}

			if len(matched) == 0 {
				return fmt.Errorf("no rule sets matching %q found in config", args[0])
			}

			// Remove in reverse order
			removed := make([]string, 0, len(matched))
			for i := len(matched) - 1; i >= 0; i-- {
				idx := matched[i]
				removed = append(removed, cfg.Output.Template.RuleSets[idx].URL)
				cfg.Output.Template.RuleSets = append(
					cfg.Output.Template.RuleSets[:idx],
					cfg.Output.Template.RuleSets[idx+1:]...,
				)
			}

			if err := cfg.Save(cfgFile); err != nil {
				return fmt.Errorf("save config: %w", err)
			}

			fmt.Printf("  ✅ Removed %d rule(s):\n", len(removed))
			for _, u := range removed {
				fmt.Printf("     - %s\n", u)
			}
			return nil
		},
	}
	cmd.AddCommand(removeCmd)

	// Patch command
	cmd.AddCommand(newRulePatchCmd())

	// Order command
	cmd.AddCommand(newRuleOrderCmd())

	return cmd
}

// newRulePatchCmd creates the "subai rule patch" subcommand.
func newRulePatchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "patch",
		Short: "Manage rule patches (inline rule insertions)",
		Long: `Manage rule patches — custom inline rules inserted at positions in the rules list.

Positions:
  top             — insert at the beginning
  bottom          — insert at the end
  before:<target> — insert before the rule matching <target> (URL/name substring)
  after:<target>  — insert after the rule matching <target>

Examples:
  subai rule patch add "DOMAIN-SUFFIX,example.com,Proxy" --position top --id my-patch
  subai rule patch add "GEOIP,CN,DIRECT" --position before:MATCH --id geoip-cn
  subai rule patch list
  subai rule patch remove my-patch
  subai rule patch clear`,
	}

	addCmd := &cobra.Command{
		Use:   "add <rule>",
		Short: "Add a rule patch at a position",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ruleText := args[0]
			patchID, _ := cmd.Flags().GetString("id")
			position, _ := cmd.Flags().GetString("position")

			if patchID == "" {
				return fmt.Errorf("--id is required")
			}
			if position == "" {
				return fmt.Errorf("--position is required (top/bottom/before:X/after:X)")
			}

			cfg, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			// Check duplicate ID
			for _, p := range cfg.Output.Template.RulePatches {
				if p.ID == patchID {
					return fmt.Errorf("patch %q already exists", patchID)
				}
			}

			cfg.Output.Template.RulePatches = append(cfg.Output.Template.RulePatches, template.RulePatch{
				ID:       patchID,
				Position: position,
				Rule:     ruleText,
			})

			if err := cfg.Save(cfgFile); err != nil {
				return fmt.Errorf("save config: %w", err)
			}
			fmt.Printf("  ✅ Added patch \033[33m%s\033[0m at position \033[36m%s\033[0m\n", patchID, position)
			return nil
		},
	}
	addCmd.Flags().String("id", "", "Unique patch identifier (required)")
	addCmd.Flags().String("position", "", "Insertion position: top/bottom/before:X/after:X (required)")
	cmd.AddCommand(addCmd)

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List configured rule patches",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			patches := cfg.Output.Template.RulePatches
			if len(patches) == 0 {
				fmt.Println("No rule patches configured.")
				return nil
			}

			fmt.Printf("  %d rule patch(es):\n", len(patches))
			for _, p := range patches {
				fmt.Printf("    \033[33m%-24s\033[0m [\033[36m%s\033[0m] %s\n", p.ID, p.Position, p.Rule)
			}
			return nil
		},
	}
	cmd.AddCommand(listCmd)

	removeCmd := &cobra.Command{
		Use:   "remove <id>",
		Short: "Remove a rule patch by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			found := false
			var updated []template.RulePatch
			for _, p := range cfg.Output.Template.RulePatches {
				if p.ID == id {
					found = true
					continue
				}
				updated = append(updated, p)
			}
			if !found {
				return fmt.Errorf("patch %q not found", id)
			}

			cfg.Output.Template.RulePatches = updated
			if err := cfg.Save(cfgFile); err != nil {
				return fmt.Errorf("save config: %w", err)
			}
			fmt.Printf("  ✅ Removed patch \033[33m%s\033[0m\n", id)
			return nil
		},
	}
	cmd.AddCommand(removeCmd)

	clearCmd := &cobra.Command{
		Use:   "clear",
		Short: "Remove all rule patches",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			count := len(cfg.Output.Template.RulePatches)
			if count == 0 {
				fmt.Println("No patches to clear.")
				return nil
			}

			cfg.Output.Template.RulePatches = nil
			if err := cfg.Save(cfgFile); err != nil {
				return fmt.Errorf("save config: %w", err)
			}
			fmt.Printf("  ✅ Cleared %d patch(es)\n", count)
			return nil
		},
	}
	cmd.AddCommand(clearCmd)

	return cmd
}

// newRuleOrderCmd creates the "subai rule order" subcommand.
func newRuleOrderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "order <id>",
		Short: "Reorder rule sets in config",
		Long: `Reorder rule sets by moving them up, down, or to a specific position.

The <id> can be a rule identifier, URL substring, or provider name.
Use --move-up / --move-down / --to <index> to specify the target position.

Examples:
  subai rule order blackmatrix7/OpenAI --move-up
  subai rule order acl4ssr/BanAD --move-down
  subai rule order "Netflix" --to 0`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.ToLower(args[0])
			moveUp, _ := cmd.Flags().GetBool("move-up")
			moveDown, _ := cmd.Flags().GetBool("move-down")
			toIndex, _ := cmd.Flags().GetInt("to")

			// Validate exactly one flag
			modeCount := 0
			if moveUp {
				modeCount++
			}
			if moveDown {
				modeCount++
			}
			if cmd.Flags().Changed("to") {
				modeCount++
			}
			if modeCount != 1 {
				return fmt.Errorf("exactly one of --move-up, --move-down, or --to <index> is required")
			}

			cfg, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			// Find matching rules
			var matched []int
			for i, rs := range cfg.Output.Template.RuleSets {
				lowerURL := strings.ToLower(rs.URL)
				lowerName := strings.ToLower(rs.ProviderName)
				lowerGroup := strings.ToLower(rs.Group)
				if strings.Contains(lowerURL, query) || strings.Contains(lowerName, query) || strings.Contains(lowerGroup, query) {
					matched = append(matched, i)
				}
			}

			if len(matched) == 0 {
				return fmt.Errorf("no rule sets matching %q found in config", args[0])
			}
			if len(matched) > 1 {
				return fmt.Errorf("multiple rule sets match %q; be more specific\n  Matched: %v", args[0], matched)
			}

			idx := matched[0]
			rules := cfg.Output.Template.RuleSets
			newIdx := idx

			switch {
			case moveUp:
				if idx == 0 {
					return fmt.Errorf("rule is already at the top of the list")
				}
				newIdx = idx - 1
			case moveDown:
				if idx == len(rules)-1 {
					return fmt.Errorf("rule is already at the bottom of the list")
				}
				newIdx = idx + 1
			case cmd.Flags().Changed("to"):
				if toIndex < 0 || toIndex >= len(rules) {
					return fmt.Errorf("--to %d is out of range (valid: 0-%d)", toIndex, len(rules)-1)
				}
				newIdx = toIndex
			}

			// Reorder by removing then inserting at new position
			item := rules[idx]
			rules = append(rules[:idx], rules[idx+1:]...)

			// Adjust newIdx after removal if needed
			if newIdx > idx {
				newIdx--
			}

			// Insert at new position
			rules = append(rules, item) // make room
			copy(rules[newIdx+1:], rules[newIdx:])
			rules[newIdx] = item

			cfg.Output.Template.RuleSets = rules
			if err := cfg.Save(cfgFile); err != nil {
				return fmt.Errorf("save config: %w", err)
			}
			fmt.Printf("  ✅ Moved rule to position %d\n", newIdx)
			return nil
		},
	}
	cmd.Flags().Bool("move-up", false, "Move the rule up one position")
	cmd.Flags().Bool("move-down", false, "Move the rule down one position")
	cmd.Flags().Int("to", -1, "Move the rule to a specific index (0-based)")
	return cmd
}

// newProfileCmd creates the "subai profile" command tree for profile management.
func newProfileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Manage configuration profiles",
		Long: `Manage named configuration profiles for different use cases.

Profiles override top-level config fields (sources, rules, output).
Use 'subai profile switch <name>' to activate a profile.
Use 'subai convert --profile <name>' for one-off use.

Examples:
  subai profile create mobile --template basic
  subai profile create home --template acl4ssr_full
  subai profile switch mobile
  subai convert --profile mobile`,
	}

	// List command
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List configured profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			if len(cfg.Profiles) == 0 {
				fmt.Println("No profiles configured.")
				return nil
			}

			current := cfg.CurrentProfile
			keys := make([]string, 0, len(cfg.Profiles))
			for k := range cfg.Profiles {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			fmt.Printf("  Current profile: \033[33m%s\033[0m\n\n", orDefault(current, "(none)"))
			for _, name := range keys {
				p := cfg.Profiles[name]
				mark := " "
				if name == current {
					mark = "\033[32m*\033[0m"
				}
				srcCount := len(p.Sources)
				outTarget := ""
				if p.Output != nil {
					outTarget = p.Output.Target
				}
				fmt.Printf("  %s \033[1m%s\033[0m  [%d source(s), target=%s]\n",
					mark, name, srcCount, orDefault(outTarget, "inherit"))
			}
			return nil
		},
	}
	cmd.AddCommand(listCmd)

	// Create command
	createCmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new profile",
		Long: `Create a new profile with optional overrides for sources, rules, and output.

If no --template is specified, the profile inherits the root output template.
If no --source is specified, the profile inherits root sources.

Examples:
  subai profile create mobile --template basic
  subai profile create work --template acl4ssr_full --source myvps`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			tmpl, _ := cmd.Flags().GetString("template")
			fetchRules, _ := cmd.Flags().GetBool("fetch-rules")
			ruleProviders, _ := cmd.Flags().GetBool("rule-providers")

			cfg, err := config.Load(cfgFile)
			if err != nil {
				cfg = config.DefaultConfig()
			}

			// Check for duplicates
			if cfg.Profiles == nil {
				cfg.Profiles = make(map[string]config.Profile)
			}
			if _, exists := cfg.Profiles[name]; exists {
				return fmt.Errorf("profile %q already exists", name)
			}

			p := config.Profile{}
			if tmpl != "" || fetchRules || ruleProviders {
				p.Output = &config.Output{
					Target: cfg.Output.Target,
				}
				if tmpl != "" {
					p.Output.Template.Template = tmpl
				}
				p.Output.Template.FetchRules = fetchRules
				p.Output.Template.RuleProviders = ruleProviders
			}

			cfg.Profiles[name] = p
			if err := cfg.Save(cfgFile); err != nil {
				return fmt.Errorf("save config: %w", err)
			}
			fmt.Printf("  ✅ Created profile \033[33m%s\033[0m\n", name)
			return nil
		},
	}
	createCmd.Flags().String("template", "", "Template name for this profile")
	createCmd.Flags().Bool("fetch-rules", false, "Enable fetch_rules in profile")
	createCmd.Flags().Bool("rule-providers", false, "Enable rule-providers in profile")
	cmd.AddCommand(createCmd)

	// Switch command
	switchCmd := &cobra.Command{
		Use:   "switch <name>",
		Short: "Switch the active profile",
		Long: `Set the active profile. Subsequent convert/serve commands will use this profile.
Use 'subai profile switch ""' to deactivate profiles and use root config.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			cfg, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			if name != "" {
				if _, exists := cfg.Profiles[name]; !exists {
					return fmt.Errorf("profile %q not found; use 'subai profile list' to see available profiles", name)
				}
			}

			cfg.CurrentProfile = name
			if err := cfg.Save(cfgFile); err != nil {
				return fmt.Errorf("save config: %w", err)
			}

			if name == "" {
				fmt.Println("  ✅ Profiles deactivated (using root config)")
			} else {
				fmt.Printf("  ✅ Switched to profile \033[33m%s\033[0m\n", name)
			}
			return nil
		},
	}
	cmd.AddCommand(switchCmd)

	// Delete command
	deleteCmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			cfg, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			if _, exists := cfg.Profiles[name]; !exists {
				return fmt.Errorf("profile %q not found", name)
			}

			delete(cfg.Profiles, name)
			if cfg.CurrentProfile == name {
				cfg.CurrentProfile = ""
			}

			if err := cfg.Save(cfgFile); err != nil {
				return fmt.Errorf("save config: %w", err)
			}
			fmt.Printf("  ✅ Deleted profile \033[33m%s\033[0m\n", name)
			return nil
		},
	}
	cmd.AddCommand(deleteCmd)

	return cmd
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}