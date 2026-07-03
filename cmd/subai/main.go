package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/user/subai/internal/config"
	"github.com/user/subai/internal/converter"
	"github.com/user/subai/internal/fetcher"
	"github.com/user/subai/internal/login"
	"github.com/user/subai/internal/parser"
	"github.com/user/subai/internal/rule"
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
	rootCmd.AddCommand(convertCmd)

	// Validate command
	validateCmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration and rules",
		Long:  `Check the config file for errors and validate all rule patterns.`,
		RunE:  runValidate,
	}
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
		Short: "List and inspect built-in templates",
	}
	templateCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List available built-in templates",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Available templates:")
			for _, name := range template.AvailableTemplates() {
				desc := ""
				switch name {
				case "basic":
					desc = "Simple select/url-test/fallback groups"
				case "acl4ssr_full":
					desc = "Full ACL4SSR rules (16 proxy groups, 16 rule sets)"
				case "acl4ssr_lite":
					desc = "Lite ACL4SSR rules (5 groups, 7 rule sets)"
				case "loyalsoldier":
					desc = "Loyalsoldier/clash-rules (6 groups, 8 rule sets)"
				}
				fmt.Printf("  %-20s %s\n", name, desc)
			}
			fmt.Println()
			fmt.Println("Use `fetch_rules: true` in config to expand rule URLs inline.")
			fmt.Println("Rule sources: jsDelivr CDN (auto-updates from GitHub upstream)")
			return nil
		},
	})
	rootCmd.AddCommand(templateCmd)

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

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}