package gitops

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"policy-agent/internal/policy"
	"policy-agent/internal/validator"
	"policy-agent/pkg/types"
)

// Config holds configuration for GitOps validator
type Config struct {
	RequirePrune           bool
	RequireHealthChecks    bool
	RequireGPGVerification bool
	MaxIntervalSeconds     int
	RequireRollback        bool
	AllowAutoSync          bool
}

// DefaultConfig returns default GitOps validation configuration
func DefaultConfig() *Config {
	return &Config{
		RequirePrune:           true,
		RequireHealthChecks:    true,
		RequireGPGVerification: false,
		MaxIntervalSeconds:     600,
		RequireRollback:        true,
		AllowAutoSync:          true,
	}
}

// Validator validates GitOps configurations (Flux CD and ArgoCD)
type Validator struct {
	engine *policy.Engine
	config *Config
}

// NewValidator creates a new GitOps validator
func NewValidator(engine *policy.Engine, config *Config) *Validator {
	if config == nil {
		config = DefaultConfig()
	}
	return &Validator{
		engine: engine,
		config: config,
	}
}

// Domain returns the domain name
func (v *Validator) Domain() string {
	return "gitops"
}

// SupportedTypes returns supported GitOps resource types
func (v *Validator) SupportedTypes() []string {
	return []string{
		// Flux CD
		"Kustomization",
		"GitRepository",
		"HelmRelease",
		// ArgoCD
		"Application",
		"AppProject",
	}
}

// CanValidate checks if this validator can handle the input
func (v *Validator) CanValidate(input *validator.Input) bool {
	if input.ResourceType == "" {
		return false
	}

	for _, t := range v.SupportedTypes() {
		if input.ResourceType == t {
			return true
		}
	}
	return false
}

// Validate validates GitOps configuration
func (v *Validator) Validate(ctx context.Context, input *validator.Input) (*types.ValidationResult, error) {
	var config map[string]interface{}
	if err := json.Unmarshal(input.Raw, &config); err != nil {
		return nil, fmt.Errorf("failed to parse GitOps config: %w", err)
	}

	result := &types.ValidationResult{
		Domain:       v.Domain(),
		ResourceType: input.ResourceType,
		Status:       types.StatusPassed,
	}

	// Extract resource name
	if metadata, ok := config["metadata"].(map[string]interface{}); ok {
		if name, ok := metadata["name"].(string); ok {
			result.Resource = name
		}
	}
	if result.Resource == "" {
		result.Resource = "unknown"
	}

	// Validate based on type
	switch input.ResourceType {
	case "Kustomization":
		v.validateFluxKustomization(ctx, config, result)
	case "GitRepository":
		v.validateFluxGitRepository(ctx, config, result)
	case "HelmRelease":
		v.validateFluxHelmRelease(ctx, config, result)
	case "Application":
		v.validateArgoCDApplication(ctx, config, result)
	case "AppProject":
		v.validateArgoCDAppProject(ctx, config, result)
	default:
		result.Warnings = append(result.Warnings, types.Warning{
			Policy:   fmt.Sprintf("%s.unsupported", v.Domain()),
			Severity: types.SeverityMedium,
			Message:  fmt.Sprintf("Validation not yet implemented for %s", input.ResourceType),
		})
	}

	// Set overall status
	if len(result.Violations) > 0 {
		result.Status = types.StatusFailed
	} else if len(result.Warnings) > 0 {
		result.Status = types.StatusWarning
	}

	return result, nil
}

