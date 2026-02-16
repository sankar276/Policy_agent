package ai

import (
	"context"

	"policy-agent/pkg/types"
)

// Client defines the interface for AI interactions
type Client interface {
	// Generate creates a policy-compliant configuration
	Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error)

	// Remediate suggests fixes for policy violations
	Remediate(ctx context.Context, req *RemediateRequest) (*RemediateResponse, error)

	// Explain provides explanations for policies or violations
	Explain(ctx context.Context, req *ExplainRequest) (*ExplainResponse, error)
}

// GenerateRequest represents a request to generate configuration
type GenerateRequest struct {
	Domain       string
	Requirements string
	Policies     []string
	Context      map[string]interface{}
}

// GenerateResponse represents the response from generation
type GenerateResponse struct {
	Configuration string
	Format        types.Format
	Explanation   string
	PoliciesMet   []string
}

// RemediateRequest represents a request for remediation
type RemediateRequest struct {
	Domain         string
	OriginalConfig string
	Violations     []types.Violation
	Policies       []string
}

// RemediateResponse represents the response from remediation
type RemediateResponse struct {
	FixedConfig  string
	Suggestions  []Suggestion
	Explanation  string
}

// Suggestion represents a single fix suggestion
type Suggestion struct {
	Field        string
	CurrentValue interface{}
	NewValue     interface{}
	Suggestion   string
	Explanation  string
	AutoFixable  bool
}

// ExplainRequest represents a request for explanation
type ExplainRequest struct {
	Policy    string
	Violation *types.Violation
	Context   map[string]interface{}
}

// ExplainResponse represents the response from explanation
type ExplainResponse struct {
	Explanation string
	Examples    []string
}
