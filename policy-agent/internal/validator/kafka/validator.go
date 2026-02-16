package kafka

import (
	"context"
	"fmt"
	"time"

	"policy-agent/internal/policy"
	"policy-agent/internal/validator"
	"policy-agent/pkg/errors"
	"policy-agent/pkg/types"
	"gopkg.in/yaml.v3"
)

// Validator implements the validator.Validator interface for Kafka resources
type Validator struct {
	engine *policy.Engine
	config *Config
}

// Config holds Kafka-specific configuration
type Config struct {
	MinReplicationFactor      int
	RequireMinInsyncReplicas  bool
	MinInsyncReplicasValue    int
	RequireCompression        bool
	AllowedCompressionTypes   []string
	MaxRetentionDays          int
	WarnRetentionDays         int
}

// NewValidator creates a new Kafka validator
func NewValidator(engine *policy.Engine, config *Config) (*Validator, error) {
	if engine == nil {
		return nil, errors.New(errors.ErrCodeConfigInvalid, "policy engine cannot be nil")
	}

	if config == nil {
		config = DefaultConfig()
	}

	return &Validator{
		engine: engine,
		config: config,
	}, nil
}

// DefaultConfig returns default Kafka validation configuration
func DefaultConfig() *Config {
	return &Config{
		MinReplicationFactor:     3,
		RequireMinInsyncReplicas: true,
		MinInsyncReplicasValue:   2,
		RequireCompression:       true,
		AllowedCompressionTypes:  []string{"lz4", "snappy", "zstd"},
		MaxRetentionDays:         90,
		WarnRetentionDays:        60,
	}
}

// Domain returns the domain name
func (v *Validator) Domain() string {
	return "kafka"
}

// SupportedTypes returns the resource types this validator handles
func (v *Validator) SupportedTypes() []string {
	return []string{
		"KafkaTopic",
		"KafkaConnector",
		"KafkaUser",
		"KafkaConnect",
	}
}

// CanValidate determines if this validator can handle the input
func (v *Validator) CanValidate(input *validator.Input) bool {
	// Check if resource type is supported
	for _, t := range v.SupportedTypes() {
		if input.ResourceType == t {
			return true
		}
	}

	// Try to parse and check the kind field
	var resource map[string]interface{}
	if err := yaml.Unmarshal(input.Raw, &resource); err == nil {
		if kind, ok := resource["kind"].(string); ok {
			for _, t := range v.SupportedTypes() {
				if kind == t {
					return true
				}
			}
		}
	}

	return false
}

// Validate validates Kafka configuration against policies
func (v *Validator) Validate(ctx context.Context, input *validator.Input) (*types.ValidationResult, error) {
	startTime := time.Now()

	// Parse the input based on format
	var resource map[string]interface{}
	switch input.Format {
	case types.FormatYAML:
		if err := yaml.Unmarshal(input.Raw, &resource); err != nil {
			return nil, errors.Wrap(err, errors.ErrCodeParseFailed, "failed to parse YAML").
				WithDomain(v.Domain())
		}
	case types.FormatJSON:
		// TODO: Add JSON parsing
		return nil, errors.New(errors.ErrCodeUnsupportedFormat, "JSON format not yet supported")
	default:
		return nil, errors.New(errors.ErrCodeUnsupportedFormat, fmt.Sprintf("unsupported format: %s", input.Format))
	}

	// Determine resource type
	kind, ok := resource["kind"].(string)
	if !ok {
		return nil, errors.New(errors.ErrCodeParseFailed, "missing or invalid 'kind' field").
			WithDomain(v.Domain())
	}

	// Get resource name
	resourceName := extractResourceName(resource)

	// Create result
	result := &types.ValidationResult{
		Domain:       v.Domain(),
		Resource:     resourceName,
		ResourceType: kind,
		File:         input.File,
		Timestamp:    time.Now(),
		Violations:   []types.Violation{},
		Warnings:     []types.Warning{},
		Passed:       []types.PolicyCheck{},
	}

	// Validate based on resource type
	switch kind {
	case "KafkaTopic":
		if err := v.validateTopic(ctx, resource, result); err != nil {
			return nil, err
		}
	case "KafkaConnector":
		// TODO: Implement connector validation
		result.Passed = append(result.Passed, types.PolicyCheck{
			Policy:  "kafka.connector",
			Message: "Connector validation not yet implemented",
		})
	default:
		return nil, errors.New(errors.ErrCodeUnsupportedFormat, fmt.Sprintf("unsupported Kafka resource type: %s", kind)).
			WithDomain(v.Domain())
	}

	// Set overall status
	if len(result.Violations) > 0 {
		result.Status = "failed"
	} else if len(result.Warnings) > 0 {
		result.Status = "warning"
	} else {
		result.Status = "passed"
	}

	result.Duration = time.Since(startTime)

	return result, nil
}

// validateTopic validates a Kafka topic against policies
func (v *Validator) validateTopic(ctx context.Context, resource map[string]interface{}, result *types.ValidationResult) error {
	policies := []string{
		"kafka.topics.replication",
		"kafka.topics.compression",
		"kafka.topics.retention",
	}

	for _, policyPackage := range policies {
		// Evaluate policy
		evalResult, err := v.engine.Evaluate(ctx, resource, policyPackage)
		if err != nil {
			// Policy not found or evaluation error - log but continue
			continue
		}

		// Extract denials (violations)
		denials := evalResult.GetDeny()
		for _, denial := range denials {
			violation := types.Violation{
				Policy:   policyPackage,
				Severity: v.getSeverityForPolicy(policyPackage),
				Message:  denial,
			}
			result.Violations = append(result.Violations, violation)
		}

		// Extract warnings
		warnings := evalResult.GetWarn()
		for _, warning := range warnings {
			result.Warnings = append(result.Warnings, types.Warning{
				Policy:   policyPackage,
				Severity: types.SeverityMedium,
				Message:  warning,
			})
		}

		// Extract recommendations
		recommendations := evalResult.GetRecommendations()
		for i, rec := range recommendations {
			if i < len(result.Violations) {
				// Add recommendation to the corresponding violation
				field, _ := rec["field"].(string)
				current := rec["current"]
				recommended := rec["recommended"]
				reason, _ := rec["reason"].(string)

				result.Violations[i].Field = field
				result.Violations[i].CurrentValue = current
				result.Violations[i].ExpectedValue = recommended
				result.Violations[i].Remediation = &types.Remediation{
					AutoFixable: true,
					Suggestion:  reason,
				}
			}
		}

		// If no violations or warnings, mark as passed
		if len(denials) == 0 && len(warnings) == 0 {
			result.Passed = append(result.Passed, types.PolicyCheck{
				Policy:  policyPackage,
				Message: "Policy check passed",
			})
		}
	}

	return nil
}

// getSeverityForPolicy returns the severity level for a given policy
func (v *Validator) getSeverityForPolicy(policy string) types.Severity {
	severityMap := map[string]types.Severity{
		"kafka.topics.replication": types.SeverityHigh,
		"kafka.topics.compression": types.SeverityMedium,
		"kafka.topics.retention":   types.SeverityMedium,
	}

	if severity, ok := severityMap[policy]; ok {
		return severity
	}

	return types.SeverityMedium
}

// extractResourceName extracts the resource name from the Kafka resource
func extractResourceName(resource map[string]interface{}) string {
	if metadata, ok := resource["metadata"].(map[string]interface{}); ok {
		if name, ok := metadata["name"].(string); ok {
			return name
		}
	}
	return "unknown"
}