// validateFluxKustomization validates Flux Kustomization resources
func (v *Validator) validateFluxKustomization(ctx context.Context, config map[string]interface{}, result *types.ValidationResult) {
	spec, ok := config["spec"].(map[string]interface{})
	if !ok {
		result.Violations = append(result.Violations, types.Violation{
			Policy:   "gitops.flux.kustomization",
			Severity: types.SeverityHigh,
			Message:  "Kustomization must have spec",
			Field:    "spec",
		})
		return
	}

	// Check prune enabled
	if v.config.RequirePrune {
		if prune, ok := spec["prune"].(bool); !ok || !prune {
			result.Violations = append(result.Violations, types.Violation{
				Policy:   "gitops.flux.kustomization",
				Severity: types.SeverityHigh,
				Message:  "Kustomization should enable prune to clean up deleted resources",
				Field:    "spec.prune",
			})
		}
	}

	// Check source reference
	if _, ok := spec["sourceRef"]; !ok {
		result.Violations = append(result.Violations, types.Violation{
			Policy:   "gitops.flux.kustomization",
			Severity: types.SeverityHigh,
			Message:  "Kustomization must specify sourceRef",
			Field:    "spec.sourceRef",
		})
	}

	// Check interval
	if interval, ok := spec["interval"].(string); ok {
		if !v.isValidInterval(interval) {
			result.Warnings = append(result.Warnings, types.Warning{
				Policy:   "gitops.flux.kustomization",
				Severity: types.SeverityLow,
				Message:  fmt.Sprintf("Kustomization interval '%s' may be too short or too long", interval),
				Field:    "spec.interval",
			})
		}
	} else {
		result.Violations = append(result.Violations, types.Violation{
			Policy:   "gitops.flux.kustomization",
			Severity: types.SeverityMedium,
			Message:  "Kustomization must specify reconciliation interval",
			Field:    "spec.interval",
		})
	}

	// Check force sync (dangerous)
	if force, ok := spec["force"].(bool); ok && force {
		result.Warnings = append(result.Warnings, types.Warning{
			Policy:   "gitops.flux.kustomization",
			Severity: types.SeverityHigh,
			Message:  "Kustomization has force enabled - can cause unexpected resource replacements",
			Field:    "spec.force",
		})
	}

	// Check health checks for production
	if v.config.RequireHealthChecks && v.isProduction(config) {
		if _, ok := spec["healthChecks"]; !ok {
			result.Warnings = append(result.Warnings, types.Warning{
				Policy:   "gitops.flux.kustomization",
				Severity: types.SeverityMedium,
				Message:  "Production Kustomization should configure health checks",
				Field:    "spec.healthChecks",
			})
		}
	}

	// Check service account
	if _, ok := spec["serviceAccountName"]; !ok {
		result.Warnings = append(result.Warnings, types.Warning{
			Policy:   "gitops.flux.kustomization",
			Severity: types.SeverityLow,
			Message:  "Kustomization should specify serviceAccountName for RBAC",
			Field:    "spec.serviceAccountName",
		})
	}

	// Check path traversal
	if path, ok := spec["path"].(string); ok {
		if strings.Contains(path, "..") {
			result.Violations = append(result.Violations, types.Violation{
				Policy:   "gitops.flux.kustomization",
				Severity: types.SeverityCritical,
				Message:  "Kustomization path contains '..' - potential path traversal vulnerability",
				Field:    "spec.path",
			})
		}
	}
}

