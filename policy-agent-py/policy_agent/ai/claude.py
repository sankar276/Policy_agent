"""Claude AI client implementation."""

import os
import time
from datetime import datetime, timedelta
from typing import Any, Dict, List, Optional

from anthropic import Anthropic

from policy_agent.ai.client import AIClient
from policy_agent.ai.prompts import (
    build_explain_prompt,
    build_generate_prompt,
    build_remediate_prompt,
    format_violations,
)
from policy_agent.types.result import (
    FixResponse,
    Format,
    GenerateRequest,
    GenerateResponse,
    Violation,
)


class ResponseCache:
    """Simple in-memory cache for AI responses."""

    def __init__(self, ttl: timedelta = timedelta(hours=1)):
        """Initialize cache.

        Args:
            ttl: Time-to-live for cache entries
        """
        self.cache: Dict[str, Dict[str, Any]] = {}
        self.ttl = ttl

    def get(self, key: str) -> Optional[Any]:
        """Get cached value.

        Args:
            key: Cache key

        Returns:
            Cached value or None if not found/expired
        """
        if key not in self.cache:
            return None

        entry = self.cache[key]
        if datetime.now() > entry["expires_at"]:
            del self.cache[key]
            return None

        return entry["value"]

    def set(self, key: str, value: Any) -> None:
        """Set cached value.

        Args:
            key: Cache key
            value: Value to cache
        """
        self.cache[key] = {"value": value, "expires_at": datetime.now() + self.ttl}

    def clear(self) -> None:
        """Clear all cached entries."""
        self.cache.clear()


class RateLimiter:
    """Simple rate limiter for API calls."""

    def __init__(self, requests_per_minute: int = 50):
        """Initialize rate limiter.

        Args:
            requests_per_minute: Maximum requests per minute
        """
        self.requests_per_minute = requests_per_minute
        self.min_interval = 60.0 / requests_per_minute
        self.last_request: Optional[float] = None

    def wait(self) -> None:
        """Wait if necessary to comply with rate limit."""
        if self.last_request is None:
            self.last_request = time.time()
            return

        elapsed = time.time() - self.last_request
        if elapsed < self.min_interval:
            wait_time = self.min_interval - elapsed
            time.sleep(wait_time)

        self.last_request = time.time()


