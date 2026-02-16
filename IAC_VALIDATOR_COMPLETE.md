## ✅ IaC Validator Implementation Complete!

I've successfully added comprehensive Infrastructure as Code (Terraform) validation support to the Policy AI Agent. Here's what was created:

### Files Created

#### 1. **OPA/Rego Policies** (4 files)
- [policies/iac/terraform/provider.rego](policies/iac/terraform/provider.rego) - Provider version pinning
- [policies/iac/terraform/state.rego](policies/iac/terraform/state.rego) - Remote state backend configuration
- [policies/iac/terraform/resources.rego](policies/iac/terraform/resources.rego) - Resource tagging and configuration
- [policies/iac/terraform/security.rego](policies/iac/terraform/security.rego) - Security best practices

#### 2. **Go Implementation**
- [policy-agent/internal/validator/iac/validator.go](policy-agent/internal/validator/iac/validator.go) - Complete Go validator

#### 3. **Python Implementation**
- [policy-agent-py/policy_agent/validators/iac.py](policy-agent-py/policy_agent/validators/iac.py) - Complete Python validator

#### 4. **AI Prompts**
- Updated [policy-agent/internal/ai/prompts.go](policy-agent/internal/ai/prompts.go) - Enhanced Terraform prompts
- Updated [policy-agent-py/policy_agent/ai/prompts.py](policy-agent-py/policy_agent/ai/prompts.py) - Enhanced Terraform prompts

#### 5. **Examples**
- [examples/terraform/valid-s3-bucket.yaml](examples/terraform/valid-s3-bucket.yaml) - Valid configuration
- [examples/terraform/invalid-infrastructure.yaml](examples/terraform/invalid-infrastructure.yaml) - Multiple violations

---

## Terraform Policies

### 1. Provider Configuration
✅ **Validates:**
- Terraform version pinning (`required_version`)
- Provider version constraints (using `~>`)
- No loose version constraints (e.g., `>= 5.0` without upper bound)

**Example violation:**
```hcl
terraform {
  # Missing required_version ❌
  required_providers {
    aws = {
      source = "hashicorp/aws"
      # Missing version ❌
    }
  }
}
```

**Fixed:**
```hcl
terraform {
  required_version = ">= 1.5.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"  # ✅ Pinned
    }
  }
}
```

### 2. State Backend Configuration
✅ **Validates:**
- Remote state backend (S3, Azure, GCS)
- Encryption enabled
- State locking (DynamoDB for S3)
- No local backend in production

**Example violation:**
```hcl
terraform {
  backend "local" {  # ❌ Local backend
    path = "terraform.tfstate"
  }
}
```

**Fixed:**
```hcl
terraform {
  backend "s3" {
    bucket         = "myorg-tfstate"
    key            = "prod/terraform.tfstate"
    region         = "us-east-1"
    encrypt        = true              # ✅ Encrypted
    dynamodb_table = "terraform-lock"  # ✅ State locking
  }
}
```

### 3. Resource Tagging
✅ **Validates:**
- Required tags present: `Environment`, `Owner`, `Project`, `ManagedBy`
- Valid environment values: `dev`, `staging`, `prod`
- Naming conventions (lowercase with hyphens)

**Example violation:**
```hcl
resource "aws_s3_bucket" "data" {
  bucket = "MyBucket"  # ❌ Wrong naming convention
  # Missing tags ❌
}
```

**Fixed:**
```hcl
resource "aws_s3_bucket" "data" {
  bucket = "my-bucket-prod"  # ✅ Lowercase with hyphens

  tags = {
    Environment = "prod"      # ✅ Required
    Owner       = "data-team"
    Project     = "analytics"
    ManagedBy   = "terraform"
  }
}
```

### 4. Security Best Practices
✅ **Validates:**
- **S3 Buckets:**
  - Private ACL (no public access)
  - Server-side encryption enabled
  - Versioning enabled (recommended)
  - Access logging (recommended)

- **RDS Instances:**
  - Not publicly accessible
  - Storage encryption enabled
  - Backup retention ≥ 7 days
  - CloudWatch logs export (recommended)

- **Security Groups:**
  - No 0.0.0.0/0 ingress except ports 80/443
  - Specific CIDR blocks for other ports

- **IAM Policies:**
  - No wildcard (*:*) admin access
  - Principle of least privilege

- **General:**
  - No hardcoded secrets
  - VPC flow logs enabled
  - Encryption at rest for all storage

**Example violations:**
```hcl
resource "aws_s3_bucket" "data" {
  bucket = "my-bucket"
  acl    = "public-read"  # ❌ PUBLIC!
  # No encryption ❌
}

resource "aws_rds_instance" "db" {
  publicly_accessible = true   # ❌ PUBLIC!
  storage_encrypted   = false  # ❌ NO ENCRYPTION!
  backup_retention_period = 0  # ❌ NO BACKUPS!
}

resource "aws_security_group" "web" {
  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]  # ❌ SSH OPEN TO WORLD!
  }
}
```

