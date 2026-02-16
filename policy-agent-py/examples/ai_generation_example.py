#!/usr/bin/env python3
"""Example: Generate Kafka topic using Claude AI."""

import os
import sys

# Add parent directory to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from policy_agent.ai.claude import ClaudeClient
from policy_agent.types.result import GenerateRequest

def main():
    """Generate a Kafka topic configuration using AI."""

    # Check API key
    if not os.getenv("ANTHROPIC_API_KEY"):
        print("Error: ANTHROPIC_API_KEY environment variable not set")
        print("Please set it with: export ANTHROPIC_API_KEY='your-key-here'")
        return 1

    # Create AI client
    print("Initializing Claude AI client...")
    client = ClaudeClient(
        model="claude-sonnet-4-5-20250929",
        enable_cache=True,
    )

    # Create generation request
    request = GenerateRequest(
        domain="kafka",
        requirements="""Create a high-throughput user events topic with:
        - Suitable for 10,000 events/second
        - 30 day retention
        - High durability and availability
        - Optimized for storage costs""",
        policies=[
            "kafka.topics.replication",
            "kafka.topics.compression",
            "kafka.topics.retention",
        ],
    )

    # Generate configuration
    print("\nGenerating Kafka topic configuration...")
    print(f"Requirements: {request.requirements}")
    print(f"Policies: {', '.join(request.policies)}")
    print("\nCalling Claude AI API...\n")

    try:
        response = client.generate(request)

        print("=" * 80)
        print("GENERATED CONFIGURATION:")
        print("=" * 80)
        print(response.configuration)
        print("\n" + "=" * 80)
        print("EXPLANATION:")
        print("=" * 80)
        print(response.explanation)
        print("\n" + "=" * 80)
        print(f"Policies Met: {', '.join(response.policies_met)}")
        print("=" * 80)

        # Save to file
        output_file = "generated-topic.yaml"
        with open(output_file, "w") as f:
            f.write(response.configuration)

        print(f"\nConfiguration saved to: {output_file}")
        print("\nYou can now validate it with:")
        print(f"  policy-agent validate --file {output_file}")

        return 0

    except Exception as e:
        print(f"\nError: {e}")
        return 1


if __name__ == "__main__":
    sys.exit(main())