class ClaudeClient(AIClient):
    """Claude AI client with caching and rate limiting."""

    def __init__(
        self,
        api_key: Optional[str] = None,
        model: str = "claude-sonnet-4-5-20250929",
        enable_cache: bool = True,
        cache_ttl: timedelta = timedelta(hours=1),
        requests_per_minute: int = 50,
    ):
        """Initialize Claude client.

        Args:
            api_key: Anthropic API key (defaults to ANTHROPIC_API_KEY env var)
            model: Claude model to use
            enable_cache: Whether to enable response caching
            cache_ttl: Cache time-to-live
            requests_per_minute: Rate limit for API calls
        """
        self.api_key = api_key or os.getenv("ANTHROPIC_API_KEY")
        if not self.api_key:
            raise ValueError("ANTHROPIC_API_KEY not set")

        self.client = Anthropic(api_key=self.api_key)
        self.model = model
        self.cache = ResponseCache(cache_ttl) if enable_cache else None
        self.rate_limiter = RateLimiter(requests_per_minute)

    def generate(self, request: GenerateRequest) -> GenerateResponse:
        """Generate policy-compliant configuration.

        Args:
            request: Generation request

        Returns:
            GenerateResponse with generated configuration
        """
        # Check cache
        cache_key = f"generate:{request.domain}:{request.requirements}"
        if self.cache:
            cached = self.cache.get(cache_key)
            if cached:
                return cached

        # Rate limit
        self.rate_limiter.wait()

        # Build prompt
        prompt = build_generate_prompt(
            request.domain, request.requirements, request.policies
        )

        # Call Claude API
        response = self.client.messages.create(
            model=self.model, max_tokens=4096, messages=[{"role": "user", "content": prompt}]
        )

        # Extract content
        content = self._extract_text_content(response)
        config, explanation = self._parse_generated_config(content)

        result = GenerateResponse(
            configuration=config,
            format=Format.YAML,
            explanation=explanation,
            policies_met=request.policies or [],
        )

        # Cache result
        if self.cache:
            self.cache.set(cache_key, result)

        return result

    def remediate(
        self, domain: str, original_config: str, violations: List[Violation]
    ) -> FixResponse:
        """Suggest fixes for policy violations.

        Args:
            domain: Domain name
            original_config: Original configuration
            violations: List of violations to fix

        Returns:
            FixResponse with fixed configuration
        """
        # Check cache
        cache_key = f"remediate:{domain}:{len(violations)}"
        if self.cache:
            cached = self.cache.get(cache_key)
            if cached:
                return cached

        # Rate limit
        self.rate_limiter.wait()

        # Build prompt
        violations_text = format_violations(violations)
        prompt = build_remediate_prompt(domain, original_config, violations_text)

        # Call Claude API
        response = self.client.messages.create(
            model=self.model, max_tokens=4096, messages=[{"role": "user", "content": prompt}]
        )

        # Extract content
        content = self._extract_text_content(response)
        fixed_config, explanation = self._parse_generated_config(content)

        # Create changes list
        changes = [
            {
                "field": v.field,
                "from": v.current_value,
                "to": v.expected_value,
                "reason": v.message,
            }
            for v in violations
            if v.field
        ]

        result = FixResponse(
            original=original_config,
            fixed=fixed_config,
            changes=changes,
            explanation=explanation,
        )

        # Cache result
        if self.cache:
            self.cache.set(cache_key, result)

        return result

    def explain(self, policy: str, violation: Optional[Violation] = None) -> str:
        """Explain a policy or violation.

        Args:
            policy: Policy name to explain
            violation: Optional specific violation

        Returns:
            Human-readable explanation
        """
        # Determine domain from policy name (e.g., "kafka.topics.replication" -> "kafka")
        domain = policy.split(".")[0] if "." in policy else "general"

        # Rate limit
        self.rate_limiter.wait()

        # Build prompt
        violation_msg = violation.message if violation else None
        prompt = build_explain_prompt(domain, policy, violation_msg)

        # Call Claude API
        response = self.client.messages.create(
            model=self.model, max_tokens=2048, messages=[{"role": "user", "content": prompt}]
        )

        # Extract content
        return self._extract_text_content(response)

    def _extract_text_content(self, response) -> str:
        """Extract text content from Claude API response.

        Args:
            response: Anthropic API response

        Returns:
            Extracted text content
        """
        content_blocks = []
        for block in response.content:
            if block.type == "text":
                content_blocks.append(block.text)

        return "\n".join(content_blocks)

    def _parse_generated_config(self, content: str) -> tuple[str, str]:
        """Parse generated configuration and explanation from response.

        Args:
            content: Response content from Claude

        Returns:
            Tuple of (config, explanation)
        """
        # Try to extract YAML block
        config = ""
        explanation = ""

        if "```yaml" in content:
            # Extract YAML code block
            start = content.find("```yaml")
            start = content.find("\n", start) + 1
            end = content.find("```", start)

            if end != -1:
                config = content[start:end].strip()
                # Everything else is explanation
                explanation = (
                    content[: start - 6].strip() + "\n\n" + content[end + 3 :].strip()
                ).strip()
        elif "```" in content:
            # Generic code block
            start = content.find("```")
            start = content.find("\n", start) + 1
            end = content.find("```", start)

            if end != -1:
                config = content[start:end].strip()
                explanation = (
                    content[: start - 3].strip() + "\n\n" + content[end + 3 :].strip()
                ).strip()

        if not config:
            # No code block found, use entire response as config
            config = content
            explanation = "Generated configuration based on requirements"

        return config, explanation
