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
	"github.com/user/subai/internal/parser"
	"github.com/user/subai/internal/rule"
	"github.com/user/subai/internal/server"
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
		cfg, err := config.Load(cfgFile)
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

	eng := converter.New()
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