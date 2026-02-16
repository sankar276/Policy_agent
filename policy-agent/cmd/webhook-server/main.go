package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"policy-agent/internal/agent"
	"policy-agent/internal/cli"
	"policy-agent/internal/config"
	"policy-agent/internal/webhook"

	"github.com/spf13/cobra"
)

var (
	cfgFile  string
	port     int
	certFile string
	keyFile  string
	Version  = "0.1.0"
)

func main() {
	rootCmd := &cobra.Command{
		Use:     "webhook-server",
		Short:   "Policy Agent Kubernetes Admission Webhook Server",
		Long:    `Kubernetes admission webhook server for real-time policy validation and enforcement.`,
		Version: Version,
		RunE:    runWebhook,
	}

	// Flags
	rootCmd.Flags().StringVar(&cfgFile, "config", "", "config file (default is /etc/policy-agent/config.yaml)")
	rootCmd.Flags().IntVar(&port, "port", 8443, "webhook server port")
	rootCmd.Flags().StringVar(&certFile, "tls-cert", "/etc/certs/tls.crt", "TLS certificate file")
	rootCmd.Flags().StringVar(&keyFile, "tls-key", "/etc/certs/tls.key", "TLS key file")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runWebhook(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Load configuration
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Println("🔧 Initializing Policy Agent Webhook Server...")
	fmt.Printf("   Version: %s\n", Version)
	fmt.Printf("   Config: %s\n", cfgFile)
	fmt.Println()

	// Initialize policy engine
	fmt.Printf("📚 Loading policies from: %s\n", cfg.Policy.PolicyPath)
	policyEngine, err := cli.LoadPolicyEngine(ctx, cfg.Policy.PolicyPath)
	if err != nil {
		return err
	}
	fmt.Println("✓ Policies loaded")

	// Setup validators
	fmt.Println("🔍 Registering validators...")
	registry, err := cli.SetupValidators(cfg, policyEngine)
	if err != nil {
		return err
	}
	fmt.Printf("✓ Registered domains: %v\n", registry.Domains())
	fmt.Println()

	// Create orchestrator
	orchestrator := agent.New(registry, policyEngine, nil)

	// Create webhook server
	webhookConfig := &webhook.Config{
		Port:     port,
		CertFile: certFile,
		KeyFile:  keyFile,
	}

	server := webhook.NewServer(orchestrator, webhookConfig)

	// Setup graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := server.Start(); err != nil {
			errChan <- err
		}
	}()

	// Wait for shutdown signal or error
	select {
	case <-stop:
		fmt.Println("\n🛑 Shutting down webhook server...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Stop(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Error during shutdown: %v\n", err)
			return err
		}
		fmt.Println("✓ Server stopped gracefully")

	case err := <-errChan:
		return fmt.Errorf("webhook server error: %w", err)
	}

	return nil
}
