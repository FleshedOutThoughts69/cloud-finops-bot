# Configuration Matrix: Cloud FinOps Bot

**Document:** 04
**Version:** 3.0 (Go Edition - Enhanced & Audited)
**Author:** Jibrin Ahmed
**Date:** June 22, 2026
**Status:** Final

---

## 1. Document Purpose

This document defines the **complete configuration inventory** for the FinOps Bot. It maps every configuration value to its source:

- **Lambda Environment Variables:** Non-sensitive, frequently toggled values.
- **AWS Secrets Manager:** Sensitive values (encrypted, auditable, rotatable).
- **SSM Parameter Store:** Non-sensitive configuration values (auditable, versioned).
- **IAM Roles:** AWS credentials (never stored).
- **Terraform Variables:** Infrastructure-level configuration.

This separation ensures:
- **Security:** Secrets are never exposed in logs, CloudWatch, or Terraform state.
- **Operability:** Non-sensitive values can be changed without rotating secrets.
- **Auditability:** Secret rotations are tracked in AWS CloudTrail.
- **Least Privilege:** Only the Lambda role has access to secrets and parameters.

---

## 2. Configuration Overview

### 2.1 Configuration Sources

| Source | Purpose | Examples |
| :--- | :--- | :--- |
| **Lambda Environment Variables** | Non-sensitive, frequently toggled values | `DRY_RUN`, `LOG_LEVEL`, `REGIONS` |
| **AWS Secrets Manager** | Sensitive values (encrypted, auditable) | `SLACK_WEBHOOK_URL` |
| **SSM Parameter Store** | Non-sensitive, versioned configuration | `regions`, `quarantine_days`, `snapshot_retention` |
| **IAM Role** | AWS credentials (auto-rotated) | `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` |

### 2.2 Complete Configuration Matrix

| Configuration Key | Source | Sensitive? | Default Value | Purpose |
| :--- | :--- | :--- | :--- | :--- |
| `DRY_RUN` | Lambda Env Var | ❌ No | `true` | If `true`, the bot logs actions but does NOT execute deletions/stops. |
| `COST_PER_GB` | Lambda Env Var | ❌ No | `0.08` | Cost per GB-month for EBS volumes (us-east-1 gp3 pricing). Override for other regions. |
| `EXCLUDED_IDS` | Lambda Env Var | ❌ No | `""` | Comma-separated list of resource IDs to skip unconditionally (emergency kill-switch). **No spaces.** |
| `REGIONS` | SSM Parameter Store | ❌ No | `us-east-1,us-west-2,eu-west-1` | Comma-separated list of AWS regions to scan. **No spaces.** |
| `QUARANTINE_DAYS` | SSM Parameter Store | ❌ No | `7` | Number of days to quarantine a resource before deletion. |
| `SNAPSHOT_RETENTION_DAYS` | SSM Parameter Store | ❌ No | `30` | Number of days to retain snapshots (older snapshots are candidates for deletion). |
| `SNAPSHOTS_TO_KEEP` | Lambda Env Var | ❌ No | `3` | Number of most recent snapshots to preserve per volume (regardless of age). |
| `RDS_STOP_AGE_DAYS` | Lambda Env Var | ❌ No | `7` | Minimum age in days for an RDS instance to be considered for stopping. |
| `LOG_LEVEL` | Lambda Env Var | ❌ No | `info` | Logging verbosity. Values: `debug`, `info`, `warn`, `error`. |
| `SLACK_WEBHOOK_URL` | **Secrets Manager** | ✅ Yes | `""` | Slack Incoming Webhook URL for notifications. |
| `SLACK_CHANNEL` | Lambda Env Var | ❌ No | `""` | Slack channel to send notifications to. If empty, uses webhook's default channel. |
| `S3_REPORT_BUCKET` | Lambda Env Var | ❌ No | (Required) | Name of the S3 bucket for storing audit logs and the static HTML dashboard. |
| `S3_REPORT_PREFIX` | Lambda Env Var | ❌ No | `audit/` | Prefix (folder) for storing audit logs and the dashboard in the S3 bucket. |
| `TZ` | Lambda Env Var | ❌ No | `UTC` | Timezone for logging and audit timestamps. Example: `America/New_York`. |
| `SNS_TOPIC_ARN` | Lambda Env Var | ❌ No | `""` | ARN of an SNS topic for critical failure alerts (e.g., secret fetch failure). Optional. |
| `SECRETS_MANAGER_ARN` | Lambda Env Var | ❌ No | (Required) | ARN of the Secrets Manager secret containing `SLACK_WEBHOOK_URL`. |
| `ENABLE_RDS_SAVINGS` | Lambda Env Var | ❌ No | `true` | Enable RDS savings calculations and stopping. |
| `ENVIRONMENT` | Lambda Env Var | ❌ No | `dev` | Environment name (dev, staging, prod). Used for tagging and contextual logging. |
| `AWS_ACCESS_KEY_ID` | IAM Role | ✅ Yes | N/A | **Never stored.** Lambda assumes an IAM role. |
| `AWS_SECRET_ACCESS_KEY` | IAM Role | ✅ Yes | N/A | **Never stored.** Lambda assumes an IAM role. |
| `AWS_SESSION_TOKEN` | IAM Role | ✅ Yes | N/A | **Never stored.** Lambda assumes an IAM role. |
| `AWS_ENDPOINT_URL` | Lambda Env Var (Local Only) | ❌ No | `""` | **Local development only.** Overrides AWS endpoints to point to Floci (`http://localhost:4566`). |
| `TERRAFORM_STATE_BUCKET` | Terraform Backend | ❌ No | `finops-terraform-state-<account-id>` | S3 bucket name for Terraform remote state. |
| `TERRAFORM_STATE_REGION` | Terraform Backend | ❌ No | `us-east-1` | AWS region for the Terraform state bucket. |
| `TERRAFORM_LOCK_TABLE` | Terraform Backend | ❌ No | `terraform-locks` | DynamoDB table name for Terraform state locking. |

