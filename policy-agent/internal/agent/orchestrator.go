package agent

import (
	"context"
	"fmt"
	"time"

	"policy-agent/internal/ai"
	"policy-agent/internal/policy"
	"policy-agent/internal/validator"
	"policy-agent/pkg/errors"
	"policy-agent/pkg/types"
)

// Orchestrator coordinates validation across multiple domains
type Orchestrator struct {
	registry validator.Registry
	engine   *policy.Engine
	aiClient ai.Client
}

// New creates a new orchestrator
func New(registry validator.Registry, engine *policy.Engine, aiClient ai.Client) *Orchestrator {
	return &Orchestrator{
		registry: registry,
		engine:   engine,
		aiClient: aiClient,
	}
}

// Validate performs validation on the given input
func (o *Orchestrator) Validate(ctx context.Context, req *ValidateRequest) (*ValidateResponse, error) {
	startTime := time.Now()

	// Determine which validator to use
	var v validator.Validator
	var err error

	if req.Domain != "" {
		// Use domain-specific validator
		v, err = o.registry.Get(req.Domain)
		if err != nil {
			return nil, err
		}
	} else if req.ResourceType != "" {
		// Use resource type to determine validator
		v, err = o.registry.GetForResourceType(req.ResourceType)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New(errors.ErrCodeConfigInvalid, "either domain or resource_type must be specified")
	}

	// Prepare validator input
	input := &validator.Input{
		Raw:          req.Raw,
		Format:       req.Format,
		ResourceType: req.ResourceType,
		Metadata:     req.Metadata,
		File:         req.File,
	}

	// Validate
	result, err := v.Validate(ctx, input)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeValidationFailed, "validation failed")
	}

	result.Duration = time.Since(startTime)

	// If auto-fix is requested and there are violations, attempt remediation
	if req.AutoFix && len(result.Violations) > 0 && o.aiClient != nil {
		fixReq := &ai.RemediateRequest{
			Domain:         result.Domain,
			OriginalConfig: string(req.Raw),
			Violations:     result.Violations,
		}

		fixResp, err := o.aiClient.Remediate(ctx, fixReq)
		if err == nil && fixResp != nil {
			// Add AI remediation suggestions to violations
			for i := range result.Violations {
				if i < len(fixResp.Suggestions) {
					result.Violations[i].Remediation = &types.Remediation{
						AutoFixable:   fixResp.Suggestions[i].AutoFixable,
						Suggestion:    fixResp.Suggestions[i].Suggestion,
						AIExplanation: fixResp.Suggestions[i].Explanation,
					}
				}
			}
		}
	}

	return &ValidateResponse{
		Result: result,
	}, nil
}

// Generate generates a policy-compliant configuration using AI
func (o *Orchestrator) Generate(ctx context.Context, req *types.GenerateRequest) (*types.GenerateResponse, error) {
	if o.aiClient == nil {
		return nil, errors.New(errors.ErrCodeConfigInvalid, "AI client not configured")
	}

	// Convert to AI request
	aiReq := &ai.GenerateRequest{
		Domain:       req.Domain,
		Requirements: req.Requirements,
		Context:      req.Context,
		Policies:     req.Policies,
	}

	// Call AI to generate configuration
	resp, err := o.aiClient.Generate(ctx, aiReq)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeAIRequestFailed, "AI generation failed")
	}

	return &types.GenerateResponse{
		Configuration: resp.Configuration,
		Format:        resp.Format,
		Explanation:   resp.Explanation,
		PoliciesMet:   resp.PoliciesMet,
	}, nil
}

// Fix attempts to fix policy violations in a configuration
func (o *Orchestrator) Fix(ctx context.Context, req *types.FixRequest) (*types.FixResponse, error) {
	if o.aiClient == nil {
		return nil, errors.New(errors.ErrCodeConfigInvalid, "AI client not configured")
	}

	// First, validate to find violations
	validateReq := &ValidateRequest{
		Raw:    []byte(req.File),
		Format: types.FormatYAML, // TODO: Detect format
	}

	validateResp, err := o.Validate(ctx, validateReq)
	if err != nil {
		return nil, err
	}

	// If no violations, nothing to fix
	if len(validateResp.Result.Violations) == 0 {
		return &types.FixResponse{
			Original: req.File,
			Fixed:    req.File,
			Changes:  []types.Change{},
			Explanation: "No policy violations found",
		}, nil
	}

	// Use AI to remediate
	remediateReq := &ai.RemediateRequest{
		Domain:         validateResp.Result.Domain,
		OriginalConfig: req.File,
		Violations:     validateResp.Result.Violations,
	}

	remediateResp, err := o.aiClient.Remediate(ctx, remediateReq)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeAIRequestFailed, "remediation failed")
	}

	// Convert AI suggestions to changes
	changes := make([]types.Change, len(remediateResp.Suggestions))
	for i, suggestion := range remediateResp.Suggestions {
		changes[i] = types.Change{
			Field:    validateResp.Result.Violations[i].Field,
			OldValue: validateResp.Result.Violations[i].CurrentValue,
			NewValue: validateResp.Result.Violations[i].ExpectedValue,
			Reason:   suggestion.Suggestion,
		}
	}

	return &types.FixResponse{
		Original:    req.File,
		Fixed:       remediateResp.FixedConfig,
		Changes:     changes,
		Explanation: remediateResp.Explanation,
	}, nil
}

// ValidateRequest represents a validation request
type ValidateRequest struct {
	Raw          []byte
	Format       types.Format
	Domain       string
	ResourceType string
	Metadata     map[string]interface{}
	File         string
	AutoFix      bool
}

// ValidateResponse represents a validation response
type ValidateResponse struct {
	Result *types.ValidationResult
}

// GetAvailableDomains returns all available validation domains
func (o *Orchestrator) GetAvailableDomains() []string {
	return o.registry.Domains()
}

// GetPolicyPackages returns all loaded policy packages
func (o *Orchestrator) GetPolicyPackages() []string {
	return o.engine.GetPolicyPackages()
}
