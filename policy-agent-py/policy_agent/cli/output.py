"""Output formatting for CLI."""

import json
from typing import List
from datetime import datetime

import click
import yaml

from policy_agent.types.result import ValidationResult, Severity


class OutputFormatter:
    """Formatter for validation results."""

    def __init__(self, format_type: str = "text", color: bool = True):
        """Initialize formatter.

        Args:
            format_type: Output format (text, json, yaml)
            color: Enable colored output
        """
        self.format_type = format_type
        self.color = color

    def format_result(self, result: ValidationResult) -> str:
        """Format a single validation result.

        Args:
            result: Validation result to format

        Returns:
            Formatted output string
        """
        if self.format_type == "json":
            return self._format_json(result)
        elif self.format_type == "yaml":
            return self._format_yaml(result)
        else:
            return self._format_text(result)

    def format_batch_results(self, results: List[ValidationResult]) -> str:
        """Format multiple validation results.

        Args:
            results: List of validation results

        Returns:
            Formatted output string
        """
        if self.format_type == "json":
            return self._format_json_batch(results)
        elif self.format_type == "yaml":
            return self._format_yaml_batch(results)
        else:
            return self._format_text_batch(results)

    def _format_text(self, result: ValidationResult) -> str:
        """Format result as human-readable text."""
        output = []

        # Header
        output.append("\n" + "=" * 70)
        output.append(self._bold(f"Domain: {result.domain} ({result.resource_type})"))
        output.append(f"Resource: {result.resource}")

        # Status with color
        status_text = f"Status: {result.status}"
        if result.status == "passed":
            output.append(self._green(status_text))
        elif result.status == "failed":
            output.append(self._red(status_text))
        else:
            output.append(self._yellow(status_text))

        if result.duration:
            output.append(f"Duration: {result.duration:.3f}s")
        if result.file:
            output.append(f"File: {result.file}")

        output.append("=" * 70)

        # Summary
        summary = (
            f"\n📊 Summary: {len(result.violations)} violations, "
            f"{len(result.warnings)} warnings, {len(result.passed)} passed"
        )
        output.append(self._bold(summary))

        # Violations
        if result.violations:
            output.append("\n" + self._red("❌ VIOLATIONS:"))
            for i, v in enumerate(result.violations, 1):
                output.append(self._format_violation(i, v))

        # Warnings
        if result.warnings:
            output.append("\n" + self._yellow("⚠️  WARNINGS:"))
            for i, w in enumerate(result.warnings, 1):
                output.append(self._format_warning(i, w))

        # Passed checks
        if result.passed:
            output.append("\n" + self._green("✅ PASSED:"))
            for p in result.passed:
                msg = f"   • {p.policy}"
                if p.message:
                    msg += f": {p.message}"
                output.append(msg)

        output.append("")
        return "\n".join(output)

    def _format_violation(self, index: int, violation) -> str:
        """Format a single violation."""
        lines = []

        # Header
        severity_color = {
            Severity.CRITICAL: self._red,
            Severity.HIGH: self._red,
            Severity.MEDIUM: self._yellow,
            Severity.LOW: self._dim,
        }.get(violation.severity, self._dim)

        header = f"\n{index}. [{violation.severity.value.upper()}] {violation.policy}"
        lines.append(self._bold(severity_color(header)))

        # Message
        lines.append(f"   → {violation.message}")

        # Field details
        if violation.field:
            lines.append(self._dim(f"   Field: {violation.field}"))
            if violation.current_value is not None:
                lines.append(self._dim(f"   Current: {violation.current_value}"))
            if violation.expected_value is not None:
                lines.append(self._dim(f"   Expected: {violation.expected_value}"))

        # Location
        if violation.line and violation.line > 0:
            location = f"   Location: line {violation.line}"
            if violation.column and violation.column > 0:
                location += f", column {violation.column}"
            lines.append(self._dim(location))

        # Remediation
        if violation.remediation:
            if violation.remediation.auto_fixable:
                lines.append(self._green("   🔧 Auto-fixable"))
            if violation.remediation.suggestion:
                lines.append(f"   💡 Suggestion: {violation.remediation.suggestion}")
            if violation.remediation.example:
                lines.append("   📝 Example:")
                for line in violation.remediation.example.split("\n"):
                    lines.append(self._dim(f"      {line}"))

        return "\n".join(lines)

    def _format_warning(self, index: int, warning) -> str:
        """Format a single warning."""
        lines = []

        header = f"\n{index}. [{warning.severity.value.upper()}] {warning.policy}"
        lines.append(self._bold(header))
        lines.append(f"   → {warning.message}")

        if warning.field:
            lines.append(self._dim(f"   Field: {warning.field}"))

        return "\n".join(lines)

    def _format_text_batch(self, results: List[ValidationResult]) -> str:
        """Format batch results as text."""
        output = []

        # Overall summary
        total_violations = sum(len(r.violations) for r in results)
        total_warnings = sum(len(r.warnings) for r in results)
        total_passed = sum(len(r.passed) for r in results)
        failed = sum(1 for r in results if r.status == "failed")
        passed = sum(1 for r in results if r.status == "passed")

        output.append("\n" + self._bold("=" * 70))
        output.append(self._bold(f"BATCH VALIDATION RESULTS ({len(results)} files)"))
        output.append(self._bold("=" * 70))
        output.append(f"\n📊 Overall Summary:")
        output.append(f"   Files: {len(results)} total")
        output.append(self._green(f"   Passed: {passed}"))
        output.append(self._red(f"   Failed: {failed}"))
        output.append(f"   Violations: {total_violations}")
        output.append(f"   Warnings: {total_warnings}")
        output.append(f"   Checks passed: {total_passed}\n")

        # Individual results
        for result in results:
            output.append(self._format_text(result))

        return "\n".join(output)

    def _format_json(self, result: ValidationResult) -> str:
        """Format result as JSON."""
        output = self._result_to_dict(result)
        return json.dumps(output, indent=2, default=str)

    def _format_json_batch(self, results: List[ValidationResult]) -> str:
        """Format batch results as JSON."""
        output = [self._result_to_dict(r) for r in results]
        return json.dumps(output, indent=2, default=str)

    def _format_yaml(self, result: ValidationResult) -> str:
        """Format result as YAML."""
        output = self._result_to_dict(result)
        return yaml.dump(output, default_flow_style=False, sort_keys=False)

    def _format_yaml_batch(self, results: List[ValidationResult]) -> str:
        """Format batch results as YAML."""
        output = [self._result_to_dict(r) for r in results]
        return yaml.dump(output, default_flow_style=False, sort_keys=False)

    def _result_to_dict(self, result: ValidationResult) -> dict:
        """Convert ValidationResult to dictionary."""
        return {
            "status": result.status,
            "domain": result.domain,
            "resource": result.resource,
            "resource_type": result.resource_type,
            "file": result.file,
            "violations": [
                {
                    "policy": v.policy,
                    "severity": v.severity.value,
                    "message": v.message,
                    "field": v.field,
                    "current_value": v.current_value,
                    "expected_value": v.expected_value,
                    "line": v.line,
                    "column": v.column,
                    "remediation": (
                        {
                            "auto_fixable": v.remediation.auto_fixable,
                            "suggestion": v.remediation.suggestion,
                            "example": v.remediation.example,
                        }
                        if v.remediation
                        else None
                    ),
                }
                for v in result.violations
            ],
            "warnings": [
                {
                    "policy": w.policy,
                    "severity": w.severity.value,
                    "message": w.message,
                    "field": w.field,
                }
                for w in result.warnings
            ],
            "passed": [{"policy": p.policy, "message": p.message} for p in result.passed],
            "timestamp": result.timestamp.isoformat() if result.timestamp else None,
            "duration": result.duration,
        }

    # Color helpers
    def _bold(self, text: str) -> str:
        """Make text bold."""
        if not self.color:
            return text
        return click.style(text, bold=True)

    def _dim(self, text: str) -> str:
        """Make text dim."""
        if not self.color:
            return text
        return click.style(text, dim=True)

    def _red(self, text: str) -> str:
        """Make text red."""
        if not self.color:
            return text
        return click.style(text, fg="red")

    def _green(self, text: str) -> str:
        """Make text green."""
        if not self.color:
            return text
        return click.style(text, fg="green")

    def _yellow(self, text: str) -> str:
        """Make text yellow."""
        if not self.color:
            return text
        return click.style(text, fg="yellow")