---

## 3. Detailed Configuration Specifications

### 3.1 Lambda Environment Variables

These are set directly in the Lambda function configuration via Terraform (or manually in the AWS Console). They can be changed without deploying new code.

**Config Map (Terraform):**
```hcl
# terraform/lambda.tf

environment {
  variables = {
    DRY_RUN                 = var.dry_run
    COST_PER_GB             = var.cost_per_gb
    EXCLUDED_IDS            = var.excluded_ids
    # REGIONS is fetched from SSM Parameter Store
    # QUARANTINE_DAYS is fetched from SSM Parameter Store
    # SNAPSHOT_RETENTION_DAYS is fetched from SSM Parameter Store
    SNAPSHOTS_TO_KEEP       = var.snapshots_to_keep
    RDS_STOP_AGE_DAYS       = var.rds_stop_age_days
    LOG_LEVEL               = var.log_level
    SLACK_CHANNEL           = var.slack_channel
    S3_REPORT_BUCKET        = var.s3_report_bucket
    S3_REPORT_PREFIX        = var.s3_report_prefix
    TZ                      = var.timezone
    SECRETS_MANAGER_ARN     = aws_secretsmanager_secret.slack_webhook.arn
    ENABLE_RDS_SAVINGS      = var.enable_rds_savings
    ENVIRONMENT             = var.environment
  }
}
```

**Validation Rules:**
| Variable | Validation |
| :--- | :--- |
| `DRY_RUN` | Must be `true` or `false` (string). |
| `COST_PER_GB` | Must be a positive float (e.g., `0.08`). |
| `EXCLUDED_IDS` | Comma-separated list with no spaces: `vol-123,vol-456`. |
| `SNAPSHOTS_TO_KEEP` | Must be an integer >= 0. |
| `RDS_STOP_AGE_DAYS` | Must be an integer > 0. |
| `LOG_LEVEL` | Must be one of: `debug`, `info`, `warn`, `error`. |
| `SLACK_CHANNEL` | Valid Slack channel name (e.g., `#finops-alerts`). Optional. |
| `S3_REPORT_BUCKET` | Must be a valid S3 bucket name. |
| `S3_REPORT_PREFIX` | Must be a valid S3 prefix (e.g., `audit/`). |
| `TZ` | Valid IANA timezone (e.g., `America/New_York`). |
| `SECRETS_MANAGER_ARN` | Valid Secrets Manager ARN. |
| `ENVIRONMENT` | Must be one of: `dev`, `staging`, `prod`. |

