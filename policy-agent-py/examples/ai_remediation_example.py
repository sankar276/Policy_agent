#!/usr/bin/env python3
"""Example: Fix Kafka topic violations using Claude AI."""

import os
import sys

# Add parent directory to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

import yaml

from policy_agent.ai.claude import ClaudeClient
from policy_agent.validators.kafka import KafkaValidator

def main():
    """Fix violations in an invalid Kafka topic using AI."""

    # Check API key
    if not os.getenv("ANTHROPIC_API_KEY"):
        print("Error: ANTHROPIC_API_KEY environment variable not set")
        print("Please set it with: export ANTHROPIC_API_KEY='your-key-here'")
        return 1

    # Load invalid topic configuration
    invalid_topic_path = "../../examples/kafka/invalid-topic.yaml"

    if not os.path.exists(invalid_topic_path):
        # Create a simple invalid topic for demo
        invalid_config = """apiVersion: kafka.strimzi.io/v1beta2
kind: KafkaTopic
metadata:
  name: test-topic
  labels:
    app: test-app
spec:
  replicas: 1  # Too low!
  partitions: 3
  config:
    retention.ms: "604800000"  # 7 days
    # Missing compression and min.insync.replicas!
"""
        with open("invalid-topic.yaml", "w") as f:
            f.write(invalid_config)
        invalid_topic_path = "invalid-topic.yaml"

    with open(invalid_topic_path) as f:
        content = f.read()
        data = yaml.safe_load(content)

    print("=" * 80)
    print("ORIGINAL CONFIGURATION (with violations):")
    print("=" * 80)
    print(content)

    # Validate to find violations
    print("\n" + "=" * 80)
    print("VALIDATING CONFIGURATION:")
    print("=" * 80)

    validator = KafkaValidator()
    result = validator.validate(data)

    print(f"Status: {result.status}")
    print(f"Violations found: {len(result.violations)}")

    if result.violations:
        print("\nViolations:")
        for i, v in enumerate(result.violations, 1):
            print(f"\n{i}. [{v.severity.value}] {v.policy}")
            print(f"   {v.message}")
            if v.field:
                print(f"   Field: {v.field}")
                print(f"   Current: {v.current_value}")
                print(f"   Expected: {v.expected_value}")

    # Use AI to fix
    print("\n" + "=" * 80)
    print("FIXING WITH CLAUDE AI:")
    print("=" * 80)
    print("Calling Claude API to remediate violations...\n")

    try:
        client = ClaudeClient()
        fix_response = client.remediate("kafka", content, result.violations)

        print("=" * 80)
        print("FIXED CONFIGURATION:")
        print("=" * 80)
        print(fix_response.fixed)

        print("\n" + "=" * 80)
        print("CHANGES MADE:")
        print("=" * 80)
        for change in fix_response.changes:
            print(f"\nField: {change['field']}")
            print(f"  From: {change['from']}")
            print(f"  To: {change['to']}")
            print(f"  Reason: {change['reason']}")

        print("\n" + "=" * 80)
        print("EXPLANATION:")
        print("=" * 80)
        print(fix_response.explanation)

        # Save fixed configuration
        output_file = "fixed-topic.yaml"
        with open(output_file, "w") as f:
            f.write(fix_response.fixed)

        print("\n" + "=" * 80)
        print(f"Fixed configuration saved to: {output_file}")
        print("\nYou can now validate it with:")
        print(f"  policy-agent validate --file {output_file}")

        # Re-validate fixed config
        print("\n" + "=" * 80)
        print("RE-VALIDATING FIXED CONFIGURATION:")
        print("=" * 80)

        fixed_data = yaml.safe_load(fix_response.fixed)
        fixed_result = validator.validate(fixed_data)

        print(f"Status: {fixed_result.status}")
        print(f"Violations: {len(fixed_result.violations)}")
        print(f"Warnings: {len(fixed_result.warnings)}")
        print(f"Passed checks: {len(fixed_result.passed)}")

        if fixed_result.status == "passed":
            print("\n✅ SUCCESS! All violations fixed.")
        else:
            print("\n⚠️  Some issues remain:")
            for v in fixed_result.violations:
                print(f"  - {v.message}")

        return 0

    except Exception as e:
        print(f"\nError: {e}")
        import traceback
        traceback.print_exc()
        return 1


if __name__ == "__main__":
    sys.exit(main())
