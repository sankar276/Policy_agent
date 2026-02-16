package cli

import (
	"context"
	"fmt"

	"policy-agent/internal/config"
	"policy-agent/internal/policy"
	"policy-agent/internal/validator"
	"policy-agent/internal/validator/cicd"
	"policy-agent/internal/validator/gitops"
	"policy-agent/internal/validator/iac"
	"policy-agent/internal/validator/kafka"
	"policy-agent/internal/validator/kubernetes"
)

// SetupValidators initializes and registers all domain validators
func SetupValidators(cfg *config.Config, policyEngine *policy.Engine) (*validator.Registry, error) {
	registry := validator.NewRegistry()

	// Register Kafka validator
	if cfg.Policy.IsDomainEnabled("kafka") {
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
			return nil, fmt.Errorf("failed to create Kafka validator: %w", err)
		}
		registry.Register(kafkaValidator)
	}

	// Register Kubernetes validator
	if cfg.Policy.IsDomainEnabled("kubernetes") {
		k8sConfig := &kubernetes.Config{
			RequireResourceLimits:      cfg.Domains.Kubernetes.EnforceResourceLimits,
			RequireLabels:              true,
			RequiredLabels:             cfg.Domains.Kubernetes.RequiredLabels,
			DisallowLatestTag:          true,
			DisallowPrivilegedContainers: true,
			RequireReadOnlyRootFS:      false,
		}
		k8sValidator := kubernetes.NewValidator(policyEngine, k8sConfig)
		registry.Register(k8sValidator)
	}

	// Register IaC validator
	if cfg.Policy.IsDomainEnabled("iac") {
		iacConfig := iac.DefaultConfig()
		iacValidator := iac.NewValidator(policyEngine, iacConfig)
		registry.Register(iacValidator)
	}

	// Register CI/CD validator
	if cfg.Policy.IsDomainEnabled("cicd") {
		cicdConfig := cicd.DefaultConfig()
		cicdValidator := cicd.NewValidator(policyEngine, cicdConfig)
		registry.Register(cicdValidator)
	}

	// Register GitOps validator
	if cfg.Policy.IsDomainEnabled("gitops") {
		gitopsConfig := gitops.DefaultConfig()
		gitopsValidator := gitops.NewValidator(policyEngine, gitopsConfig)
		registry.Register(gitopsValidator)
	}

	return registry, nil
}

// LoadPolicyEngine initializes the policy engine and loads policies
func LoadPolicyEngine(ctx context.Context, policyPath string) (*policy.Engine, error) {
	engine, err := policy.NewEngine(policyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create policy engine: %w", err)
	}

	if err := engine.LoadPolicies(ctx); err != nil {
		return nil, fmt.Errorf("failed to load policies: %w", err)
	}

	return engine, nil
}