// validateFluxGitRepository validates Flux GitRepository sources
func (v *Validator) validateFluxGitRepository(ctx context.Context, config map[string]interface{}, result *types.ValidationResult) {
	spec, ok := config["spec"].(map[string]interface{})
	if !ok {
		result.Violations = append(result.Violations, types.Violation{
			Policy:   "gitops.flux.gitrepository",
			Severity: types.SeverityHigh,
			Message:  "GitRepository must have spec",
			Field:    "spec",
		})
		return
	}

	// Check URL protocol
	if url, ok := spec["url"].(string); ok {
		if strings.HasPrefix(url, "http://") {
			result.Violations = append(result.Violations, types.Violation{
				Policy:   "gitops.flux.gitrepository",
				Severity: types.SeverityHigh,
				Message:  fmt.Sprintf("GitRepository URL '%s' uses insecure HTTP - use HTTPS or SSH", url),
				Field:    "spec.url",
			})
		}
	} else {
		result.Violations = append(result.Violations, types.Violation{
			Policy:   "gitops.flux.gitrepository",
			Severity: types.SeverityHigh,
			Message:  "GitRepository must specify URL",
			Field:    "spec.url",
		})
	}

	// Check GPG verification for production
	if v.config.RequireGPGVerification && v.isProduction(config) {
		if verify, ok := spec["verify"].(map[string]interface{}); ok {
			if mode, ok := verify["mode"].(string); !ok || mode == "" {
				result.Warnings = append(result.Warnings, types.Warning{
					Policy:   "gitops.flux.gitrepository",
					Severity: types.SeverityMedium,
					Message:  "Production GitRepository should enable GPG verification",
					Field:    "spec.verify",
				})
			}
		} else {
			result.Warnings = append(result.Warnings, types.Warning{
				Policy:   "gitops.flux.gitrepository",
				Severity: types.SeverityMedium,
				Message:  "Production GitRepository should configure GPG verification",
				Field:    "spec.verify",
			})
		}
	}

	// Check interval
	if interval, ok := spec["interval"].(string); ok {
		if !v.isValidInterval(interval) {
			result.Warnings = append(result.Warnings, types.Warning{
				Policy:   "gitops.flux.gitrepository",
				Severity: types.SeverityLow,
				Message:  fmt.Sprintf("GitRepository interval '%s' may be too short or too long", interval),
				Field:    "spec.interval",
			})
		}
	} else {
		result.Violations = append(result.Violations, types.Violation{
			Policy:   "gitops.flux.gitrepository",
			Severity: types.SeverityMedium,
			Message:  "GitRepository must specify reconciliation interval",
			Field:    "spec.interval",
		})
	}

	// Check ref specified
	if ref, ok := spec["ref"].(map[string]interface{}); !ok || len(ref) == 0 {
		result.Warnings = append(result.Warnings, types.Warning{
			Policy:   "gitops.flux.gitrepository",
			Severity: types.SeverityMedium,
			Message:  "GitRepository should specify explicit ref (branch, tag, or commit)",
			Field:    "spec.ref",
		})
	}
}

// validateFluxHelmRelease validates Flux HelmRelease resources
func (v *Validator) validateFluxHelmRelease(ctx context.Context, config map[string]interface{}, result *types.ValidationResult) {
	spec, ok := config["spec"].(map[string]interface{})
	if !ok {
		result.Violations = append(result.Violations, types.Violation{
			Policy:   "gitops.flux.helmrelease",
			Severity: types.SeverityHigh,
			Message:  "HelmRelease must have spec",
			Field:    "spec",
		})
		return
	}

	// Check chart reference
	if _, ok := spec["chart"]; !ok {
		result.Violations = append(result.Violations, types.Violation{
			Policy:   "gitops.flux.helmrelease",
			Severity: types.SeverityHigh,
			Message:  "HelmRelease must specify chart reference",
			Field:    "spec.chart",
		})
	}

	// Check interval
	if _, ok := spec["interval"]; !ok {
		result.Violations = append(result.Violations, types.Violation{
			Policy:   "gitops.flux.helmrelease",
			Severity: types.SeverityMedium,
			Message:  "HelmRelease must specify reconciliation interval",
			Field:    "spec.interval",
		})
	}

	// Check rollback for production
	if v.config.RequireRollback && v.isProduction(config) {
		if rollback, ok := spec["rollback"].(map[string]interface{}); ok {
			if enable, ok := rollback["enable"].(bool); ok && !enable {
				result.Violations = append(result.Violations, types.Violation{
					Policy:   "gitops.flux.helmrelease",
					Severity: types.SeverityHigh,
					Message:  "Production HelmRelease has rollback disabled",
					Field:    "spec.rollback.enable",
				})
			}
		} else {
			result.Violations = append(result.Violations, types.Violation{
				Policy:   "gitops.flux.helmrelease",
				Severity: types.SeverityHigh,
				Message:  "Production HelmRelease should configure automatic rollback",
				Field:    "spec.rollback",
			})
		}
	}

	// Check timeout
	if _, ok := spec["timeout"]; !ok {
		result.Warnings = append(result.Warnings, types.Warning{
			Policy:   "gitops.flux.helmrelease",
			Severity: types.SeverityMedium,
			Message:  "HelmRelease should set timeout to prevent hanging installations",
			Field:    "spec.timeout",
		})
	}

	// Check upgrade configuration
	if upgrade, ok := spec["upgrade"].(map[string]interface{}); ok {
		if force, ok := upgrade["force"].(bool); ok && force {
			result.Warnings = append(result.Warnings, types.Warning{
				Policy:   "gitops.flux.helmrelease",
				Severity: types.SeverityMedium,
				Message:  "HelmRelease has force upgrade enabled - can cause unexpected resource replacements",
				Field:    "spec.upgrade.force",
			})
		}
	}

	// Check chart version pinned
	if chart, ok := spec["chart"].(map[string]interface{}); ok {
		if chartSpec, ok := chart["spec"].(map[string]interface{}); ok {
			if version, ok := chartSpec["version"].(string); ok {
				// Check for version ranges
				if v.isMutableVersion(version) {
					result.Warnings = append(result.Warnings, types.Warning{
						Policy:   "gitops.flux.helmrelease",
						Severity: types.SeverityMedium,
						Message:  fmt.Sprintf("HelmRelease chart version '%s' uses range - pin to specific version", version),
						Field:    "spec.chart.spec.version",
					})
				}
			} else {
				result.Violations = append(result.Violations, types.Violation{
					Policy:   "gitops.flux.helmrelease",
					Severity: types.SeverityMedium,
					Message:  "HelmRelease should pin chart version for reproducible deployments",
					Field:    "spec.chart.spec.version",
				})
			}
		}
	}
}

