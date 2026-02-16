package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"policy-agent/internal/agent"
	"policy-agent/internal/ai"
	"policy-agent/internal/cli"
	"policy-agent/internal/config"
	"policy-agent/pkg/types"

	"github.com/spf13/cobra"
)

var (
	cfgFile string
	Version = "0.1.0"
)

func main() {
	rootCmd := &cobra.Command{
		Use:     "policy-agent",
		Short:   "Unified policy validation and enforcement tool",
		Long:    `AI-powered policy validation, generation, and enforcement across multiple infrastructure domains.`,
		Version: Version,
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./config/policy-agent.yaml)")

	// Add commands
	rootCmd.AddCommand(
		newValidateCommand(),
		newGenerateCommand(),
		newFixCommand(),
		newPolicyCommand(),
		newHooksCommand(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func newValidateCommand() *cobra.Command {
	var (
		file      string
		directory string
		domain    string
		format    string
		noColor   bool
		failOn    string
	)

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration files against policies",
		Long:  `Validate configuration files against OPA/Rego policies for the specified domain.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runValidate(file, directory, domain, format, !noColor, failOn)
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "File to validate")
	cmd.Flags().StringVarP(&directory, "dir", "d", "", "Directory to validate")
	cmd.Flags().StringVar(&domain, "domain", "", "Specific domain to validate (kafka, kubernetes, iac, cicd, gitops)")
	cmd.Flags().StringVar(&format, "format", "text", "Output format (text, json, yaml)")
	cmd.Flags().BoolVar(&noColor, "no-color", false, "Disable colored output")
	cmd.Flags().StringVar(&failOn, "fail-on", "violation", "Fail on (violation, warning, or never)")

	return cmd
}

func newGenerateCommand() *cobra.Command {
	var (
		domain       string
		requirements string
		output       string
	)

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate policy-compliant configurations using AI",
		Long:  `Use Claude AI to generate configuration files that comply with all relevant policies.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenerate(domain, requirements, output)
		},
	}

	cmd.Flags().StringVar(&domain, "domain", "", "Target domain (kafka, kubernetes, iac, cicd, gitops)")
	cmd.Flags().StringVar(&requirements, "requirements", "", "Requirements in natural language")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file path")

	cmd.MarkFlagRequired("domain")
	cmd.MarkFlagRequired("requirements")

	return cmd
}

func newFixCommand() *cobra.Command {
	var (
		file        string
		output      string
		interactive bool
	)

	cmd := &cobra.Command{
		Use:   "fix",
		Short: "Fix policy violations using AI",
		Long:  `Automatically fix policy violations in configuration files using Claude AI.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFix(file, output, interactive)
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "File to fix")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file (default: overwrite original)")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Interactive mode (preview before applying)")

	cmd.MarkFlagRequired("file")

	return cmd
}

func newPolicyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "policy",
		Short: "Policy management commands",
		Long:  `Manage OPA/Rego policies: list, test, validate, sync.`,
	}

	cmd.AddCommand(
		newPolicyListCommand(),
		newPolicyTestCommand(),
	)

	return cmd
}

func newPolicyListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available policies",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			fmt.Printf("Policy path: %s\n\n", cfg.Policy.PolicyPath)
			fmt.Println("Enabled domains:")
			for _, domain := range cfg.Policy.EnabledDomains {
				fmt.Printf("  • %s\n", domain)
			}

			return nil
		},
	}
}

func newPolicyTestCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "test",
		Short: "Test policies",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Policy testing - Coming soon!")
			return nil
		},
	}
}

func newHooksCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hooks",
		Short: "Manage Git hooks",
		Long:  `Install, uninstall, or list Git hooks for automatic policy validation.`,
	}

	cmd.AddCommand(
		newHooksInstallCommand(),
		newHooksUninstallCommand(),
		newHooksListCommand(),
	)

	return cmd
}

func newHooksInstallCommand() *cobra.Command {
	var (
		preCommit bool
		prePush   bool
		all       bool
	)

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install Git hooks",
		RunE: func(cmd *cobra.Command, args []string) error {
			manager, err := cli.NewGitHooksManager("")
			if err != nil {
				return err
			}

			var hooks []cli.HookType
			if all || (!preCommit && !prePush) {
				// Install all hooks by default
				hooks = []cli.HookType{cli.HookPreCommit, cli.HookPrePush}
			} else {
				if preCommit {
					hooks = append(hooks, cli.HookPreCommit)
				}
				if prePush {
					hooks = append(hooks, cli.HookPrePush)
				}
			}

			return manager.Install(hooks)
		},
	}

	cmd.Flags().BoolVar(&preCommit, "pre-commit", false, "Install pre-commit hook")
	cmd.Flags().BoolVar(&prePush, "pre-push", false, "Install pre-push hook")
	cmd.Flags().BoolVar(&all, "all", false, "Install all hooks")

	return cmd
}

func newHooksUninstallCommand() *cobra.Command {
	var all bool

	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall Git hooks",
		RunE: func(cmd *cobra.Command, args []string) error {
			manager, err := cli.NewGitHooksManager("")
			if err != nil {
				return err
			}

			hooks := []cli.HookType{cli.HookPreCommit, cli.HookPrePush}
			return manager.Uninstall(hooks)
		},
	}

	cmd.Flags().BoolVar(&all, "all", true, "Uninstall all hooks")

	return cmd
}

func newHooksListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List installed Git hooks",
		RunE: func(cmd *cobra.Command, args []string) error {
			manager, err := cli.NewGitHooksManager("")
			if err != nil {
				return err
			}

			hooks, err := manager.List()
			if err != nil {
				return err
			}

			fmt.Println("Git hooks status:")
			for hookType, installed := range hooks {
				status := "❌ Not installed"
				if installed {
					status = "✅ Installed"
				}
				fmt.Printf("  %s: %s\n", hookType, status)
			}

			return nil
		},
	}
}

// Command implementations

func runValidate(file, directory, domain, format string, color bool, failOn string) error {
	ctx := context.Background()

	// Validate input
	if file == "" && directory == "" {
		return fmt.Errorf("either --file or --dir must be specified")
	}

	if file != "" && directory != "" {
		return fmt.Errorf("cannot specify both --file and --dir")
	}

	// Load configuration
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize policy engine
	policyEngine, err := cli.LoadPolicyEngine(ctx, cfg.Policy.PolicyPath)
	if err != nil {
		return err
	}

	fmt.Printf("✓ Loaded policies from: %s\n", cfg.Policy.PolicyPath)

	// Setup validators
	registry, err := cli.SetupValidators(cfg, policyEngine)
	if err != nil {
		return err
	}

	fmt.Printf("✓ Registered validators: %v\n\n", registry.Domains())

	// Create orchestrator
	orchestrator := agent.New(registry, policyEngine, nil)

	// Create formatter
	formatter := cli.NewFormatter(format, color)

	// Validate file or directory
	if file != "" {
		result, err := validateFile(ctx, orchestrator, file, domain)
		if err != nil {
			return err
		}

		output, err := formatter.FormatResult(result)
		if err != nil {
			return err
		}
		fmt.Print(output)

		return checkFailCondition(result, failOn)
	}

	// Validate directory
	results, err := validateDirectory(ctx, orchestrator, directory, domain)
	if err != nil {
		return err
	}

	output, err := formatter.FormatBatchResults(results)
	if err != nil {
		return err
	}
	fmt.Print(output)

	// Check fail condition for batch
	for _, result := range results {
		if err := checkFailCondition(result, failOn); err != nil {
			return err
		}
	}

	return nil
}

func validateFile(ctx context.Context, orchestrator *agent.Orchestrator, filePath, domain string) (*types.ValidationResult, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Auto-detect format
	format := detectFormat(filePath)

	req := &agent.ValidateRequest{
		Raw:     content,
		Format:  format,
		Domain:  domain,
		File:    filePath,
		AutoFix: false,
	}

	fmt.Printf("Validating: %s\n", filePath)
	resp, err := orchestrator.Validate(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return resp.Result, nil
}

func validateDirectory(ctx context.Context, orchestrator *agent.Orchestrator, dirPath, domain string) ([]*types.ValidationResult, error) {
	var results []*types.ValidationResult

	// Walk directory
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Check if file is a configuration file
		if !isConfigFile(path) {
			return nil
		}

		result, err := validateFile(ctx, orchestrator, path, domain)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to validate %s: %v\n", path, err)
			return nil // Continue with other files
		}

		results = append(results, result)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no configuration files found in directory")
	}

	return results, nil
}

func runGenerate(domain, requirements, outputFile string) error {
	ctx := context.Background()

	fmt.Printf("🤖 Generating %s configuration using Claude AI...\n\n", domain)
	fmt.Printf("Requirements: %s\n\n", requirements)

	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	aiConfig := &ai.ClaudeConfig{
		APIKey:            cfg.AI.GetAPIKey(),
		Model:             cfg.AI.Model,
		RequestsPerMinute: cfg.AI.RateLimiting.RequestsPerMinute,
		EnableCache:       cfg.AI.Cache.Enabled,
		CacheTTL:          1 * time.Hour,
	}

	aiClient, err := ai.NewClaudeClient(aiConfig)
	if err != nil {
		return fmt.Errorf("failed to create AI client: %w", err)
	}

	req := &types.GenerateRequest{
		Domain:       domain,
		Requirements: requirements,
	}

	fmt.Println("⏳ Calling Claude API...")
	resp, err := aiClient.Generate(ctx, req)
	if err != nil {
		return fmt.Errorf("generation failed: %w", err)
	}

	fmt.Println("✅ Configuration generated!\n")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println("GENERATED CONFIGURATION:")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println(resp.Configuration)
	fmt.Println()

	if resp.Explanation != "" {
		fmt.Println(strings.Repeat("-", 70))
		fmt.Println("EXPLANATION:")
		fmt.Println(strings.Repeat("-", 70))
		fmt.Println(resp.Explanation)
		fmt.Println()
	}

	if outputFile != "" {
		if err := os.WriteFile(outputFile, []byte(resp.Configuration), 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Printf("✅ Saved to: %s\n", outputFile)
	}

	return nil
}

func runFix(file, outputFile string, interactive bool) error {
	ctx := context.Background()

	content, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	fmt.Printf("🔧 Fixing policy violations in: %s\n\n", file)

	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	policyEngine, err := cli.LoadPolicyEngine(ctx, cfg.Policy.PolicyPath)
	if err != nil {
		return err
	}

	registry, err := cli.SetupValidators(cfg, policyEngine)
	if err != nil {
		return err
	}

	aiConfig := &ai.ClaudeConfig{
		APIKey:            cfg.AI.GetAPIKey(),
		Model:             cfg.AI.Model,
		RequestsPerMinute: cfg.AI.RateLimiting.RequestsPerMinute,
		EnableCache:       cfg.AI.Cache.Enabled,
		CacheTTL:          1 * time.Hour,
	}

	aiClient, err := ai.NewClaudeClient(aiConfig)
	if err != nil {
		return fmt.Errorf("failed to create AI client: %w", err)
	}

	orchestrator := agent.New(registry, policyEngine, aiClient)

	// Validate first
	fmt.Println("📋 Validating current configuration...")
	validateReq := &agent.ValidateRequest{
		Raw:     content,
		Format:  detectFormat(file),
		File:    file,
		AutoFix: false,
	}

	validateResp, err := orchestrator.Validate(ctx, validateReq)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if len(validateResp.Result.Violations) == 0 {
		fmt.Println("✅ No violations found! Configuration is already compliant.")
		return nil
	}

	fmt.Printf("Found %d violation(s)\n\n", len(validateResp.Result.Violations))

	formatter := cli.NewFormatter("text", true)
	output, _ := formatter.FormatResult(validateResp.Result)
	fmt.Print(output)

	// Use AI to fix
	fmt.Println("\n⏳ Generating fixes using Claude AI...")
	fixReq := &types.FixRequest{
		File:        string(content),
		Interactive: interactive,
	}

	fixResp, err := orchestrator.Fix(ctx, fixReq)
	if err != nil {
		return fmt.Errorf("fix failed: %w", err)
	}

	fmt.Println("\n✅ Fixes generated!\n")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println("FIXED CONFIGURATION:")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println(fixResp.Fixed)
	fmt.Println()

	if fixResp.Explanation != "" {
		fmt.Println(strings.Repeat("-", 70))
		fmt.Println("CHANGES MADE:")
		fmt.Println(strings.Repeat("-", 70))
		fmt.Println(fixResp.Explanation)
		fmt.Println()
	}

	// Interactive mode
	if interactive {
		fmt.Print("Apply these fixes? (y/n): ")
		var answer string
		fmt.Scanln(&answer)
		if answer != "y" && answer != "Y" {
			fmt.Println("Fixes not applied.")
			return nil
		}
	}

	if outputFile == "" {
		outputFile = file
	}

	if err := os.WriteFile(outputFile, []byte(fixResp.Fixed), 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	fmt.Printf("✅ Fixed configuration saved to: %s\n", outputFile)

	return nil
}

// Helper functions

func detectFormat(filePath string) types.Format {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".yaml", ".yml":
		return types.FormatYAML
	case ".json":
		return types.FormatJSON
	case ".tf", ".hcl":
		return types.FormatHCL
	default:
		return types.FormatYAML
	}
}

func isConfigFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	configExts := []string{".yaml", ".yml", ".json", ".tf", ".hcl"}

	for _, configExt := range configExts {
		if ext == configExt {
			return true
		}
	}

	return false
}

func checkFailCondition(result *types.ValidationResult, failOn string) error {
	switch strings.ToLower(failOn) {
	case "violation":
		if len(result.Violations) > 0 {
			return fmt.Errorf("validation failed with %d violation(s)", len(result.Violations))
		}
	case "warning":
		if len(result.Warnings) > 0 || len(result.Violations) > 0 {
			return fmt.Errorf("validation failed with %d violation(s) and %d warning(s)",
				len(result.Violations), len(result.Warnings))
		}
	case "never":
		return nil
	}

	return nil
}
