#!/usr/bin/env python3
"""Quick test script to verify policy-agent installation."""

import sys


def test_imports():
    """Test that all modules can be imported."""
    print("Testing imports...")

    try:
        from policy_agent.types.result import ValidationResult, Violation, Severity

        print("  ✓ Types module")
    except ImportError as e:
        print(f"  ✗ Types module: {e}")
        return False

    try:
        from policy_agent.validators.base import Validator, ValidatorRegistry

        print("  ✓ Validators base")
    except ImportError as e:
        print(f"  ✗ Validators base: {e}")
        return False

    try:
        from policy_agent.validators.kafka import KafkaValidator, KafkaConfig

        print("  ✓ Kafka validator")
    except ImportError as e:
        print(f"  ✗ Kafka validator: {e}")
        return False

    try:
        from policy_agent.agent import Orchestrator

        print("  ✓ Orchestrator")
    except ImportError as e:
        print(f"  ✗ Orchestrator: {e}")
        return False

    try:
        from policy_agent.ai.client import AIClient
        from policy_agent.ai.claude import ClaudeClient
        from policy_agent.ai.prompts import get_prompt_template

        print("  ✓ AI integration")
    except ImportError as e:
        print(f"  ✗ AI integration: {e}")
        return False

    try:
        from policy_agent.cli.main import cli

        print("  ✓ CLI")
    except ImportError as e:
        print(f"  ✗ CLI: {e}")
        return False

    return True


def test_kafka_validator():
    """Test basic Kafka validator functionality."""
    print("\nTesting Kafka validator...")

    from policy_agent.validators.kafka import KafkaValidator

    validator = KafkaValidator()

    # Test with invalid topic
    invalid_topic = {
        "kind": "KafkaTopic",
        "metadata": {"name": "test-topic"},
        "spec": {
            "replicas": 1,  # Too low
            "partitions": 3,
            "config": {
                "retention.ms": "604800000"
                # Missing compression and min.insync.replicas
            },
        },
    }

    result = validator.validate(invalid_topic)

    if result.status == "failed" and len(result.violations) > 0:
        print(f"  ✓ Validation detected {len(result.violations)} violations")
        return True
    else:
        print(f"  ✗ Expected violations, got: {result.status}")
        return False


def test_cli_available():
    """Test that CLI is available."""
    print("\nTesting CLI availability...")

    import subprocess

    try:
        result = subprocess.run(
            ["policy-agent", "--version"],
            capture_output=True,
            text=True,
            timeout=5,
        )

        if result.returncode == 0:
            print(f"  ✓ CLI available: {result.stdout.strip()}")
            return True
        else:
            print(f"  ✗ CLI returned error code {result.returncode}")
            return False

    except FileNotFoundError:
        print("  ✗ CLI not found in PATH")
        print("     Run: pip install -e .")
        return False
    except Exception as e:
        print(f"  ✗ Error running CLI: {e}")
        return False


def main():
    """Run all tests."""
    print("=" * 60)
    print("Policy Agent - Installation Test")
    print("=" * 60)

    tests = [
        ("Module imports", test_imports),
        ("Kafka validator", test_kafka_validator),
        ("CLI availability", test_cli_available),
    ]

    results = []
    for name, test_func in tests:
        try:
            passed = test_func()
            results.append((name, passed))
        except Exception as e:
            print(f"\n✗ {name} failed with exception: {e}")
            results.append((name, False))

    # Summary
    print("\n" + "=" * 60)
    print("TEST SUMMARY")
    print("=" * 60)

    all_passed = True
    for name, passed in results:
        status = "✓ PASS" if passed else "✗ FAIL"
        print(f"{status}: {name}")
        if not passed:
            all_passed = False

    print("=" * 60)

    if all_passed:
        print("\n🎉 All tests passed! Installation successful.")
        print("\nNext steps:")
        print("  1. Set ANTHROPIC_API_KEY for AI features")
        print("  2. Try: policy-agent validate --file examples/kafka/valid-topic.yaml")
        print("  3. See README.md for more examples")
        return 0
    else:
        print("\n⚠️  Some tests failed. Please check the errors above.")
        print("\nTry:")
        print("  pip install -e .")
        return 1


if __name__ == "__main__":
    sys.exit(main())
