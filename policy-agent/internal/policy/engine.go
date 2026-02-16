package policy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/rego"
	"github.com/policy-agent/policy-agent/pkg/errors"
)

// Engine manages OPA policy evaluation
type Engine struct {
	policyPath string
	policies   map[string]*rego.PreparedEvalQuery
	modules    map[string]*ast.Module
	mu         sync.RWMutex
	loader     *Loader
}

// NewEngine creates a new policy engine
func NewEngine(policyPath string) (*Engine, error) {
	if policyPath == "" {
		return nil, errors.New(errors.ErrCodeConfigInvalid, "policy path cannot be empty")
	}

	return &Engine{
		policyPath: policyPath,
		policies:   make(map[string]*rego.PreparedEvalQuery),
		modules:    make(map[string]*ast.Module),
		loader:     NewLoader(policyPath),
	}, nil
}

// LoadPolicies loads all Rego policies from the policy path
func (e *Engine) LoadPolicies(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Load all .rego files
	modules, err := e.loader.LoadAll(ctx)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodePolicyLoadFailed, "failed to load policies")
	}

	// Compile and prepare each module
	for path, module := range modules {
		query, err := e.prepareQuery(ctx, module)
		if err != nil {
			return errors.Wrap(err, errors.ErrCodePolicyLoadFailed, fmt.Sprintf("failed to prepare policy: %s", path))
		}

		e.modules[path] = module
		e.policies[path] = query
	}

	return nil
}

// Evaluate evaluates input against a specific policy
func (e *Engine) Evaluate(ctx context.Context, input interface{}, policyPackage string) (*EvalResult, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Find the policy query for this package
	var query *rego.PreparedEvalQuery
	for path, q := range e.policies {
		if containsPackage(e.modules[path], policyPackage) {
			query = q
			break
		}
	}

	if query == nil {
		return nil, errors.New(errors.ErrCodePolicyLoadFailed, fmt.Sprintf("policy not found: %s", policyPackage))
	}

	// Execute the query
	results, err := query.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeValidationFailed, "policy evaluation failed")
	}

	return &EvalResult{
		Package: policyPackage,
		Results: results,
	}, nil
}

// prepareQuery compiles and prepares a Rego module for evaluation
func (e *Engine) prepareQuery(ctx context.Context, module *ast.Module) (*rego.PreparedEvalQuery, error) {
	// Create a new Rego instance with the module
	r := rego.New(
		rego.Module(module.Package.Path.String(), module.String()),
		rego.Query("data"),
	)

	// Compile and prepare
	query, err := r.PrepareForEval(ctx)
	if err != nil {
		return nil, err
	}

	return &query, nil
}

// Reload reloads all policies from disk
func (e *Engine) Reload(ctx context.Context) error {
	return e.LoadPolicies(ctx)
}

// GetPolicyPackages returns all loaded policy packages
func (e *Engine) GetPolicyPackages() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	packages := make([]string, 0)
	for _, module := range e.modules {
		packages = append(packages, module.Package.Path.String())
	}

	return packages
}

// containsPackage checks if a module contains a specific package
func containsPackage(module *ast.Module, packagePath string) bool {
	return module.Package.Path.String() == packagePath
}

// EvalResult represents the result of a policy evaluation
type EvalResult struct {
	Package string
	Results rego.ResultSet
}

// GetDeny extracts denial messages from evaluation results
func (r *EvalResult) GetDeny() []string {
	var denials []string

	for _, result := range r.Results {
		if deny, ok := result.Bindings["deny"]; ok {
			if denySlice, ok := deny.([]interface{}); ok {
				for _, d := range denySlice {
					if msg, ok := d.(string); ok {
						denials = append(denials, msg)
					}
				}
			}
		}
	}

	return denials
}

// GetWarn extracts warning messages from evaluation results
func (r *EvalResult) GetWarn() []string {
	var warnings []string

	for _, result := range r.Results {
		if warn, ok := result.Bindings["warn"]; ok {
			if warnSlice, ok := warn.([]interface{}); ok {
				for _, w := range warnSlice {
					if msg, ok := w.(string); ok {
						warnings = append(warnings, msg)
					}
				}
			}
		}
	}

	return warnings
}

// GetRecommendations extracts recommendations from evaluation results
func (r *EvalResult) GetRecommendations() []map[string]interface{} {
	var recommendations []map[string]interface{}

	for _, result := range r.Results {
		if rec, ok := result.Bindings["recommend"]; ok {
			if recSlice, ok := rec.([]interface{}); ok {
				for _, r := range recSlice {
					if recMap, ok := r.(map[string]interface{}); ok {
						recommendations = append(recommendations, recMap)
					}
				}
			}
		}
	}

	return recommendations
}

// Loader handles loading Rego policy files from disk
type Loader struct {
	policyPath string
}

// NewLoader creates a new policy loader
func NewLoader(policyPath string) *Loader {
	return &Loader{
		policyPath: policyPath,
	}
}

// LoadAll loads all .rego files from the policy directory
func (l *Loader) LoadAll(ctx context.Context) (map[string]*ast.Module, error) {
	modules := make(map[string]*ast.Module)

	err := filepath.Walk(l.policyPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-.rego files
		if info.IsDir() || filepath.Ext(path) != ".rego" {
			return nil
		}

		// Skip test files
		if filepath.Base(path)[:5] == "test_" {
			return nil
		}

		// Read and parse the module
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read policy file %s: %w", path, err)
		}

		module, err := ast.ParseModule(path, string(content))
		if err != nil {
			return fmt.Errorf("failed to parse policy file %s: %w", path, err)
		}

		modules[path] = module
		return nil
	})

	if err != nil {
		return nil, err
	}

	return modules, nil
}
