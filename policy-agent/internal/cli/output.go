package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"policy-agent/pkg/types"
	"gopkg.in/yaml.v3"
)

// OutputFormat represents the output format
type OutputFormat string

const (
	FormatText OutputFormat = "text"
	FormatJSON OutputFormat = "json"
	FormatYAML OutputFormat = "yaml"
)

// Formatter formats validation results
type Formatter struct {
	format OutputFormat
	color  bool
}

// NewFormatter creates a new output formatter
func NewFormatter(format string, enableColor bool) *Formatter {
	return &Formatter{
		format: OutputFormat(format),
		color:  enableColor,
	}
}

// FormatResult formats a validation result
func (f *Formatter) FormatResult(result *types.ValidationResult) (string, error) {
	switch f.format {
	case FormatJSON:
		return f.formatJSON(result)
	case FormatYAML:
		return f.formatYAML(result)
	default:
		return f.formatText(result), nil
	}
}

// formatJSON formats result as JSON
func (f *Formatter) formatJSON(result *types.ValidationResult) (string, error) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return string(data), nil
}

// formatYAML formats result as YAML
func (f *Formatter) formatYAML(result *types.ValidationResult) (string, error) {
	data, err := yaml.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal YAML: %w", err)
	}
	return string(data), nil
}

// formatText formats result as human-readable text
func (f *Formatter) formatText(result *types.ValidationResult) string {
	var output strings.Builder

	// Header
	output.WriteString("\n")
	output.WriteString(f.separator("=", 70))
	output.WriteString(f.bold(fmt.Sprintf("Domain: %s (%s)\n", result.Domain, result.ResourceType)))
	output.WriteString(fmt.Sprintf("Resource: %s\n", result.Resource))

	// Status with color
	statusText := fmt.Sprintf("Status: %s", result.Status)
	switch result.Status {
	case types.StatusPassed:
		output.WriteString(f.green(statusText) + "\n")
	case types.StatusFailed:
		output.WriteString(f.red(statusText) + "\n")
	case types.StatusWarning:
		output.WriteString(f.yellow(statusText) + "\n")
	default:
		output.WriteString(statusText + "\n")
	}

	if result.Duration > 0 {
		output.WriteString(fmt.Sprintf("Duration: %v\n", result.Duration))
	}
	if result.File != "" {
		output.WriteString(fmt.Sprintf("File: %s\n", result.File))
	}
	output.WriteString(f.separator("=", 70))

	// Summary
	summary := fmt.Sprintf("\n📊 Summary: %d violations, %d warnings, %d passed\n",
		len(result.Violations), len(result.Warnings), len(result.Passed))
	output.WriteString(f.bold(summary))

	// Violations
	if len(result.Violations) > 0 {
		output.WriteString("\n" + f.red("❌ VIOLATIONS:") + "\n")
		for i, v := range result.Violations {
			output.WriteString(f.formatViolation(i+1, v))
		}
	}

	// Warnings
	if len(result.Warnings) > 0 {
		output.WriteString("\n" + f.yellow("⚠️  WARNINGS:") + "\n")
		for i, w := range result.Warnings {
			output.WriteString(f.formatWarning(i+1, w))
		}
	}

	// Passed checks
	if len(result.Passed) > 0 {
		output.WriteString("\n" + f.green("✅ PASSED:") + "\n")
		for _, p := range result.Passed {
			output.WriteString(fmt.Sprintf("   • %s", p.Policy))
			if p.Message != "" {
				output.WriteString(fmt.Sprintf(": %s", p.Message))
			}
			output.WriteString("\n")
		}
	}

	output.WriteString("\n")
	return output.String()
}

