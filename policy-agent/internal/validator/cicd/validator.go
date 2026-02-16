package cicd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"policy-agent/internal/policy"
	"policy-agent/internal/validator"
	"policy-agent/pkg/types"
)

// Config holds configuration for CI/CD validator
type Config struct {
	RequireSecurityScanning bool
	RequireTesting          bool
	AllowedRegistries       []string
	MaxJobTimeout           int // minutes
	RequireApproval         bool
}

// DefaultConfig returns default CI/CD validation configuration
func DefaultConfig() *Config {
	return &Config{
		RequireSecurityScanning: true,
		RequireTesting:          true,
		AllowedRegistries:       []string{"docker.io", "ghcr.io", "gcr.io"},
		MaxJobTimeout:           60,
		RequireApproval:         true,
	}
}

// Validator validates CI/CD pipeline configurations
type Validator struct {
	engine *policy.Engine
	config *Config
}

// NewValidator creates a new CI/CD validator
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
	return "cicd"
}

// SupportedTypes returns supported CI/CD pipeline types
func (v *Validator) SupportedTypes() []string {
	return []string{
		"GitHubWorkflow",
		"GitLabCI",
		"JenkinsFile",
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

// Validate validates CI/CD configuration
func (v *Validator) Validate(ctx context.Context, input *validator.Input) (*types.ValidationResult, error) {
	var config map[string]interface{}
	if err := json.Unmarshal(input.Raw, &config); err != nil {
		return nil, fmt.Errorf("failed to parse CI/CD config: %w", err)
	}

	result := &types.ValidationResult{
		Domain:       v.Domain(),
		ResourceType: input.ResourceType,
		Status:       types.StatusPassed,
	}

	// Extract resource name
	if name, ok := config["name"].(string); ok {
		result.Resource = name
	} else {
		result.Resource = "pipeline"
	}

	// Validate based on type
	switch input.ResourceType {
	case "GitHubWorkflow":
		v.validateGitHubWorkflow(ctx, config, result)
	case "GitLabCI":
		v.validateGitLabCI(ctx, config, result)
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

// validateGitHubWorkflow validates GitHub Actions workflow
func (v *Validator) validateGitHubWorkflow(ctx context.Context, config map[string]interface{}, result *types.ValidationResult) {
	// Check workflow name
	if _, ok := config["name"]; !ok {
		result.Violations = append(result.Violations, types.Violation{
			Policy:   "cicd.github.workflows",
			Severity: types.SeverityMedium,
			Message:  "Workflow must have a descriptive name",
			Field:    "name",
		})
	}

	// Check jobs
	jobs, ok := config["jobs"].(map[string]interface{})
	if !ok || len(jobs) == 0 {
		result.Violations = append(result.Violations, types.Violation{
			Policy:   "cicd.github.workflows",
			Severity: types.SeverityHigh,
			Message:  "Workflow must define at least one job",
			Field:    "jobs",
		})
		return
	}

	hasTestJob := false
	for jobName, jobData := range jobs {
		jobMap, ok := jobData.(map[string]interface{})
		if !ok {
			continue
		}

		v.validateGitHubJob(jobName, jobMap, result)

		// Check for test jobs
		if strings.Contains(strings.ToLower(jobName), "test") {
			hasTestJob = true
		}
	}

	// Require test job
	if v.config.RequireTesting && !hasTestJob {
		result.Violations = append(result.Violations, types.Violation{
			Policy:   "cicd.github.workflows",
			Severity: types.SeverityHigh,
			Message:  "Workflow should include a test job to validate code quality",
		})
	}

	// Check security scanning
	if v.config.RequireSecurityScanning && !v.hasSecurityScanning(jobs) {
		result.Warnings = append(result.Warnings, types.Warning{
			Policy:   "cicd.github.security",
			Severity: types.SeverityMedium,
			Message:  "Workflow should include security scanning (CodeQL, dependency scanning, or SAST)",
		})
	}
}

// validateGitHubJob validates individual GitHub Actions job
func (v *Validator) validateGitHubJob(jobName string, job map[string]interface{}, result *types.ValidationResult) {
	// Check timeout
	timeout, hasTimeout := job["timeout-minutes"]
	if hasTimeout {
		if timeoutNum, ok := timeout.(float64); ok {
			if int(timeoutNum) > v.config.MaxJobTimeout {
				result.Violations = append(result.Violations, types.Violation{
					Policy:        "cicd.github.workflows",
					Severity:      types.SeverityMedium,
					Message:       fmt.Sprintf("Job '%s' timeout of %d minutes exceeds maximum %d minutes", jobName, int(timeoutNum), v.config.MaxJobTimeout),
					Field:         fmt.Sprintf("jobs.%s.timeout-minutes", jobName),
					CurrentValue:  int(timeoutNum),
					ExpectedValue: v.config.MaxJobTimeout,
				})
			}
		}
	} else {
		result.Warnings = append(result.Warnings, types.Warning{
			Policy:   "cicd.github.workflows",
			Severity: types.SeverityLow,
			Message:  fmt.Sprintf("Job '%s' should set timeout-minutes to prevent hanging builds", jobName),
			Field:    fmt.Sprintf("jobs.%s.timeout-minutes", jobName),
		})
	}

	// Check steps for security issues
	if steps, ok := job["steps"].([]interface{}); ok {
		v.validateGitHubSteps(jobName, steps, result)
	}

	// Check permissions
	if permissions, ok := job["permissions"].(map[string]interface{}); ok {
		v.validatePermissions(jobName, permissions, result)
	}
}

// validateGitHubSteps validates workflow steps
func (v *Validator) validateGitHubSteps(jobName string, steps []interface{}, result *types.ValidationResult) {
	for i, stepData := range steps {
		step, ok := stepData.(map[string]interface{})
		if !ok {
			continue
		}

		// Check for unpinned actions
		if uses, ok := step["uses"].(string); ok {
			if !v.isActionPinned(uses) && !v.isOfficialAction(uses) {
				result.Violations = append(result.Violations, types.Violation{
					Policy:   "cicd.github.security",
					Severity: types.SeverityHigh,
					Message:  fmt.Sprintf("Job '%s' uses unpinned third-party action '%s' - pin to commit SHA for security", jobName, uses),
					Field:    fmt.Sprintf("jobs.%s.steps[%d].uses", jobName, i),
					Remediation: &types.Remediation{
						AutoFixable: false,
						Suggestion:  "Pin actions to commit SHA (e.g., action@a1b2c3d4...)",
					},
				})
			}
		}

		// Check for secret exposure
		if run, ok := step["run"].(string); ok {
			if v.mayExposeSecrets(run) {
				result.Violations = append(result.Violations, types.Violation{
					Policy:   "cicd.github.security",
					Severity: types.SeverityCritical,
					Message:  fmt.Sprintf("Job '%s' may expose secrets in logs - avoid echoing secret values", jobName),
					Field:    fmt.Sprintf("jobs.%s.steps[%d].run", jobName, i),
				})
			}
		}
	}
}

// validatePermissions validates workflow permissions
func (v *Validator) validatePermissions(jobName string, permissions map[string]interface{}, result *types.ValidationResult) {
	hasWrite := false
	for perm, value := range permissions {
		if valueStr, ok := value.(string); ok {
			if valueStr == "write" {
				hasWrite = true
			}
		}
	}

	if hasWrite {
		result.Warnings = append(result.Warnings, types.Warning{
			Policy:   "cicd.github.security",
			Severity: types.SeverityMedium,
			Message:  fmt.Sprintf("Job '%s' has write permissions - use minimal required permissions", jobName),
		})
	}
}

// validateGitLabCI validates GitLab CI configuration
func (v *Validator) validateGitLabCI(ctx context.Context, config map[string]interface{}, result *types.ValidationResult) {
	hasTestStage := false

	// Iterate through jobs
	for jobName, jobData := range config {
		// Skip special keys
		if v.isGitLabSpecialKey(jobName) {
			continue
		}

		jobMap, ok := jobData.(map[string]interface{})
		if !ok {
			continue
		}

		v.validateGitLabJob(jobName, jobMap, result)

		// Check for test stage
		if stage, ok := jobMap["stage"].(string); ok {
			if strings.Contains(strings.ToLower(stage), "test") {
				hasTestStage = true
			}
		}
	}

	// Require test stage
	if v.config.RequireTesting && !hasTestStage {
		result.Violations = append(result.Violations, types.Violation{
			Policy:   "cicd.gitlab.security",
			Severity: types.SeverityHigh,
			Message:  "Pipeline should include testing stage to validate code quality",
		})
	}
}

// validateGitLabJob validates individual GitLab CI job
func (v *Validator) validateGitLabJob(jobName string, job map[string]interface{}, result *types.ValidationResult) {
	// Check for privileged Docker
	if services, ok := job["services"].([]interface{}); ok {
		for _, service := range services {
			if serviceMap, ok := service.(map[string]interface{}); ok {
				if command, ok := serviceMap["command"].([]interface{}); ok {
					for _, cmd := range command {
						if cmdStr, ok := cmd.(string); ok {
							if strings.Contains(cmdStr, "--privileged") {
								result.Violations = append(result.Violations, types.Violation{
									Policy:   "cicd.gitlab.security",
									Severity: types.SeverityCritical,
									Message:  fmt.Sprintf("Job '%s' uses privileged Docker mode - security risk", jobName),
									Field:    fmt.Sprintf("%s.services", jobName),
								})
							}
						}
					}
				}
			}
		}
	}

	// Check image tag
	if image, ok := job["image"].(string); ok {
		if strings.HasSuffix(image, ":latest") {
			result.Violations = append(result.Violations, types.Violation{
				Policy:        "cicd.gitlab.security",
				Severity:      types.SeverityMedium,
				Message:       fmt.Sprintf("Job '%s' uses ':latest' Docker image tag - pin to specific version", jobName),
				Field:         fmt.Sprintf("%s.image", jobName),
				CurrentValue:  image,
				ExpectedValue: "specific version tag",
			})
		}
	}

	// Check for production deployment without manual approval
	if environment, ok := job["environment"].(string); ok {
		if strings.Contains(strings.ToLower(environment), "prod") {
			if when, ok := job["when"].(string); !ok || when != "manual" {
				result.Violations = append(result.Violations, types.Violation{
					Policy:   "cicd.gitlab.security",
					Severity: types.SeverityHigh,
					Message:  fmt.Sprintf("Production deployment job '%s' should require manual approval", jobName),
					Field:    fmt.Sprintf("%s.when", jobName),
				})
			}
		}
	}

	// Check artifacts expiration
	if artifacts, ok := job["artifacts"].(map[string]interface{}); ok {
		if _, hasExpire := artifacts["expire_in"]; !hasExpire {
			result.Warnings = append(result.Warnings, types.Warning{
				Policy:   "cicd.gitlab.security",
				Severity: types.SeverityLow,
				Message:  fmt.Sprintf("Job '%s' artifacts should have expire_in set to manage storage", jobName),
				Field:    fmt.Sprintf("%s.artifacts.expire_in", jobName),
			})
		}
	}
}

// Helper functions

func (v *Validator) hasSecurityScanning(jobs map[string]interface{}) bool {
	for _, jobData := range jobs {
		jobMap, ok := jobData.(map[string]interface{})
		if !ok {
			continue
		}

		steps, ok := jobMap["steps"].([]interface{})
		if !ok {
			continue
		}

		for _, stepData := range steps {
			step, ok := stepData.(map[string]interface{})
			if !ok {
				continue
			}

			if uses, ok := step["uses"].(string); ok {
				securityActions := []string{"github/codeql-action", "aquasecurity/trivy", "snyk/actions"}
				for _, action := range securityActions {
					if strings.Contains(uses, action) {
						return true
					}
				}
			}
		}
	}
	return false
}

func (v *Validator) isActionPinned(action string) bool {
	// Check if action is pinned to commit SHA (40 hex characters)
	parts := strings.Split(action, "@")
	if len(parts) != 2 {
		return false
	}

	ref := parts[1]
	// SHA is 40 hex characters
	if len(ref) == 40 {
		for _, c := range ref {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				return false
			}
		}
		return true
	}
	return false
}

func (v *Validator) isOfficialAction(action string) bool {
	officialPrefixes := []string{"actions/", "github/"}
	for _, prefix := range officialPrefixes {
		if strings.HasPrefix(action, prefix) {
			return true
		}
	}
	return false
}

func (v *Validator) mayExposeSecrets(script string) bool {
	// Check for potential secret exposure patterns
	if strings.Contains(script, "echo") && strings.Contains(script, "secrets.") {
		return true
	}
	if strings.Contains(script, "echo") && strings.Contains(script, "${{") {
		return true
	}
	return false
}

func (v *Validator) isGitLabSpecialKey(key string) bool {
	specialKeys := []string{"stages", "variables", "default", "workflow", "include"}
	for _, special := range specialKeys {
		if key == special {
			return true
		}
	}
	return false
}