// validateArgoCDApplication validates ArgoCD Application resources
func (v *Validator) validateArgoCDApplication(ctx context.Context, config map[string]interface{}, result *types.ValidationResult) {
	spec, ok := config["spec"].(map[string]interface{})
	if !ok {
		result.Violations = append(result.Violations, types.Violation{
			Policy:   "gitops.argocd.application",
			Severity: types.SeverityHigh,
			Message:  "Application must have spec",
			Field:    "spec",
		})
		return
	}

	// Check source
	if source, ok := spec["source"].(map[string]interface{}); ok {
		// Check repo URL protocol
		if repoURL, ok := source["repoURL"].(string); ok {
			if strings.HasPrefix(repoURL, "http://") {
				result.Violations = append(result.Violations, types.Violation{
					Policy:   "gitops.argocd.application",
					Severity: types.SeverityHigh,
					Message:  fmt.Sprintf("Application source '%s' uses insecure HTTP - use HTTPS or SSH", repoURL),
					Field:    "spec.source.repoURL",
				})
			}
		}

		// Check target revision
		if targetRevision, ok := source["targetRevision"].(string); !ok || targetRevision == "" {
			result.Warnings = append(result.Warnings, types.Warning{
				Policy:   "gitops.argocd.application",
				Severity: types.SeverityMedium,
				Message:  "Application should specify targetRevision for predictability",
				Field:    "spec.source.targetRevision",
			})
		} else if targetRevision == "HEAD" {
			result.Warnings = append(result.Warnings, types.Warning{
				Policy:   "gitops.argocd.application",
				Severity: types.SeverityMedium,
				Message:  "Application targetRevision 'HEAD' is unpredictable - use specific branch or tag",
				Field:    "spec.source.targetRevision",
			})
		}
	} else {
		result.Violations = append(result.Violations, types.Violation{
			Policy:   "gitops.argocd.application",
			Severity: types.SeverityHigh,
			Message:  "Application must specify source repository",
			Field:    "spec.source",
		})
	}

	// Check destination
	if destination, ok := spec["destination"].(map[string]interface{}); ok {
		if namespace, ok := destination["namespace"].(string); !ok || namespace == "" {
			result.Violations = append(result.Violations, types.Violation{
				Policy:   "gitops.argocd.application",
				Severity: types.SeverityMedium,
				Message:  "Application destination must specify namespace",
				Field:    "spec.destination.namespace",
			})
		}
	} else {
		result.Violations = append(result.Violations, types.Violation{
			Policy:   "gitops.argocd.application",
			Severity: types.SeverityHigh,
			Message:  "Application must specify destination cluster and namespace",
			Field:    "spec.destination",
		})
	}

	// Check project assignment
	if project, ok := spec["project"].(string); ok {
		if project == "default" {
			result.Violations = append(result.Violations, types.Violation{
				Policy:   "gitops.argocd.application",
				Severity: types.SeverityMedium,
				Message:  "Application should use a dedicated Project (not 'default') for proper isolation",
				Field:    "spec.project",
			})
		}
	}

	// Check sync policy
	if syncPolicy, ok := spec["syncPolicy"].(map[string]interface{}); ok {
		// Check auto-sync with prune
		if automated, ok := syncPolicy["automated"].(map[string]interface{}); ok {
			if v.isProduction(config) {
				result.Warnings = append(result.Warnings, types.Warning{
					Policy:   "gitops.argocd.application",
					Severity: types.SeverityMedium,
					Message:  "Production Application has automated sync - consider requiring manual approval",
					Field:    "spec.syncPolicy.automated",
				})
			}

			// Check prune enabled
			if v.config.RequirePrune {
				if prune, ok := automated["prune"].(bool); !ok || !prune {
					result.Violations = append(result.Violations, types.Violation{
						Policy:   "gitops.argocd.application",
						Severity: types.SeverityHigh,
						Message:  "Application with automated sync should enable prune to clean up deleted resources",
						Field:    "spec.syncPolicy.automated.prune",
					})
				}
			}
		}

		// Check sync options
		if syncOptions, ok := syncPolicy["syncOptions"].([]interface{}); ok {
			for _, opt := range syncOptions {
				if optStr, ok := opt.(string); ok && optStr == "Replace=true" {
					result.Violations = append(result.Violations, types.Violation{
						Policy:   "gitops.argocd.application",
						Severity: types.SeverityHigh,
						Message:  "Application uses Replace=true sync option - can cause data loss",
						Field:    "spec.syncPolicy.syncOptions",
					})
				}
			}
		}
	}
}

