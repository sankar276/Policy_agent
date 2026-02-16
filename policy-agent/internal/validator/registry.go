package validator

import (
	"fmt"
	"sync"

	"github.com/policy-agent/policy-agent/pkg/errors"
)

// DefaultRegistry is the default validator registry
type DefaultRegistry struct {
	mu         sync.RWMutex
	validators map[string]Validator
	typeMap    map[string]Validator
}

// NewRegistry creates a new validator registry
func NewRegistry() Registry {
	return &DefaultRegistry{
		validators: make(map[string]Validator),
		typeMap:    make(map[string]Validator),
	}
}

// Register adds a validator to the registry
func (r *DefaultRegistry) Register(validator Validator) error {
	if validator == nil {
		return errors.New(errors.ErrCodeConfigInvalid, "validator cannot be nil")
	}

	domain := validator.Domain()
	if domain == "" {
		return errors.New(errors.ErrCodeConfigInvalid, "validator domain cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if domain already registered
	if _, exists := r.validators[domain]; exists {
		return errors.New(errors.ErrCodeConfigInvalid, fmt.Sprintf("validator for domain %s already registered", domain))
	}

	// Register by domain
	r.validators[domain] = validator

	// Register by resource types
	for _, resourceType := range validator.SupportedTypes() {
		r.typeMap[resourceType] = validator
	}

	return nil
}

// Get returns a validator for the given domain
func (r *DefaultRegistry) Get(domain string) (Validator, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	validator, exists := r.validators[domain]
	if !exists {
		return nil, errors.New(errors.ErrCodeUnknownDomain, fmt.Sprintf("no validator found for domain: %s", domain)).
			WithDomain(domain)
	}

	return validator, nil
}

// GetForResourceType returns a validator that can handle the resource type
func (r *DefaultRegistry) GetForResourceType(resourceType string) (Validator, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	validator, exists := r.typeMap[resourceType]
	if !exists {
		return nil, errors.New(errors.ErrCodeUnknownDomain, fmt.Sprintf("no validator found for resource type: %s", resourceType))
	}

	return validator, nil
}

// List returns all registered validators
func (r *DefaultRegistry) List() []Validator {
	r.mu.RLock()
	defer r.mu.RUnlock()

	validators := make([]Validator, 0, len(r.validators))
	for _, v := range r.validators {
		validators = append(validators, v)
	}

	return validators
}

// Domains returns all registered domain names
func (r *DefaultRegistry) Domains() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	domains := make([]string, 0, len(r.validators))
	for domain := range r.validators {
		domains = append(domains, domain)
	}

	return domains
}