**SLACK_CHANNEL Behavior:**
- If `SLACK_CHANNEL` is set, the bot sends messages to this channel (overrides the webhook's default).
- If `SLACK_CHANNEL` is not set, the bot sends messages to the webhook's default channel.

**EXCLUDED_IDS Format:**
- Comma-separated list with **no spaces**: `vol-123,vol-456,eipalloc-789`.
- **Go Parsing Example:**
  ```go
  excludedIDs := strings.Split(os.Getenv("EXCLUDED_IDS"), ",")
  for _, id := range excludedIDs {
      id = strings.TrimSpace(id) // Sanitize just in case
      if id != "" {
          // Add to exclusion map
      }
  }
  ```

---

### 3.2 AWS Secrets Manager

The Slack Webhook URL is stored in AWS Secrets Manager. This ensures:

- **Encryption at rest** (using AWS KMS).
- **Auditable access** (CloudTrail logs every access).
- **Rotatable** (can be changed without updating Lambda configuration).

**Secret Structure:**
```json
{
  REPLACE_WITH_YOUR_SLACK_WEBHOOK_URL
}
```

**KMS Key Configuration:**
- Use a **customer-managed KMS key** (or AWS-managed `aws/secretsmanager` key).
- Lambda IAM role must have `kms:Decrypt` permission on the key.

**Terraform Configuration:**
```hcl
# terraform/secrets.tf

resource "aws_secretsmanager_secret" "slack_webhook" {
  name                    = "finops/slack-webhook-${var.environment}"
  description             = "Slack webhook URL for FinOps Bot notifications"
  kms_key_id              = aws_kms_key.secrets_key.arn
  recovery_window_in_days = 30
}

resource "aws_secretsmanager_secret_version" "slack_webhook" {
  secret_id     = aws_secretsmanager_secret.slack_webhook.id
  secret_string = jsonencode({
    SLACK_WEBHOOK_URL = "
  })
}
```

**Lambda IAM Permissions Required:**
```json
{
  "Effect": "Allow",
  "Action": [
    "secretsmanager:GetSecretValue",
    "kms:Decrypt"
  ],
  "Resource": [
    "arn:aws:secretsmanager:<region>:<account-id>:secret:finops/slack-webhook-*",
    "arn:aws:kms:<region>:<account-id>:key/<key-id>"
  ]
}
```

**Go SDK Code to Fetch Secret:**
```go
// internal/secrets/manager.go

package secrets

import (
    "context"
    "encoding/json"
    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type SlackSecret struct {
    WebhookURL string `json:"SLACK_WEBHOOK_URL"`
}

func GetSlackWebhook(ctx context.Context, client *secretsmanager.Client, secretARN string) (string, error) {
    input := &secretsmanager.GetSecretValueInput{
        SecretId: aws.String(secretARN),
    }

    result, err := client.GetSecretValue(ctx, input)
    if err != nil {
        return "", err
    }

    var secret SlackSecret
    if err := json.Unmarshal([]byte(*result.SecretString), &secret); err != nil {
        return "", err
    }

    return secret.WebhookURL, nil
}
```

**Secret Rotation Strategy:**
| Secret | Rotation Method | Frequency | Downtime |
| :--- | :--- | :--- | :--- |
| Slack Webhook URL | Manual via AWS Console/CLI | As needed | None |
| KMS Key | Automatic (AWS-managed rotation) | Every 365 days | None |

**Rotation Procedure:**
```bash
# 1. Create a new webhook URL in Slack
# 2. Update the secret
aws secretsmanager update-secret \
  --secret-id finops/slack-webhook-dev \
  --secret-string '{"SLACK_WEBHOOK_URL":"REPLACE_WITH_YOUR_SLACK_WEBHOOK_URL"}'

# 3. The Lambda will fetch the new value on the next invocation
```

---

### 3.3 SSM Parameter Store

Non-sensitive configuration values are stored in SSM Parameter Store. This provides:

- **Versioning:** Changes are tracked and can be rolled back.
- **Auditing:** CloudTrail logs all access.
- **Centralized Management:** Configuration can be updated without code changes.

**Parameter Definitions:**
```hcl
# terraform/secrets.tf

resource "aws_ssm_parameter" "regions" {
  name  = "/finops/${var.environment}/regions"
  type  = "String"
  value = var.regions
  description = "AWS regions to scan"
  tags = {
    Name = "finops-regions-${var.environment}"
  }
}

resource "aws_ssm_parameter" "quarantine_days" {
  name  = "/finops/${var.environment}/quarantine_days"
  type  = "String"
  value = tostring(var.quarantine_days)
  description = "Number of days to quarantine resources"
  tags = {
    Name = "finops-quarantine-${var.environment}"
  }
}

resource "aws_ssm_parameter" "snapshot_retention" {
  name  = "/finops/${var.environment}/snapshot_retention_days"
  type  = "String"
  value = tostring(var.snapshot_retention_days)
  description = "Number of days to retain snapshots"
  tags = {
    Name = "finops-snapshot-retention-${var.environment}"
  }
}
```

**Parameter Hierarchy:**
```
/finops/
├── dev/
│   ├── regions
│   ├── quarantine_days
│   └── snapshot_retention_days
├── staging/
│   ├── regions
│   ├── quarantine_days
│   └── snapshot_retention_days
└── prod/
    ├── regions
    ├── quarantine_days
    └── snapshot_retention_days
```

**Go SDK Code to Fetch Parameters:**
```go
// internal/secrets/manager.go

func GetSSMParameter(ctx context.Context, client *ssm.Client, name string) (string, error) {
    input := &ssm.GetParameterInput{
        Name:           aws.String(name),
        WithDecryption: aws.Bool(true),
    }
    
    result, err := client.GetParameter(ctx, input)
    if err != nil {
        return "", err
    }
    
    return *result.Parameter.Value, nil
}

// Load configuration from SSM
func LoadSSMConfig(ctx context.Context, ssmClient *ssm.Client, environment string) (*SSMConfig, error) {
    basePath := fmt.Sprintf("/finops/%s/", environment)
    
    // Get all parameters under the path
    input := &ssm.GetParametersByPathInput{
        Path:           aws.String(basePath),
        WithDecryption: aws.Bool(true),
        Recursive:      aws.Bool(true),
    }
    
    result, err := ssmClient.GetParametersByPath(ctx, input)
    if err != nil {
        return nil, err
    }
    
    // Parse parameters into config struct
    config := &SSMConfig{}
    for _, param := range result.Parameters {
        switch *param.Name {
        case basePath + "regions":
            config.Regions = *param.Value
        case basePath + "quarantine_days":
            config.QuarantineDays, _ = strconv.Atoi(*param.Value)
        case basePath + "snapshot_retention_days":
            config.SnapshotRetentionDays, _ = strconv.Atoi(*param.Value)
        }
    }
    
    return config, nil
}
```

---

### 3.4 IAM Role (AWS Credentials)

The Lambda function assumes an IAM role. **No credentials are ever stored in code, environment variables, or configuration files.**

- **Authentication Method:** IAM Role (Instance Metadata Service).
- **Credentials Fetching:** Automatically handled by the AWS SDK for Go v2.
- **Credential Refresh:** Automatically refreshed by the SDK.

**Go SDK Code (Credentials are auto-loaded):**
```go
import (
    "context"
    "github.com/aws/aws-sdk-go-v2/config"
)

cfg, err := config.LoadDefaultConfig(context.TODO())
// No credentials are passed. The SDK auto-discovers them from the Lambda runtime.
```

**Local Development Override (Floci):**
When running locally with Floci, credentials are **not required** because Floci emulates the services without authentication. However, the SDK will still look for them. Set dummy credentials:

```bash
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_DEFAULT_REGION=us-east-1
export AWS_ENDPOINT_URL=http://localhost:4566
```

---

### 3.5 Terraform Variables (Infrastructure Configuration)

These variables are used to customize the infrastructure provisioning. They are passed to Terraform via `terraform.tfvars` or environment variables.

| Variable | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| `aws_region` | String | `us-east-1` | AWS region for primary deployment. |
| `environment` | String | `dev` | Environment name (dev, staging, prod). |
| `dry_run` | Boolean | `true` | Default DRY_RUN value for the Lambda. |
| `cost_per_gb` | Number | `0.08` | Default cost per GB-month. |
| `excluded_ids` | String | `""` | Default excluded resource IDs. |
| `regions` | String | `us-east-1,us-west-2,eu-west-1` | Default regions to scan. |
| `secrets_manager_arn` | String | (Required) | ARN of the Secrets Manager secret containing Slack webhook. |
| `slack_channel` | String | `""` | Default Slack channel. If empty, uses webhook's default. |
| `s3_report_bucket` | String | (Required) | S3 bucket for audit logs and dashboard. |
| `s3_report_prefix` | String | `audit/` | Prefix for S3 objects. |
| `timezone` | String | `UTC` | Timezone for logging and reports. |
| `enable_rds_savings` | Boolean | `true` | Enable RDS savings calculations and stopping. |

**Example `terraform.tfvars`:**
```hcl
aws_region = "us-east-1"
environment = "dev"

dry_run = true
cost_per_gb = 0.08
excluded_ids = "vol-12345678,vol-87654321"
regions = "us-east-1,us-west-2,eu-west-1"

quarantine_days = 7
snapshot_retention_days = 30
snapshots_to_keep = 3
rds_stop_age_days = 7
log_level = "info"

slack_channel = "#finops-alerts"

s3_report_bucket = "finops-audit-123456789012"
secrets_manager_arn = "arn:aws:secretsmanager:us-east-1:123456789012:secret:finops/slack-webhook-abc123"
sns_topic_arn = "arn:aws:sns:us-east-1:123456789012:finops-alerts"

enable_rds_savings = true

lambda_memory_size = 256
lambda_timeout = 300

timezone = "America/New_York"
```

---

## 4. Security Best Practices

| Practice | Implementation |
| :--- | :--- |
| **No Hardcoded Credentials** | All AWS authentication is via IAM roles (production) or dummy credentials (local development). |
| **Secrets Not in Logs** | The Lambda will never log the `SLACK_WEBHOOK_URL` or any secret value. |
| **Secrets Not in Terraform State** | Sensitive variables are stored in **Terraform Variables** but are marked `sensitive = true` in the Terraform schema. They are never written to the state file. |
| **Secrets Rotated Regularly** | Slack Webhook URL can be rotated via the AWS Console without changing Lambda code. |
| **Least Privilege** | The Lambda IAM role has only the necessary permissions (see Document 08 for the full matrix). |
| **KMS Encryption** | Secrets Manager secret is encrypted with a customer-managed KMS key. |
| **SSM Parameter Store Versioning** | Changes to SSM parameters are versioned and auditable. |
| **Configuration Change Audit** | All Lambda environment variable changes are logged in CloudTrail. |

---

## 5. Configuration Override Precedence (Order of Evaluation)

When the Lambda starts, configuration values are resolved in the following order (highest priority first):

1. **Hardcoded Override:** Values explicitly passed in the Lambda event payload (for manual testing).
2. **Secrets Manager:** Sensitive values (`SLACK_WEBHOOK_URL`).
3. **Lambda Environment Variables:** Non-sensitive values that need to be updated frequently.
4. **SSM Parameter Store:** Non-sensitive values that are centrally managed.
5. **Default Values:** Fallback values defined in the Terraform configuration.
6. **AWS SDK Defaults:** IAM role credentials, endpoint discovery, etc.

This precedence ensures that manual testing overrides do not persist to production.

---

## 6. Manual Testing Override Payload

For manual testing via the AWS CLI, you can pass an event payload that overrides environment variables:

```bash
aws lambda invoke --function-name finops-cleaner-dev \
  --payload '{
    "dry_run": false,
    "excluded_ids": "vol-12345",
    "regions": "us-east-1"
  }' \
  output.json
```

**Note:** These overrides are **ephemeral** and only apply to the specific invocation. They do not change the Lambda's configuration.

---

## 7. Local Development Configuration (`.env` file)

For local development with Floci, create a `.env` file in the project root:

```env
# Local AWS Emulator (Floci)
AWS_ENDPOINT_URL=http://localhost:4566
AWS_ACCESS_KEY_ID=test
AWS_SECRET_ACCESS_KEY=test
AWS_DEFAULT_REGION=us-east-1
AWS_SDK_LOAD_CONFIG=1

# Bot Configuration
DRY_RUN=true
COST_PER_GB=0.08
EXCLUDED_IDS=vol-12345,vol-67890
SNAPSHOTS_TO_KEEP=3
RDS_STOP_AGE_DAYS=7
LOG_LEVEL=debug
SLACK_CHANNEL=#finops-alerts
S3_REPORT_BUCKET=finops-audit-local
S3_REPORT_PREFIX=audit/
TZ=America/New_York
SECRETS_MANAGER_ARN=arn:aws:secretsmanager:us-east-1:123456789012:secret:finops/slack-webhook-local
ENABLE_RDS_SAVINGS=true
ENVIRONMENT=dev

# Secrets (mocked for local development)
SLACK_WEBHOOK_URL=REPLACE_WITH_YOUR_SLACK_WEBHOOK_URL

# SSM Parameters (mocked)
REGIONS=us-east-1,us-west-2
QUARANTINE_DAYS=7
SNAPSHOT_RETENTION_DAYS=30
```

**Security Warning:** The `.env` file contains sensitive values (even if mocked). It **must** be added to `.gitignore`:
```gitignore
# Local development environment variables
.env
.env.local
.env.*.local
```

---

## 8. Environment Variable Naming Convention

| Prefix | Purpose | Example |
| :--- | :--- | :--- |
| (None) | Production bot configuration | `DRY_RUN`, `COST_PER_GB` |
| `LOCAL_` | Local development overrides | `LOCAL_ENDPOINT_URL` |
| `TEST_` | Unit/integration test overrides | `TEST_EXCLUDED_IDS` |

---

## 9. Configuration Change Audit

### 9.1 Lambda Environment Variable Changes

All changes to Lambda environment variables are logged in CloudTrail:
- **Event Name:** `UpdateFunctionConfiguration`
- **Logged Fields:** `Environment.Variables` (shows changed variables)
- **Who:** IAM principal making the change
- **When:** Timestamp of the change
- **Where:** Source IP

### 9.2 SSM Parameter Store Changes

All changes to SSM parameters are logged in CloudTrail:
- **Event Name:** `PutParameter`
- **Logged Fields:** Parameter name, version
- **Who:** IAM principal making the change
- **When:** Timestamp of the change
- **Where:** Source IP

### 9.3 Secrets Manager Changes

All changes to Secrets Manager secrets are logged in CloudTrail:
- **Event Name:** `UpdateSecret`, `PutSecretValue`
- **Logged Fields:** Secret ID, version
- **Who:** IAM principal making the change
- **When:** Timestamp of the change
- **Where:** Source IP

---

## 10. Sign-Off

| Role | Name | Date | Signature |
| :--- | :--- | :--- | :--- |
| **Project Lead / Architect** | Jibrin Ahmed | June 22, 2026 | JA |
| **Security Reviewer** | [Security Team / Imaginary CISO] | [Date] | [Initials] |

---
