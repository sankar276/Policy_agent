package validator

import (
	"context"

	"policy-agent/pkg/types"
)

// Validator defines the interface that all domain validators must implement
type Validator interface {
	// Validate checks if input conforms to policies
	Validate(ctx context.Context, input *Input) (*types.ValidationResult, error)

	// Domain returns the domain name (kafka, kubernetes, etc.)
	Domain() string

	// SupportedTypes returns resource types this validator handles
	// Example: ["KafkaTopic", "KafkaConnector"] for Kafka validator
	SupportedTypes() []string

	// CanValidate determines if this validator can handle the given input
	CanValidate(input *Input) bool
}

// Input represents the input to be validated
type Input struct {
	// Raw contains the raw configuration bytes
	Raw []byte

	// Format specifies the configuration format
	Format types.Format

	// ResourceType is the type of resource being validated
	// Examples: "KafkaTopic", "Deployment", "terraform"
	ResourceType string

	// Metadata contains additional contextual information
	Metadata map[string]interface{}

	// File is the source file path (optional)
	File string
}

// Registry manages available validators
type Registry interface {
	// Register adds a validator to the registry
	Register(validator Validator) error

	// Get returns a validator for the given domain
	Get(domain string) (Validator, error)

	// GetForResourceType returns a validator that can handle the resource type
	GetForResourceType(resourceType string) (Validator, error)

	// List returns all registered validators
	List() []Validator

	// Domains returns all registered domain names
	Domains() []string
}