// validateArgoCDAppProject validates ArgoCD AppProject resources
func (v *Validator) validateArgoCDAppProject(ctx context.Context, config map[string]interface{}, result *types.ValidationResult) {
	spec, ok := config["spec"].(map[string]interface{})
	if !ok {
		result.Violations = append(result.Violations, types.Violation{
			Policy:   "gitops.argocd.appproject",
			Severity: types.SeverityHigh,
			Message:  "AppProject must have spec",
			Field:    "spec",
		})
		return
	}

	// Check source repositories
	if sourceRepos, ok := spec["sourceRepos"].([]interface{}); ok {
		if len(sourceRepos) == 0 {
			result.Violations = append(result.Violations, types.Violation{
				Policy:   "gitops.argocd.appproject",
				Severity: types.SeverityHigh,
				Message:  "AppProject has no source repositories defined",
				Field:    "spec.sourceRepos",
			})
		}

		// Check for wildcard
		for _, repo := range sourceRepos {
			if repoStr, ok := repo.(string); ok && repoStr == "*" {
				result.Violations = append(result.Violations, types.Violation{
					Policy:   "gitops.argocd.appproject",
					Severity: types.SeverityHigh,
					Message:  "AppProject allows all repositories (*) - specify explicit repositories for security",
					Field:    "spec.sourceRepos",
				})
			}
		}
	} else {
		result.Violations = append(result.Violations, types.Violation{
			Policy:   "gitops.argocd.appproject",
			Severity: types.SeverityHigh,
			Message:  "AppProject must specify allowed source repositories",
			Field:    "spec.sourceRepos",
		})
	}

	// Check destinations
	if destinations, ok := spec["destinations"].([]interface{}); ok {
		if len(destinations) == 0 {
			result.Violations = append(result.Violations, types.Violation{
				Policy:   "gitops.argocd.appproject",
				Severity: types.SeverityHigh,
				Message:  "AppProject has no destinations defined",
				Field:    "spec.destinations",
			})
		}

		// Check for wildcard namespaces
		for _, dest := range destinations {
			if destMap, ok := dest.(map[string]interface{}); ok {
				if namespace, ok := destMap["namespace"].(string); ok && namespace == "*" {
					if server, ok := destMap["server"].(string); ok && server != "https://kubernetes.default.svc" {
						result.Violations = append(result.Violations, types.Violation{
							Policy:   "gitops.argocd.appproject",
							Severity: types.SeverityHigh,
							Message:  "AppProject allows deployment to all namespaces (*) - specify explicit namespaces",
							Field:    "spec.destinations",
						})
					}
				}
			}
		}
	} else {
		result.Violations = append(result.Violations, types.Violation{
			Policy:   "gitops.argocd.appproject",
			Severity: types.SeverityHigh,
			Message:  "AppProject must specify allowed destinations",
			Field:    "spec.destinations",
		})
	}

	// Check RBAC roles for production
	if v.isProduction(config) {
		if roles, ok := spec["roles"].([]interface{}); !ok || len(roles) == 0 {
			result.Violations = append(result.Violations, types.Violation{
				Policy:   "gitops.argocd.appproject",
				Severity: types.SeverityMedium,
				Message:  "Production AppProject should define RBAC roles for access control",
				Field:    "spec.roles",
			})
		}
	}

	// Check cluster resource whitelist
	if clusterResourceWhitelist, ok := spec["clusterResourceWhitelist"].([]interface{}); ok {
		for _, resource := range clusterResourceWhitelist {
			if resMap, ok := resource.(map[string]interface{}); ok {
				if group, ok := resMap["group"].(string); ok && group == "*" {
					if kind, ok := resMap["kind"].(string); ok && kind == "*" {
						result.Violations = append(result.Violations, types.Violation{
							Policy:   "gitops.argocd.appproject",
							Severity: types.SeverityHigh,
							Message:  "AppProject allows all cluster resources (*/*) - specify explicit resources",
							Field:    "spec.clusterResourceWhitelist",
						})
					}
				}
			}
		}
	}

	// Check orphaned resources policy
	if _, ok := spec["orphanedResources"]; !ok {
		result.Warnings = append(result.Warnings, types.Warning{
			Policy:   "gitops.argocd.appproject",
			Severity: types.SeverityMedium,
			Message:  "AppProject should configure orphanedResources policy to detect orphaned resources",
			Field:    "spec.orphanedResources",
		})
	}
}

