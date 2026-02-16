# Policy Agent Examples

This directory contains example configurations for testing the Policy AI Agent across different domains.

## Directory Structure

```
examples/
├── cicd/
│   ├── github-actions/
│   │   ├── valid-workflow.yaml
│   │   └── invalid-workflow.yaml
│   └── gitlab-ci/
│       ├── valid-pipeline.yaml
│       └── invalid-pipeline.yaml
└── gitops/
    ├── flux/
    │   ├── valid-kustomization.yaml
    │   ├── invalid-kustomization.yaml
    │   ├── valid-gitrepository.yaml
    │   ├── invalid-gitrepository.yaml
    │   ├── valid-helmrelease.yaml
    │   └── invalid-helmrelease.yaml
    └── argocd/
        ├── valid-application.yaml
        ├── invalid-application.yaml
        ├── valid-appproject.yaml
        └── invalid-appproject.yaml
```

## Usage

### Validating CI/CD Configurations

#### GitHub Actions

**Valid Workflow** (should pass all checks):
```bash
# Go
policy-agent validate --file examples/cicd/github-actions/valid-workflow.yaml

# Python
policy-agent validate --file examples/cicd/github-actions/valid-workflow.yaml
```

**Invalid Workflow** (should fail with violations):
```bash
policy-agent validate --file examples/cicd/github-actions/invalid-workflow.yaml
```

Expected violations:
- Missing workflow name
- No test job defined
- Third-party actions not pinned to commit SHA
- Potential secret exposure in logs
- Write permissions granted
- No timeout set on jobs
- Timeout exceeds maximum allowed
- No security scanning configured

#### GitLab CI

**Valid Pipeline** (should pass all checks):
```bash
policy-agent validate --file examples/cicd/gitlab-ci/valid-pipeline.yaml
```

**Invalid Pipeline** (should fail with violations):
```bash
policy-agent validate --file examples/cicd/gitlab-ci/invalid-pipeline.yaml
```

Expected violations:
- No test stage defined
- Using `:latest` Docker tag
- Privileged Docker mode enabled
- Production deployment without manual approval
- Missing artifact expiration

### Validating GitOps Configurations

#### Flux CD

**Kustomization**:
```bash
# Valid
policy-agent validate --file examples/gitops/flux/valid-kustomization.yaml

# Invalid
policy-agent validate --file examples/gitops/flux/invalid-kustomization.yaml
```

Invalid violations:
- Interval too long
- Missing sourceRef
- Path traversal vulnerability
- Prune not enabled
- Force enabled
- No timeout set
- No health checks for production

**GitRepository**:
```bash
# Valid
policy-agent validate --file examples/gitops/flux/valid-gitrepository.yaml

# Invalid
policy-agent validate --file examples/gitops/flux/invalid-gitrepository.yaml
```

Invalid violations:
- Using insecure HTTP protocol
- Interval too short
- No explicit ref specified
- No GPG verification for production

**HelmRelease**:
```bash
# Valid
policy-agent validate --file examples/gitops/flux/valid-helmrelease.yaml

# Invalid
policy-agent validate --file examples/gitops/flux/invalid-helmrelease.yaml
```

Invalid violations:
- No interval specified
- Missing chart reference
- Too many install retries
- CRDs set to Skip
- Force upgrade enabled
- Rollback disabled for production
- Using version range instead of pinned version
- Resource is suspended

#### ArgoCD

**Application**:
```bash
# Valid
policy-agent validate --file examples/gitops/argocd/valid-application.yaml

# Invalid
policy-agent validate --file examples/gitops/argocd/invalid-application.yaml
```

Invalid violations:
- No finalizers (accidental deletion risk)
- Using default project
- Using insecure HTTP
- targetRevision set to HEAD
- No namespace specified
- Auto-sync for production
- Prune not enabled
- Using Replace=true sync option
- Too many retries
- Ignoring all fields (too broad)

**AppProject**:
```bash
# Valid
policy-agent validate --file examples/gitops/argocd/valid-appproject.yaml

# Invalid
policy-agent validate --file examples/gitops/argocd/invalid-appproject.yaml
```

Invalid violations:
- Using 'default' project name
- Wildcard source repositories
- Empty destinations list
- Wildcard cluster resources
- No RBAC roles defined
- No orphaned resources detection
- No sync windows for change control
- No signature keys
- Role with wildcard permissions

## Batch Validation

Validate all examples in a directory:

```bash
# GitHub Actions
policy-agent validate --dir examples/cicd/github-actions/

# GitLab CI
policy-agent validate --dir examples/cicd/gitlab-ci/

# Flux CD
policy-agent validate --dir examples/gitops/flux/

# ArgoCD
policy-agent validate --dir examples/gitops/argocd/

# All examples
policy-agent validate --dir examples/
```

## Generating Compliant Configurations

Use the AI-powered generation feature to create policy-compliant configurations:

```bash
# Generate GitHub Actions workflow
policy-agent generate --domain cicd \
  --requirements "Create a CI workflow with testing, security scanning, and Docker build" \
  --output my-workflow.yaml

# Generate Flux Kustomization
policy-agent generate --domain gitops \
  --requirements "Create a Kustomization for production with health checks and prune enabled" \
  --output my-kustomization.yaml

# Generate ArgoCD Application
policy-agent generate --domain gitops \
  --requirements "Create an Application for staging environment with auto-sync and self-heal" \
  --output my-application.yaml
```

## Fixing Violations

Interactive fix mode:

```bash
# Fix CI/CD violations
policy-agent fix --file examples/cicd/github-actions/invalid-workflow.yaml --interactive

# Fix GitOps violations
policy-agent fix --file examples/gitops/flux/invalid-kustomization.yaml --interactive
```

## CI/CD Integration

### GitHub Actions

```yaml
name: Policy Validation
on: [push, pull_request]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install Policy Agent
        run: curl -sSfL https://policy-agent.io/install.sh | sh

      - name: Validate Configurations
        run: policy-agent validate --dir examples/
```

### GitLab CI

```yaml
validate:
  stage: test
  image: policy-agent/policy-agent:latest
  script:
    - policy-agent validate --dir examples/
  allow_failure: false
```

## Learning from Examples

Each example file contains:
- **Valid examples**: Demonstrate best practices and proper configuration
- **Invalid examples**: Show common violations with inline comments explaining each issue

Use these examples to:
1. Understand security best practices for CI/CD and GitOps
2. Test the Policy Agent validation capabilities
3. Learn what to avoid in your configurations
4. Generate baseline configurations for your projects

## Custom Policies

To add custom policies for these examples:

1. Create new Rego policies in `policies/cicd/` or `policies/gitops/`
2. Update `policy-agent.yaml` to enable custom policies
3. Run validation against examples to test new policies

## Contributing

To add new examples:
1. Create valid and invalid versions of your configuration
2. Add inline comments explaining violations in invalid examples
3. Update this README with usage instructions
4. Test with both Go and Python implementations