// formatViolation formats a single violation
func (f *Formatter) formatViolation(index int, v types.Violation) string {
	var output strings.Builder

	// Header
	header := fmt.Sprintf("\n%d. [%s] %s\n", index, v.Severity, v.Policy)
	output.WriteString(f.bold(header))

	// Message
	output.WriteString(fmt.Sprintf("   → %s\n", v.Message))

	// Field details
	if v.Field != "" {
		output.WriteString(f.dim(fmt.Sprintf("   Field: %s\n", v.Field)))
		if v.CurrentValue != nil {
			output.WriteString(f.dim(fmt.Sprintf("   Current: %v\n", v.CurrentValue)))
		}
		if v.ExpectedValue != nil {
			output.WriteString(f.dim(fmt.Sprintf("   Expected: %v\n", v.ExpectedValue)))
		}
	}

	// Location
	if v.Line > 0 {
		location := fmt.Sprintf("   Location: line %d", v.Line)
		if v.Column > 0 {
			location += fmt.Sprintf(", column %d", v.Column)
		}
		output.WriteString(f.dim(location + "\n"))
	}

	// Remediation
	if v.Remediation != nil {
		if v.Remediation.AutoFixable {
			output.WriteString(f.green("   🔧 Auto-fixable\n"))
		}
		if v.Remediation.Suggestion != "" {
			output.WriteString(fmt.Sprintf("   💡 Suggestion: %s\n", v.Remediation.Suggestion))
		}
		if v.Remediation.Example != "" {
			output.WriteString("   📝 Example:\n")
			for _, line := range strings.Split(v.Remediation.Example, "\n") {
				output.WriteString(f.dim(fmt.Sprintf("      %s\n", line)))
			}
		}
	}

	return output.String()
}

// formatWarning formats a single warning
func (f *Formatter) formatWarning(index int, w types.Warning) string {
	var output strings.Builder

	header := fmt.Sprintf("\n%d. [%s] %s\n", index, w.Severity, w.Policy)
	output.WriteString(f.bold(header))
	output.WriteString(fmt.Sprintf("   → %s\n", w.Message))

	if w.Field != "" {
		output.WriteString(f.dim(fmt.Sprintf("   Field: %s\n", w.Field)))
	}

	return output.String()
}

// Color and formatting helpers

func (f *Formatter) bold(s string) string {
	if !f.color {
		return s
	}
	return "\033[1m" + s + "\033[0m"
}

func (f *Formatter) dim(s string) string {
	if !f.color {
		return s
	}
	return "\033[2m" + s + "\033[0m"
}

func (f *Formatter) red(s string) string {
	if !f.color {
		return s
	}
	return "\033[31m" + s + "\033[0m"
}

func (f *Formatter) green(s string) string {
	if !f.color {
		return s
	}
	return "\033[32m" + s + "\033[0m"
}

func (f *Formatter) yellow(s string) string {
	if !f.color {
		return s
	}
	return "\033[33m" + s + "\033[0m"
}

func (f *Formatter) separator(char string, length int) string {
	return strings.Repeat(char, length) + "\n"
}

// FormatBatchResults formats multiple validation results
func (f *Formatter) FormatBatchResults(results []*types.ValidationResult) (string, error) {
	if f.format == FormatJSON {
		data, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal JSON: %w", err)
		}
		return string(data), nil
	}

	if f.format == FormatYAML {
		data, err := yaml.Marshal(results)
		if err != nil {
			return "", fmt.Errorf("failed to marshal YAML: %w", err)
		}
		return string(data), nil
	}

	// Text format - show summary first, then individual results
	var output strings.Builder

	// Overall summary
	totalViolations := 0
	totalWarnings := 0
	totalPassed := 0
	failed := 0
	passed := 0

	for _, result := range results {
		totalViolations += len(result.Violations)
		totalWarnings += len(result.Warnings)
		totalPassed += len(result.Passed)

		if result.Status == types.StatusFailed {
			failed++
		} else if result.Status == types.StatusPassed {
			passed++
		}
	}

	output.WriteString("\n" + f.bold(f.separator("=", 70)))
	output.WriteString(f.bold(fmt.Sprintf("BATCH VALIDATION RESULTS (%d files)\n", len(results))))
	output.WriteString(f.bold(f.separator("=", 70)))
	output.WriteString(fmt.Sprintf("\n📊 Overall Summary:\n"))
	output.WriteString(fmt.Sprintf("   Files: %d total\n", len(results)))
	output.WriteString(f.green(fmt.Sprintf("   Passed: %d\n", passed)))
	output.WriteString(f.red(fmt.Sprintf("   Failed: %d\n", failed)))
	output.WriteString(fmt.Sprintf("   Violations: %d\n", totalViolations))
	output.WriteString(fmt.Sprintf("   Warnings: %d\n", totalWarnings))
	output.WriteString(fmt.Sprintf("   Checks passed: %d\n\n", totalPassed))

	// Individual results
	for _, result := range results {
		output.WriteString(f.FormatResult(result))
	}

	return output.String(), nil
}