// Helper functions

func (v *Validator) isProduction(config map[string]interface{}) bool {
	// Check metadata
	if metadata, ok := config["metadata"].(map[string]interface{}); ok {
		// Check name
		if name, ok := metadata["name"].(string); ok {
			nameLower := strings.ToLower(name)
			if strings.Contains(nameLower, "prod") || strings.Contains(nameLower, "production") || strings.Contains(nameLower, "live") {
				return true
			}
		}

		// Check namespace
		if namespace, ok := metadata["namespace"].(string); ok {
			namespaceLower := strings.ToLower(namespace)
			if strings.Contains(namespaceLower, "prod") || strings.Contains(namespaceLower, "production") || strings.Contains(namespaceLower, "live") {
				return true
			}
		}
	}

	// Check spec for ArgoCD Application
	if spec, ok := config["spec"].(map[string]interface{}); ok {
		if destination, ok := spec["destination"].(map[string]interface{}); ok {
			if namespace, ok := destination["namespace"].(string); ok {
				namespaceLower := strings.ToLower(namespace)
				if strings.Contains(namespaceLower, "prod") || strings.Contains(namespaceLower, "production") || strings.Contains(namespaceLower, "live") {
					return true
				}
			}
		}

		// Check targetNamespace for HelmRelease
		if targetNamespace, ok := spec["targetNamespace"].(string); ok {
			namespaceLower := strings.ToLower(targetNamespace)
			if strings.Contains(namespaceLower, "prod") || strings.Contains(namespaceLower, "production") || strings.Contains(namespaceLower, "live") {
				return true
			}
		}
	}

	return false
}

func (v *Validator) isValidInterval(interval string) bool {
	// Parse interval like "1m", "5m", "10m", "1h"
	// Valid range: 60s to 600s (1m to 10m)
	if strings.HasSuffix(interval, "s") {
		// Parse seconds
		var seconds int
		fmt.Sscanf(interval, "%ds", &seconds)
		return seconds >= 60 && seconds <= v.config.MaxIntervalSeconds
	} else if strings.HasSuffix(interval, "m") {
		// Parse minutes
		var minutes int
		fmt.Sscanf(interval, "%dm", &minutes)
		seconds := minutes * 60
		return seconds >= 60 && seconds <= v.config.MaxIntervalSeconds
	}
	return true // Unknown format, let it pass
}

func (v *Validator) isMutableVersion(version string) bool {
	// Check for version range patterns
	mutablePatterns := []string{"x", "*", "~", "^"}
	for _, pattern := range mutablePatterns {
		if strings.Contains(version, pattern) {
			return true
		}
	}
	return false
}
