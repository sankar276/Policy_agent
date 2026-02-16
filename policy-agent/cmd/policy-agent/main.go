package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"policy-agent/internal/agent"
	"policy-agent/internal/ai"
	"policy-agent/internal/config"
	"policy-agent/internal/policy"
	"policy-agent/internal/validator"
	"policy-agent/internal/validator/kafka"
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
	)

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration files against policies",
		Long:  `Validate configuration files against OPA/Rego policies for the specified domain.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runValidate(file, directory, domain, format)
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "File to validate")
	cmd.Flags().StringVarP(&directory, "dir", "d", "", "Directory to validate")
	cmd.Flags().StringVar(&domain, "domain", "", "Specific domain to validate (kafka, kubernetes, etc.)")
	cmd.Flags().StringVar(&format, "format", "text", "Output format (text, json, yaml)")

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

	cmd.Flags().StringVar(&domain, "domain", "", "Target domain (kafka, kubernetes, etc.)")
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
		&cobra.Command{
			Use:   "list",
			Short: "List available policies",
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Println("Policy list - Coming soon!")
				return nil
			},
		},
	)

	return cmd
}

func runValidate(file, directory, domain, format string) error {
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
	policyEngine, err := policy.NewEngine(cfg.Policy.PolicyPath)
	if err != nil {
		return fmt.Errorf("failed to create policy engine: %w", err)
	}

	// Load policies
	if err := policyEngine.LoadPolicies(ctx); err != nil {
		return fmt.Errorf("failed to load policies: %w", err)
	}

	fmt.Printf("✓ Loaded policies from: %s\n", cfg.Policy.PolicyPath)

	// Create validator registry
	registry := validator.NewRegistry()

	// Register Kafka validator
	kafkaConfig := &kafka.Config{
		MinReplicationFactor:     cfg.Domains.Kafka.MinReplicationFactor,
		RequireMinInsyncReplicas: cfg.Domains.Kafka.RequireMinInsyncReplicas,
		MinInsyncReplicasValue:   cfg.Domains.Kafka.MinInsyncReplicasValue,
		RequireCompression:       cfg.Domains.Kafka.RequireCompression,
		AllowedCompressionTypes:  cfg.Domains.Kafka.AllowedCompressionTypes,
		MaxRetentionDays:         cfg.Domains.Kafka.MaxRetentionDays,
		WarnRetentionDays:        cfg.Domains.Kafka.WarnRetentionDays,
	}
	kafkaValidator, err := kafka.NewValidator(policyEngine, kafkaConfig)
	if err != nil {
		return fmt.Errorf("failed to create Kafka validator: %w", err)
	}
	registry.Register(kafkaValidator)

	fmt.Printf("✓ Registered validators: %v\n\n", registry.Domains())

	// Create orchestrator
	orchestrator := agent.New(registry, policyEngine, nil) // AI client nil for now

	// Read file
	if file != "" {
		return validateFile(ctx, orchestrator, file, domain, format)
	}

	// TODO: Handle directory validation
	return fmt.Errorf("directory validation not yet implemented")
}

func validateFile(ctx context.Context, orchestrator *agent.Orchestrator, filePath, domain, format string) error {
	// Read file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Prepare validation request
	req := &agent.ValidateRequest{
		Raw:     content,
		Format:  types.FormatYAML, // TODO: Auto-detect format
		Domain:  domain,
		File:    filePath,
		AutoFix: false,
	}

	// Validate
	fmt.Printf("Validating: %s\n", filePath)
	resp, err := orchestrator.Validate(ctx, req)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Display results
	displayResults(resp.Result, format)

	// Exit with error if there are violations
	if len(resp.Result.Violations) > 0 {
		return fmt.Errorf("validation failed with %d violation(s)", len(resp.Result.Violations))
	}

	return nil
}

func displayResults(result *types.ValidationResult, format string) {
	if format == "json" {
		// TODO: JSON output
		fmt.Println("JSON output not yet implemented")
		return
	}

	// Text format
	fmt.Println("\n" + repeat("=", 70))
	fmt.Printf("Domain: %s (%s)\n", result.Domain, result.ResourceType)
	fmt.Printf("Resource: %s\n", result.Resource)
	fmt.Printf("Status: %s\n", result.Status)
	fmt.Printf("Duration: %v\n", result.Duration)
	fmt.Println(repeat("=", 70))

	// Violations
	if len(result.Violations) > 0 {
		fmt.Println("\n❌ VIOLATIONS:")
		for i, v := range result.Violations {
			fmt.Printf("\n%d. [%s] %s\n", i+1, v.Severity, v.Policy)
			fmt.Printf("   → %s\n", v.Message)
			if v.Field != "" {
				fmt.Printf("   Field: %s\n", v.Field)
				if v.CurrentValue != nil {
					fmt.Printf("   Current: %v\n", v.CurrentValue)
				}
				if v.ExpectedValue != nil {
					fmt.Printf("   Expected: %v\n", v.ExpectedValue)
				}
			}
			if v.Remediation != nil && v.Remediation.Suggestion != "" {
				fmt.Printf("\n   💡 Suggestion: %s\n", v.Remediation.Suggestion)
			}
		}
	}

	// Warnings
	if len(result.Warnings) > 0 {
		fmt.Println("\n⚠️  WARNINGS:")
		for i, w := range result.Warnings {
			fmt.Printf("\n%d. [%s] %s\n", i+1, w.Severity, w.Policy)
			fmt.Printf("   → %s\n", w.Message)
		}
	}

	// Passed checks
	if len(result.Passed) > 0 {
		fmt.Println("\n✅ PASSED:")
		for _, p := range result.Passed {
			fmt.Printf("   • %s\n", p.Policy)
		}
	}

	fmt.Println()
}

// Helper function for string repetition
func repeat(s string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}
// Add these functions at the end of main.go

func runGenerate(domain, requirements, outputFile string) error {
	ctx := context.Background()

	fmt.Printf("🤖 Generating %s configuration using Claude AI...\n\n", domain)
	fmt.Printf("Requirements: %s\n\n", requirements)

	// Load configuration
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create AI client
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

	// Prepare generation request
	req := &types.GenerateRequest{
		Domain:       domain,
		Requirements: requirements,
		Policies: []string{
			fmt.Sprintf("%s.topics.replication", domain),
			fmt.Sprintf("%s.topics.compression", domain),
			fmt.Sprintf("%s.topics.retention", domain),
		},
	}

	// Generate configuration
	fmt.Println("⏳ Calling Claude API...")
	resp, err := aiClient.Generate(ctx, req)
	if err != nil {
		return fmt.Errorf("generation failed: %w", err)
	}

	fmt.Println("✅ Configuration generated!\n")
	fmt.Println(repeat("=", 70))
	fmt.Println("GENERATED CONFIGURATION:")
	fmt.Println(repeat("=", 70))
	fmt.Println(resp.Configuration)
	fmt.Println()

	if resp.Explanation != "" {
		fmt.Println(repeat("-", 70))
		fmt.Println("EXPLANATION:")
		fmt.Println(repeat("-", 70))
		fmt.Println(resp.Explanation)
		fmt.Println()
	}

	// Write to file if output specified
	if outputFile != "" {
		if err := os.WriteFile(outputFile, []byte(resp.Configuration), 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Printf("✅ Saved to: %s\n", outputFile)
	}

	return nil
}

func runFix(file, outputFile, interactive bool) error {
	ctx := context.Background()

	// Read file
	content, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	fmt.Printf("🔧 Fixing policy violations in: %s\n\n", file)

	// Load configuration
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize policy engine and validators (same as validate command)
	policyEngine, err := policy.NewEngine(cfg.Policy.PolicyPath)
	if err != nil {
		return fmt.Errorf("failed to create policy engine: %w", err)
	}

	if err := policyEngine.LoadPolicies(ctx); err != nil {
		return fmt.Errorf("failed to load policies: %w", err)
	}

	registry := validator.NewRegistry()
	kafkaConfig := &kafka.Config{
		MinReplicationFactor:     cfg.Domains.Kafka.MinReplicationFactor,
		RequireMinInsyncReplicas: cfg.Domains.Kafka.RequireMinInsyncReplicas,
		MinInsyncReplicasValue:   cfg.Domains.Kafka.MinInsyncReplicasValue,
		RequireCompression:       cfg.Domains.Kafka.RequireCompression,
		AllowedCompressionTypes:  cfg.Domains.Kafka.AllowedCompressionTypes,
		MaxRetentionDays:         cfg.Domains.Kafka.MaxRetentionDays,
		WarnRetentionDays:        cfg.Domains.Kafka.WarnRetentionDays,
	}
	kafkaValidator, err := kafka.NewValidator(policyEngine, kafkaConfig)
	if err != nil {
		return fmt.Errorf("failed to create Kafka validator: %w", err)
	}
	registry.Register(kafkaValidator)

	// Create AI client
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

	// Create orchestrator with AI client
	orchestrator := agent.New(registry, policyEngine, aiClient)

	// Validate first to find violations
	fmt.Println("📋 Validating current configuration...")
	validateReq := &agent.ValidateRequest{
		Raw:     content,
		Format:  types.FormatYAML,
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
	displayResults(validateResp.Result, "text")

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
	fmt.Println(repeat("=", 70))
	fmt.Println("FIXED CONFIGURATION:")
	fmt.Println(repeat("=", 70))
	fmt.Println(fixResp.Fixed)
	fmt.Println()

	if fixResp.Explanation != "" {
		fmt.Println(repeat("-", 70))
		fmt.Println("CHANGES MADE:")
		fmt.Println(repeat("-", 70))
		fmt.Println(fixResp.Explanation)
		fmt.Println()
	}

	// Interactive mode: ask before saving
	if interactive {
		fmt.Print("Apply these fixes? (y/n): ")
		var answer string
		fmt.Scanln(&answer)
		if answer != "y" && answer != "Y" {
			fmt.Println("Fixes not applied.")
			return nil
		}
	}

	// Determine output file
	if outputFile == "" {
		outputFile = file // Overwrite original
	}

	// Write fixed configuration
	if err := os.WriteFile(outputFile, []byte(fixResp.Fixed), 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	fmt.Printf("✅ Fixed configuration saved to: %s\n", outputFile)

	return nil
}