**Fixed:**
```hcl
resource "aws_s3_bucket" "data" {
  bucket = "my-bucket"
  acl    = "private"  # ✅ Private

  server_side_encryption_configuration {
    rule {
      apply_server_side_encryption_by_default {
        sse_algorithm = "AES256"  # ✅ Encrypted
      }
    }
  }

  versioning {
    enabled = true  # ✅ Versioning
  }
}

resource "aws_rds_instance" "db" {
  publicly_accessible     = false  # ✅ Private
  storage_encrypted       = true   # ✅ Encrypted
  backup_retention_period = 7      # ✅ 7-day backups

  enabled_cloudwatch_logs_exports = ["error", "general", "slowquery"]
}

resource "aws_security_group" "web" {
  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["10.0.0.0/8"]  # ✅ Restricted to VPC
  }

  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]  # ✅ HTTPS allowed from anywhere
  }
}
```

---

## Usage

### CLI Validation

```bash
# Validate Terraform configuration
policy-agent validate --file terraform-config.yaml --domain iac

# JSON output
policy-agent validate --file terraform-config.yaml --format json
```

### Python API

```python
from policy_agent.validators.iac import IaCValidator, IaCConfig
import yaml

# Load configuration
with open("terraform-config.yaml") as f:
    data = yaml.safe_load(f)

# Create validator with custom config
config = IaCConfig(
    required_tags=["Environment", "Owner", "CostCenter"],
    require_encryption=True,
    require_remote_state=True
)

validator = IaCValidator(config=config)

# Validate
result = validator.validate(data)

# Check results
if result.status == "failed":
    print(f"Found {len(result.violations)} violations:")
    for v in result.violations:
        print(f"  [{v.severity.value}] {v.message}")
else:
    print("✅ All checks passed!")
```

### AI Generation

```bash
# Generate Terraform configuration with AI
export ANTHROPIC_API_KEY="your-key"

policy-agent generate \
  --domain iac \
  --requirements "Create S3 bucket for application logs with encryption and lifecycle policies" \
  --output s3-logs.yaml
```

### AI Remediation

```bash
# Fix violations automatically
policy-agent fix \
  --file invalid-terraform.yaml \
  --output fixed-terraform.yaml

# Interactive mode
policy-agent fix --file terraform.yaml --interactive
```

---

## Test Examples

### Valid Configuration

See [examples/terraform/valid-s3-bucket.yaml](examples/terraform/valid-s3-bucket.yaml):
- ✅ Terraform version pinned
- ✅ Provider versions specified
- ✅ Remote S3 backend with encryption
- ✅ DynamoDB state locking
- ✅ All required tags
- ✅ Private ACL
- ✅ Server-side encryption
- ✅ Versioning enabled

**Expected result:** All checks pass

### Invalid Configuration

See [examples/terraform/invalid-infrastructure.yaml](examples/terraform/invalid-infrastructure.yaml):
- ❌ Missing Terraform version
- ❌ Missing provider versions
- ❌ Using local backend
- ❌ Public S3 bucket
- ❌ No encryption
- ❌ Missing tags
- ❌ RDS publicly accessible
- ❌ Wide-open security group rules

**Expected result:** 10+ violations found

### Test Command

```bash
# Test with valid config
policy-agent validate --file examples/terraform/valid-s3-bucket.yaml

# Expected output:
# ✅ valid-s3-bucket.yaml
#    Status: PASSED
#    Passed: 5 checks

# Test with invalid config
policy-agent validate --file examples/terraform/invalid-infrastructure.yaml

# Expected output:
# ❌ invalid-infrastructure.yaml
#    Status: FAILED
#    Violations: 10+
```

---

## Policy Severity Levels

| Severity | Use Case | Examples |
|----------|----------|----------|
| **CRITICAL** | Security breaches, data exposure | Public S3 buckets, public RDS |
| **HIGH** | Major security/compliance issues | Missing encryption, no version pinning |
| **MEDIUM** | Best practices, operational issues | Missing tags, backup retention |
| **LOW** | Recommendations, optimizations | CloudWatch logs, cost optimization |

---

## Supported Resources

### AWS
- `aws_s3_bucket` - Buckets
- `aws_rds_instance` - Databases
- `aws_instance` - EC2 instances
- `aws_security_group` - Security groups
- `aws_vpc` - VPCs
- `aws_subnet` - Subnets
- `aws_ebs_volume` - EBS volumes
- `aws_lambda_function` - Lambda functions
- `aws_iam_policy` - IAM policies
- `aws_lb` - Load balancers

### Azure
- `azurerm_resource_group` - Resource groups
- `azurerm_virtual_machine` - VMs
- `azurerm_storage_account` - Storage accounts

### GCP
- `google_compute_instance` - Compute instances
- `google_storage_bucket` - Storage buckets

---

## Next Steps

The IaC validator is now complete with:
- ✅ Comprehensive OPA/Rego policies
- ✅ Go implementation
- ✅ Python implementation
- ✅ AI prompt templates
- ✅ Example configurations
- ✅ Full documentation

**You can now:**
1. Validate existing Terraform configurations
2. Generate new compliant configurations with AI
3. Auto-fix policy violations
4. Integrate into CI/CD pipelines
5. Use as a Terraform pre-commit hook

**Try it out:**
```bash
# Quick test
cd policy-agent-py
pip install -e .

policy-agent validate --file ../examples/terraform/invalid-infrastructure.yaml

# Expected: 10+ violations found!
```

🎉 **IaC domain validation is ready to use!**
