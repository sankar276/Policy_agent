package iac

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"policy-agent/internal/policy"
	"policy-agent/internal/validator"
	"policy-agent/pkg/types"
)

// Config holds configuration for IaC validator
type Config struct {
	RequiredTags         []string
	RequireVersionPin    bool
	RequireRemoteState   bool
	RequireEncryption    bool
	AllowedEnvironments  []string
	RequireNamingConv    bool
}

// DefaultConfig returns default IaC validation configuration
func DefaultConfig() *Config {
	return &Config{
		RequiredTags:        []string{"Environment", "Owner", "Project", "ManagedBy"},
		RequireVersionPin:   true,
		RequireRemoteState:  true,
		RequireEncryption:   true,
		AllowedEnvironments: []string{"dev", "staging", "prod"},
		RequireNamingConv:   true,
	}
}

// Validator validates Infrastructure as Code configurations
type Validator struct {
	engine *policy.Engine
	config *Config
}

// NewValidator creates a new IaC validator
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
	return "iac"
}

// SupportedTypes returns the list of supported IaC resource types
func (v *Validator) SupportedTypes() []string {
	return []string{
		"TerraformConfig",
		"CloudFormation",
		"Pulumi",
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

// Validate validates IaC configuration
func (v *Validator) Validate(ctx context.Context, input *validator.Input) (*types.ValidationResult, error) {
	// Parse input
	var config map[string]interface{}
	if err := json.Unmarshal(input.Raw, &config); err != nil {
		return nil, fmt.Errorf("failed to parse IaC config: %w", err)
	}

	// Create result
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
	case "TerraformConfig":
		v.validateTerraform(ctx, config, result)
	case "CloudFormation":
		v.validateCloudFormation(ctx, config, result)
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

// validateTerraform validates Terraform configuration
func (v *Validator) validateTerraform(ctx context.Context, config map[string]interface{}, result *types.ValidationResult) {
	// Check provider configuration
	v.checkProviderConfig(config, result)

	// Check state backend configuration
	v.checkStateBackend(config, result)

	// Check resources
	v.checkResources(config, result)

	// Run OPA policies if engine is available
	if v.engine != nil {
		v.runOPAPolicies(ctx, config, result)
	}
}

// checkProviderConfig validates provider configuration
func (v *Validator) checkProviderConfig(config map[string]interface{}, result *types.ValidationResult) {
	terraform, ok := config["terraform"].(map[string]interface{})
	if !ok {
		result.Violations = append(result.Violations, types.Violation{
			Policy:   "iac.terraform.provider",
			Severity: types.SeverityHigh,
			Message:  "Missing terraform configuration block",
			Field:    "terraform",
		})
		return
	}

	// Check required_version
	if _, ok := terraform["required_version"]; !ok {
		result.Violations = append(result.Violations, types.Violation{
			Policy:   "iac.terraform.provider",
			Severity: types.SeverityMedium,
			Message:  "Missing terraform.required_version - should specify minimum Terraform version",
			Field:    "terraform.required_version",
			Remediation: &types.Remediation{
				AutoFixable: true,
				Suggestion:  "Pin Terraform version for reproducible deployments",
			},
		})
	}

	// Check provider versions
	if v.config.RequireVersionPin {
		providers, ok := terraform["required_providers"].(map[string]interface{})
		if !ok || len(providers) == 0 {
			result.Violations = append(result.Violations, types.Violation{
				Policy:   "iac.terraform.provider",
				Severity: types.SeverityHigh,
				Message:  "Missing terraform.required_providers configuration",
				Field:    "terraform.required_providers",
			})
			return
		}

		for name, provider := range providers {
			providerMap, ok := provider.(map[string]interface{})
			if !ok {
				continue
			}

			if _, ok := providerMap["version"]; !ok {
				result.Violations = append(result.Violations, types.Violation{
					Policy:   "iac.terraform.provider",
					Severity: types.SeverityHigh,
					Message:  fmt.Sprintf("Provider '%s' missing version constraint", name),
					Field:    fmt.Sprintf("terraform.required_providers.%s.version", name),
					Remediation: &types.Remediation{
						AutoFixable: true,
						Suggestion:  "Pin provider versions to ensure reproducible deployments",
					},
				})
			}
		}
	}

	result.Passed = append(result.Passed, types.PolicyCheck{
		Policy:  "iac.terraform.provider.configured",
		Message: "Provider configuration present",
	})
}

// checkStateBackend validates state backend configuration
func (v *Validator) checkStateBackend(config map[string]interface{}, result *types.ValidationResult) {
	if !v.config.RequireRemoteState {
		return
	}

	terraform, ok := config["terraform"].(map[string]interface{})
	if !ok {
		return
	}

	backend, ok := terraform["backend"].(map[string]interface{})
	if !ok || len(backend) == 0 {
		result.Violations = append(result.Violations, types.Violation{
			Policy:   "iac.terraform.state",
			Severity: types.SeverityHigh,
			Message:  "No backend configuration found - configure remote state backend",
			Field:    "terraform.backend",
			Remediation: &types.Remediation{
				AutoFixable: false,
				Suggestion:  "Use remote backend (s3, azurerm, gcs) for production state management",
			},
		})
		return
	}

	// Check for local backend
	if _, ok := backend["local"]; ok {
		result.Violations = append(result.Violations, types.Violation{
			Policy:   "iac.terraform.state",
			Severity: types.SeverityHigh,
			Message:  "Local backend detected - use remote backend (s3, azurerm, gcs) for production",
			Field:    "terraform.backend.local",
		})
		return
	}

	// Check S3 backend encryption
	if s3, ok := backend["s3"].(map[string]interface{}); ok {
		if v.config.RequireEncryption {
			encrypt, ok := s3["encrypt"].(bool)
			if !ok || !encrypt {
				result.Violations = append(result.Violations, types.Violation{
					Policy:        "iac.terraform.state",
					Severity:      types.SeverityHigh,
					Message:       "S3 backend missing encryption - set 'encrypt = true'",
					Field:         "terraform.backend.s3.encrypt",
					CurrentValue:  encrypt,
					ExpectedValue: true,
					Remediation: &types.Remediation{
						AutoFixable: true,
						Suggestion:  "Enable S3 encryption to protect state file contents",
					},
				})
			}
		}

		// Check for DynamoDB locking
		if _, ok := s3["dynamodb_table"]; !ok {
			result.Warnings = append(result.Warnings, types.Warning{
				Policy:   "iac.terraform.state",
				Severity: types.SeverityMedium,
				Message:  "S3 backend missing DynamoDB table for state locking",
				Field:    "terraform.backend.s3.dynamodb_table",
			})
		} else {
			result.Passed = append(result.Passed, types.PolicyCheck{
				Policy:  "iac.terraform.state.locking",
				Message: "State locking configured with DynamoDB",
			})
		}
	}
}

// checkResources validates resource configurations
func (v *Validator) checkResources(config map[string]interface{}, result *types.ValidationResult) {
	resources, ok := config["resource"].(map[string]interface{})
	if !ok {
		return
	}

	for resourceType, resourcesOfType := range resources {
		resourceMap, ok := resourcesOfType.(map[string]interface{})
		if !ok {
			continue
		}

		for name, resource := range resourceMap {
			resourceData, ok := resource.(map[string]interface{})
			if !ok {
				continue
			}

			v.checkResourceTags(resourceType, name, resourceData, result)
			v.checkResourceSecurity(resourceType, name, resourceData, result)
		}
	}
}

// checkResourceTags validates resource tagging
func (v *Validator) checkResourceTags(resourceType, name string, resource map[string]interface{}, result *types.ValidationResult) {
	// List of AWS resources that support tags
	taggableResources := map[string]bool{
		"aws_instance":         true,
		"aws_s3_bucket":        true,
		"aws_rds_instance":     true,
		"aws_vpc":              true,
		"aws_security_group":   true,
		"aws_lambda_function":  true,
		"azurerm_virtual_machine": true,
		"google_compute_instance": true,
	}

	if !taggableResources[resourceType] {
		return
	}

	tags, hasTags := resource["tags"].(map[string]interface{})
	labels, hasLabels := resource["labels"].(map[string]interface{})

	if !hasTags && !hasLabels {
		result.Violations = append(result.Violations, types.Violation{
			Policy:   "iac.terraform.resources",
			Severity: types.SeverityMedium,
			Message:  fmt.Sprintf("Resource '%s.%s' missing tags", resourceType, name),
			Field:    fmt.Sprintf("resource.%s.%s.tags", resourceType, name),
		})
		return
	}

	// Check required tags
	tagMap := tags
	if tagMap == nil {
		tagMap = labels
	}

	for _, requiredTag := range v.config.RequiredTags {
		if _, ok := tagMap[requiredTag]; !ok {
			result.Violations = append(result.Violations, types.Violation{
				Policy:   "iac.terraform.resources",
				Severity: types.SeverityMedium,
				Message:  fmt.Sprintf("Resource '%s.%s' missing required tag '%s'", resourceType, name, requiredTag),
				Field:    fmt.Sprintf("resource.%s.%s.tags.%s", resourceType, name, requiredTag),
			})
		}
	}
}

// checkResourceSecurity validates security configurations
func (v *Validator) checkResourceSecurity(resourceType, name string, resource map[string]interface{}, result *types.ValidationResult) {
	switch resourceType {
	case "aws_s3_bucket":
		v.checkS3Security(name, resource, result)
	case "aws_rds_instance":
		v.checkRDSSecurity(name, resource, result)
	case "aws_security_group":
		v.checkSecurityGroupRules(name, resource, result)
	}
}

// checkS3Security validates S3 bucket security
func (v *Validator) checkS3Security(name string, bucket map[string]interface{}, result *types.ValidationResult) {
	// Check for public access
	if acl, ok := bucket["acl"].(string); ok {
		if strings.Contains(acl, "public") {
			result.Violations = append(result.Violations, types.Violation{
				Policy:   "iac.terraform.security",
				Severity: types.SeverityCritical,
				Message:  fmt.Sprintf("S3 bucket '%s' has public ACL - should be private", name),
				Field:    fmt.Sprintf("resource.aws_s3_bucket.%s.acl", name),
				CurrentValue: acl,
				ExpectedValue: "private",
			})
		}
	}

	// Check encryption
	if v.config.RequireEncryption {
		if _, ok := bucket["server_side_encryption_configuration"]; !ok {
			result.Violations = append(result.Violations, types.Violation{
				Policy:   "iac.terraform.security",
				Severity: types.SeverityHigh,
				Message:  fmt.Sprintf("S3 bucket '%s' missing server-side encryption", name),
				Field:    fmt.Sprintf("resource.aws_s3_bucket.%s.server_side_encryption_configuration", name),
			})
		}
	}
}

// checkRDSSecurity validates RDS security
func (v *Validator) checkRDSSecurity(name string, rds map[string]interface{}, result *types.ValidationResult) {
	// Check public accessibility
	if publicAccess, ok := rds["publicly_accessible"].(bool); ok && publicAccess {
		result.Violations = append(result.Violations, types.Violation{
			Policy:   "iac.terraform.security",
			Severity: types.SeverityCritical,
			Message:  fmt.Sprintf("RDS instance '%s' is publicly accessible", name),
			Field:    fmt.Sprintf("resource.aws_rds_instance.%s.publicly_accessible", name),
			CurrentValue: true,
			ExpectedValue: false,
		})
	}

	// Check encryption
	if v.config.RequireEncryption {
		if encrypted, ok := rds["storage_encrypted"].(bool); !ok || !encrypted {
			result.Violations = append(result.Violations, types.Violation{
				Policy:   "iac.terraform.security",
				Severity: types.SeverityHigh,
				Message:  fmt.Sprintf("RDS instance '%s' does not have storage encryption enabled", name),
				Field:    fmt.Sprintf("resource.aws_rds_instance.%s.storage_encrypted", name),
			})
		}
	}

	// Check backup retention
	if retention, ok := rds["backup_retention_period"].(float64); ok {
		if retention < 7 {
			result.Violations = append(result.Violations, types.Violation{
				Policy:   "iac.terraform.security",
				Severity: types.SeverityMedium,
				Message:  fmt.Sprintf("RDS instance '%s' has backup retention period %.0f days - should be at least 7 days", name, retention),
				Field:    fmt.Sprintf("resource.aws_rds_instance.%s.backup_retention_period", name),
				CurrentValue: retention,
				ExpectedValue: 7,
			})
		}
	}
}

// checkSecurityGroupRules validates security group rules
func (v *Validator) checkSecurityGroupRules(name string, sg map[string]interface{}, result *types.ValidationResult) {
	// Check ingress rules
	if ingress, ok := sg["ingress"].([]interface{}); ok {
		for i, rule := range ingress {
			ruleMap, ok := rule.(map[string]interface{})
			if !ok {
				continue
			}

			if cidrBlocks, ok := ruleMap["cidr_blocks"].([]interface{}); ok {
				for _, cidr := range cidrBlocks {
					if cidr == "0.0.0.0/0" {
						fromPort, _ := ruleMap["from_port"].(float64)
						if fromPort != 80 && fromPort != 443 {
							result.Violations = append(result.Violations, types.Violation{
								Policy:   "iac.terraform.security",
								Severity: types.SeverityHigh,
								Message:  fmt.Sprintf("Security group '%s' has overly permissive rule allowing 0.0.0.0/0 on port %.0f", name, fromPort),
								Field:    fmt.Sprintf("resource.aws_security_group.%s.ingress[%d].cidr_blocks", name, i),
							})
						}
					}
				}
			}
		}
	}
}

// runOPAPolicies evaluates OPA policies
func (v *Validator) runOPAPolicies(ctx context.Context, config map[string]interface{}, result *types.ValidationResult) {
	// This would integrate with the OPA engine
	// For now, it's a placeholder for future OPA integration
}

// validateCloudFormation validates CloudFormation templates
func (v *Validator) validateCloudFormation(ctx context.Context, config map[string]interface{}, result *types.ValidationResult) {
	// Placeholder for CloudFormation validation
	result.Warnings = append(result.Warnings, types.Warning{
		Policy:   "iac.cloudformation",
		Severity: types.SeverityLow,
		Message:  "CloudFormation validation not yet implemented",
	})
}
