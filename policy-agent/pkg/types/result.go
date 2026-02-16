package types

import "time"

// Format represents the configuration file format
type Format string

const (
	FormatYAML Format = "yaml"
	FormatJSON Format = "json"
	FormatHCL  Format = "hcl"
	FormatTOML Format = "toml"
)

// Severity represents the severity level of a violation
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
	SeverityInfo     Severity = "info"
)

// ValidationResult represents the result of validating a configuration
type ValidationResult struct {
	Status        string          `json:"status"` // "passed", "failed", "warning"
	Domain        string          `json:"domain"`
	Resource      string          `json:"resource"`
	ResourceType  string          `json:"resource_type"`
	File          string          `json:"file,omitempty"`
	Violations    []Violation     `json:"violations,omitempty"`
	Warnings      []Warning       `json:"warnings,omitempty"`
	Passed        []PolicyCheck   `json:"passed,omitempty"`
	Timestamp     time.Time       `json:"timestamp"`
	Duration      time.Duration   `json:"duration"`
}

// Violation represents a policy violation
type Violation struct {
	Policy       string        `json:"policy"`
	Severity     Severity      `json:"severity"`
	Message      string        `json:"message"`
	Field        string        `json:"field,omitempty"`
	CurrentValue interface{}   `json:"current_value,omitempty"`
	ExpectedValue interface{}  `json:"expected_value,omitempty"`
	Remediation  *Remediation  `json:"remediation,omitempty"`
}

// Warning represents a policy warning (non-blocking)
type Warning struct {
	Policy   string      `json:"policy"`
	Severity Severity    `json:"severity"`
	Message  string      `json:"message"`
	Field    string      `json:"field,omitempty"`
}

// PolicyCheck represents a passed policy check
type PolicyCheck struct {
	Policy  string `json:"policy"`
	Message string `json:"message,omitempty"`
}

// Remediation contains suggestions for fixing a violation
type Remediation struct {
	AutoFixable    bool   `json:"auto_fixable"`
	Suggestion     string `json:"suggestion"`
	Diff           string `json:"diff,omitempty"`
	AIExplanation  string `json:"ai_explanation,omitempty"`
}

// ValidationSummary aggregates results across multiple files
type ValidationSummary struct {
	TotalFiles   int       `json:"total_files"`
	Validated    int       `json:"validated"`
	Violations   int       `json:"violations"`
	Warnings     int       `json:"warnings"`
	Passed       int       `json:"passed"`
	Duration     time.Duration `json:"duration"`
}

// ValidateRequest represents a request to validate configuration
type ValidateRequest struct {
	Files       []string          `json:"files"`
	Directory   string            `json:"directory,omitempty"`
	Domain      string            `json:"domain,omitempty"`
	Policies    []string          `json:"policies,omitempty"`
	AutoFix     bool              `json:"auto_fix"`
	Format      Format            `json:"format"`
}

// GenerateRequest represents a request to generate configuration
type GenerateRequest struct {
	Domain       string            `json:"domain"`
	Requirements string            `json:"requirements"`
	Output       string            `json:"output,omitempty"`
	Interactive  bool              `json:"interactive"`
	Policies     []string          `json:"policies,omitempty"`
	Context      map[string]interface{} `json:"context,omitempty"`
}

// GenerateResponse represents the response from generation
type GenerateResponse struct {
	Configuration string            `json:"configuration"`
	Format        Format            `json:"format"`
	Explanation   string            `json:"explanation"`
	PoliciesMet   []string          `json:"policies_met"`
}

// FixRequest represents a request to fix policy violations
type FixRequest struct {
	File        string   `json:"file"`
	Interactive bool     `json:"interactive"`
	Auto        bool     `json:"auto"`
	DryRun      bool     `json:"dry_run"`
}

// FixResponse represents the response from fixing violations
type FixResponse struct {
	Original    string            `json:"original"`
	Fixed       string            `json:"fixed"`
	Changes     []Change          `json:"changes"`
	Explanation string            `json:"explanation"`
}

// Change represents a single change made during fixing
type Change struct {
	Field       string      `json:"field"`
	OldValue    interface{} `json:"old_value"`
	NewValue    interface{} `json:"new_value"`
	Reason      string      `json:"reason"`
}
