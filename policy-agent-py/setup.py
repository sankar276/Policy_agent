"""Setup script for policy-agent Python package."""

from setuptools import setup, find_packages
from pathlib import Path

# Read README
readme_file = Path(__file__).parent / "README.md"
long_description = readme_file.read_text() if readme_file.exists() else ""

setup(
    name="policy-agent",
    version="0.1.0",
    description="Unified AI-powered policy validation and enforcement",
    long_description=long_description,
    long_description_content_type="text/markdown",
    author="Policy Agent Contributors",
    author_email="info@policy-agent.io",
    url="https://github.com/policy-agent/policy-agent",
    license="Apache-2.0",
    packages=find_packages(exclude=["tests", "tests.*"]),
    python_requires=">=3.8",
    install_requires=[
        "click>=8.0.0",
        "pyyaml>=6.0",
        "anthropic>=0.18.0",
        "requests>=2.28.0",
    ],
    extras_require={
        "dev": [
            "pytest>=7.0.0",
            "pytest-cov>=4.0.0",
            "black>=22.0.0",
            "flake8>=5.0.0",
            "mypy>=0.990",
            "types-PyYAML",
            "types-requests",
        ],
        "opa": [
            "opa-python-client>=1.3.0",
        ],
    },
    entry_points={
        "console_scripts": [
            "policy-agent=policy_agent.cli.main_enhanced:cli",
        ],
    },
    classifiers=[
        "Development Status :: 3 - Alpha",
        "Intended Audience :: Developers",
        "Intended Audience :: System Administrators",
        "License :: OSI Approved :: Apache Software License",
        "Operating System :: OS Independent",
        "Programming Language :: Python :: 3",
        "Programming Language :: Python :: 3.8",
        "Programming Language :: Python :: 3.9",
        "Programming Language :: Python :: 3.10",
        "Programming Language :: Python :: 3.11",
        "Programming Language :: Python :: 3.12",
        "Topic :: Software Development :: Quality Assurance",
        "Topic :: System :: Systems Administration",
        "Topic :: Utilities",
    ],
    keywords="policy validation enforcement opa rego kubernetes kafka terraform ai",
    project_urls={
        "Documentation": "https://policy-agent.io/docs",
        "Source": "https://github.com/policy-agent/policy-agent",
        "Tracker": "https://github.com/policy-agent/policy-agent/issues",
    },
    include_package_data=True,
    zip_safe=False,
)
