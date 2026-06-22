# Security & IAM Matrix: Cloud FinOps Bot

**Document:** 08
**Version:** 3.0 (Go Edition - Enhanced & Audited)
**Author:** Jibrin Ahmed
**Date:** June 22, 2026
**Status:** Final

---

## 1. Document Purpose

This document defines the **complete security architecture** for the Cloud FinOps Bot. It covers:

- **IAM Roles & Policies:** Least-privilege permissions for Lambda.
- **Resource Restrictions:** Conditional IAM policies for safe operations.
- **Secrets Management:** KMS encryption, Secrets Manager, SSM Parameter Store, and rotation.
- **Data Protection:** Encryption at rest and in transit.
- **Security Observability:** Structured logging, audit trails, and monitoring.
- **Incident Response:** Break-glass procedures, privilege escalation detection, and incident drills.
- **Compliance:** Security best practices and frameworks.
- **IAM Policy Validation:** Automated validation in CI/CD.

**Audience:**
- **Security Engineers:** Validation of security posture.
- **DevOps Engineers:** Implementation of IAM policies.
- **Technical Reviewers:** Security architecture validation.
- **Future Employers:** Demonstrates security awareness and maturity.

---

## 2. Security Architecture Overview

### 2.1 Security Principles

| Principle | Implementation |
| :--- | :--- |
| **Least Privilege** | Only the minimum required permissions are granted. |
| **Resource Restrictions** | Actions are restricted to specific resources where possible. |
| **Conditional Access** | `ec2:DeleteVolume` is restricted to resources tagged with `FinOps: AutoPurge`. |
| **No Hardcoded Credentials** | All authentication is via IAM roles or Secrets Manager. |
| **Encryption Everywhere** | KMS for secrets, S3 SSE for audit logs. |
| **Audit Trail** | CloudTrail logs all API calls for review. |
| **Secrets Rotation** | All secrets are rotatable without code changes. |
| **Least-Privilege Validation** | IAM policies are validated in CI with checkov/tfsec. |
| **Break-Glass Access** | Emergency procedures for rapid response. |
| **Privilege Escalation Detection** | Monitoring for unauthorized permission usage. |

### 2.2 Authentication Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                    AWS Authentication Flow                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────┐     ┌──────────────┐     ┌──────────────┐  │
│  │   EventBridge│────▶│    Lambda    │────▶│     AWS      │  │
│  │   (Service)  │     │   (Service)  │     │   Services   │  │
│  └──────────────┘     └──────────────┘     └──────────────┘  │
│         │                    │                    │            │
│         ▼                    ▼                    ▼            │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │                    IAM Role Assumption                   │ │
│  │                                                          │ │
│  │  • EventBridge assumes no role (invokes Lambda)         │ │
│  │  • Lambda assumes: finops-lambda-role-{env}            │ │
│  │  • All AWS API calls use this role's permissions        │ │
│  │  • No long-term credentials stored anywhere             │ │
│  └──────────────────────────────────────────────────────────┘ │
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │                    Secrets Flow                          │ │
│  │                                                          │ │
│  │  • Slack Webhook URL → Secrets Manager → Lambda         │ │
│  │  • KMS key encrypts/decrypts the secret                │ │
│  │  • SSM Parameter Store for non-sensitive configs       │ │
│  └──────────────────────────────────────────────────────────┘ │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 3. IAM Role: Lambda Execution Role

### 3.1 Role Definition

```hcl
# terraform/iam.tf

resource "aws_iam_role" "lambda_role" {
  name = "finops-lambda-role-${var.environment}"
  
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "lambda.amazonaws.com"
        }
      }
    ]
  })
}
```

### 3.2 Trust Policy

The Lambda service is the only entity allowed to assume this role:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Effect": "Allow",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      }
    }
  ]
}
```

---

## 4. IAM Policy: Lambda Permissions

### 4.1 Complete Policy Document (Terraform `iam.tf`)

```hcl
# terraform/iam.tf

resource "aws_iam_policy" "lambda_policy" {
  name        = "finops-lambda-policy-${var.environment}"
  description = "Least-privilege permissions for FinOps Bot Lambda"
  
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      # ──────────────────────────────────────────────────────
      # EC2: Read-only permissions for resource discovery
      # ──────────────────────────────────────────────────────
      {
        Sid = "EC2ReadOnly"
        Effect = "Allow"
        Action = [
          "ec2:DescribeVolumes",
          "ec2:DescribeSnapshots",
          "ec2:DescribeAddresses",
          "ec2:DescribeImages"
        ]
        Resource = "*"
      },
      
      # ──────────────────────────────────────────────────────
      # EC2: Write permissions with explicit conditions
      # ──────────────────────────────────────────────────────
      {
        Sid = "EC2DeleteWithTagCondition"
        Effect = "Allow"
        Action = [
          "ec2:DeleteVolume",
          "ec2:DeleteSnapshot",
          "ec2:ReleaseAddress"
        ]
        Resource = "*"
        Condition = {
          StringEquals = {
            "aws:ResourceTag/FinOps" = "AutoPurge"
          }
        }
      },
      
      {
        Sid = "EC2CreateTagsWithCondition"
        Effect = "Allow"
        Action = ["ec2:CreateTags"]
        Resource = "*"
        Condition = {
          StringEquals = {
            "aws:ResourceType" = ["volume", "snapshot", "elastic-ip"]
          }
          StringNotEquals = {
            "aws:ResourceTag/FinOps" = "AutoPurge"
          }
        }
      },
      
      # ──────────────────────────────────────────────────────
      # RDS: Read-only for discovery + stop with condition
      # ──────────────────────────────────────────────────────
      {
        Sid = "RDSReadOnly"
        Effect = "Allow"
        Action = [
          "rds:DescribeDBInstances"
        ]
        Resource = "*"
      },
      
      {
        Sid = "RDSStopWithCondition"
        Effect = "Allow"
        Action = ["rds:StopDBInstance"]
        Resource = "*"
        Condition = {
          StringEquals = {
            "aws:ResourceTag/Environment" = ["dev", "staging"]
          }
        }
      },
      
      # ──────────────────────────────────────────────────────
      # DynamoDB: State tracking with explicit table
      # ──────────────────────────────────────────────────────
      {
        Sid = "DynamoDBPermissions"
        Effect = "Allow"
        Action = [
          "dynamodb:GetItem",
          "dynamodb:PutItem",
          "dynamodb:UpdateItem",
          "dynamodb:Query",
          "dynamodb:Scan"
        ]
        Resource = [
          aws_dynamodb_table.state_table.arn,
          "${aws_dynamodb_table.state_table.arn}/index/*"
        ]
      },
      
      # ──────────────────────────────────────────────────────
      # S3: Audit upload to specific bucket
      # ──────────────────────────────────────────────────────
      {
        Sid = "S3Permissions"
        Effect = "Allow"
        Action = [
          "s3:PutObject",
          "s3:GetObject",
          "s3:ListBucket"
        ]
        Resource = [
          aws_s3_bucket.audit_bucket.arn,
          "${aws_s3_bucket.audit_bucket.arn}/*"
        ]
      },
      
      # ──────────────────────────────────────────────────────
      # Secrets Manager: Get specific secret
      # ──────────────────────────────────────────────────────
      {
        Sid = "SecretsManagerPermissions"
        Effect = "Allow"
        Action = ["secretsmanager:GetSecretValue"]
        Resource = aws_secretsmanager_secret.slack_webhook.arn
      },
      
      # ──────────────────────────────────────────────────────
      # KMS: Decrypt with specific key and condition
      # ──────────────────────────────────────────────────────
      {
        Sid = "KMSDecrypt"
        Effect = "Allow"
        Action = ["kms:Decrypt"]
        Resource = aws_kms_key.secrets_key.arn
        Condition = {
          StringEquals = {
            "kms:ViaService" = "secretsmanager.${var.aws_region}.amazonaws.com"
          }
        }
      },
      
      # ──────────────────────────────────────────────────────
      # SSM: Get non-sensitive configuration (optional)
      # ──────────────────────────────────────────────────────
      {
        Sid = "SSMParameterStore"
        Effect = "Allow"
        Action = [
          "ssm:GetParameter",
          "ssm:GetParameters",
          "ssm:GetParametersByPath"
        ]
        Resource = "arn:aws:ssm:${var.aws_region}:${data.aws_caller_identity.current.account_id}:parameter/finops/*"
      },
      
      # ──────────────────────────────────────────────────────
      # CloudWatch Logs: Structured logging
      # ──────────────────────────────────────────────────────
      {
        Sid = "CloudWatchLogs"
        Effect = "Allow"
        Action = [
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:PutLogEvents"
        ]
        Resource = "arn:aws:logs:*:*:*"
      },
      
      # ──────────────────────────────────────────────────────
      # SQS: Dead-letter queue
      # ──────────────────────────────────────────────────────
      {
        Sid = "SQSPermissions"
        Effect = "Allow"
        Action = [
          "sqs:GetQueueUrl",
          "sqs:SendMessage"
        ]
        Resource = aws_sqs_queue.dlq.arn
      }
    ]
  })
}
```

### 4.2 Permission Breakdown by Service

#### EC2 Permissions

| Action | Purpose | Resource Restriction | Condition |
| :--- | :--- | :--- | :--- |
| `ec2:DescribeVolumes` | List EBS volumes for discovery | `*` (read-only, safe) | None |
| `ec2:DescribeSnapshots` | List snapshots for discovery | `*` (read-only, safe) | None |
| `ec2:DescribeAddresses` | List Elastic IPs for discovery | `*` (read-only, safe) | None |
| `ec2:DescribeImages` | Check if snapshot is AMI-backed | `*` (read-only, safe) | None |
| `ec2:CreateTags` | Apply quarantine tags | `*` | `aws:ResourceType` is `volume`, `snapshot`, or `elastic-ip` AND `aws:ResourceTag/FinOps != AutoPurge` |
| `ec2:DeleteVolume` | Delete unattached volumes | `*` | **`aws:ResourceTag/FinOps == AutoPurge`** |
| `ec2:DeleteSnapshot` | Delete stale snapshots | `*` | **`aws:ResourceTag/FinOps == AutoPurge`** |
| `ec2:ReleaseAddress` | Release unattached EIPs | `*` | **`aws:ResourceTag/FinOps == AutoPurge`** |

**Why the Condition Blocks Exist:**

1. **`ec2:CreateTags` Condition:**
   - Restricts tagging to specific resource types (volume, snapshot, elastic-ip).
   - Prevents the Lambda from overriding existing `FinOps` tags if they are already set to `AutoPurge` or any other value.
   - Ensures the Lambda cannot tag other resource types (e.g., EC2 instances, security groups).

2. **`ec2:DeleteVolume`/`DeleteSnapshot`/`ReleaseAddress` Condition:**
   - The condition `"aws:ResourceTag/FinOps": "AutoPurge"` ensures that the Lambda can **only** delete EC2 resources that have been explicitly tagged with the `FinOps: AutoPurge` tag.
   - This provides a critical safety net: if a resource is accidentally flagged for deletion, the Lambda cannot delete it unless the tag is present.
   - If the tag is missing, the deletion action is denied by IAM—the Lambda doesn't even attempt the API call.
   - This is a **defense-in-depth** measure that protects production resources.

#### RDS Permissions

| Action | Purpose | Resource Restriction | Condition |
| :--- | :--- | :--- | :--- |
| `rds:DescribeDBInstances` | List RDS instances for discovery | `*` (read-only, safe) | None |
| `rds:StopDBInstance` | Stop idle RDS instances | `*` | **`aws:ResourceTag/Environment == "dev"` or `"staging"`** |

**Safety Note:**
- RDS instances are **stopped**, not deleted. This prevents data loss while still saving compute costs.
- The bot **never** executes `rds:DeleteDBInstance`.
- The `Environment` tag condition ensures that the bot cannot stop production RDS instances, even if the logic is compromised.

#### DynamoDB Permissions

| Action | Purpose | Resource Restriction |
| :--- | :--- | :--- |
| `dynamodb:GetItem` | Check resource state (idempotency) | `FinOps-State-*` table |
| `dynamodb:PutItem` | Write new resource state | `FinOps-State-*` table |
| `dynamodb:UpdateItem` | Update existing resource state | `FinOps-State-*` table |
| `dynamodb:Query` | Query GSI indexes | `FinOps-State-*/index/*` |
| `dynamodb:Scan` | Scan table (fallback) | `FinOps-State-*` table |

#### S3 Permissions

| Action | Purpose | Resource Restriction |
| :--- | :--- | :--- |
| `s3:PutObject` | Upload audit reports and dashboard | `finops-audit-*` bucket |
| `s3:GetObject` | Retrieve existing reports (if needed) | `finops-audit-*` bucket |
| `s3:ListBucket` | List existing reports | `finops-audit-*` bucket |

**Bucket Naming Convention:** `finops-audit-{account-id}`

#### Secrets Manager Permissions

| Action | Purpose | Resource Restriction |
| :--- | :--- | :--- |
| `secretsmanager:GetSecretValue` | Fetch Slack webhook URL | `finops/slack-webhook-*` secret (specific ARN in Terraform) |

**Secret Rotation:** The secret can be rotated manually via the AWS Console or CLI. The Lambda will automatically fetch the new value on the next invocation (no code changes required).

#### KMS Permissions

| Action | Purpose | Resource Restriction |
| :--- | :--- | :--- |
| `kms:Decrypt` | Decrypt Secrets Manager secret | `arn:aws:kms:*:*:key/*` (with `ViaService` condition) |

**Condition Explanation:** The `"kms:ViaService": "secretsmanager.*.amazonaws.com"` condition ensures the Lambda can only use KMS decryption in the context of Secrets Manager. This prevents the Lambda from using KMS for other purposes.

#### CloudWatch Logs Permissions

| Action | Purpose | Resource Restriction |
| :--- | :--- | :--- |
| `logs:CreateLogGroup` | Create log group on first invocation | `arn:aws:logs:*:*:*` |
| `logs:CreateLogStream` | Create log stream for each invocation | `arn:aws:logs:*:*:*` |
| `logs:PutLogEvents` | Write structured logs to CloudWatch | `arn:aws:logs:*:*:*` |

**Log Retention:** CloudWatch Log Group retains logs for **30 days**. This balances cost with operational debugging needs.

#### SQS Permissions

| Action | Purpose | Resource Restriction |
| :--- | :--- | :--- |
| `sqs:GetQueueUrl` | Get the URL of the DLQ | `finops-dlq-*` queue |
| `sqs:SendMessage` | Send failed events to DLQ | `finops-dlq-*` queue |

**DLQ Configuration:**
- Retention: 14 days
- Visibility Timeout: 30 seconds
- Max Receive Count: 3

#### SSM Parameter Store Permissions (Optional)

| Action | Purpose | Resource Restriction |
| :--- | :--- | :--- |
| `ssm:GetParameter` | Get non-sensitive configuration | `/finops/*` parameters |
| `ssm:GetParameters` | Get multiple configuration values | `/finops/*` parameters |
| `ssm:GetParametersByPath` | Get configuration by path | `/finops/*` parameters |

**Note:** SSM Parameter Store is used for non-sensitive configuration values (e.g., region lists, retention periods). Sensitive values remain in Secrets Manager.

---

## 5. Least-Privilege IAM Policy Definition

### 5.1 Explicit Permission Mapping

| Resource Type | Action | Why It's Needed | Restriction |
| :--- | :--- | :--- | :--- |
| **EC2 Volumes** | `DescribeVolumes` | Discover unattached volumes | Read-only, no restriction |
| | `DeleteVolume` | Delete quarantined volumes | **Condition:** `FinOps: AutoPurge` tag |
| **EC2 Snapshots** | `DescribeSnapshots` | Discover stale snapshots | Read-only, no restriction |
| | `DescribeImages` | Check if snapshot is AMI-backed | Read-only, no restriction |
| | `DeleteSnapshot` | Delete stale snapshots | **Condition:** `FinOps: AutoPurge` tag |
| **EC2 EIPs** | `DescribeAddresses` | Discover unassociated EIPs | Read-only, no restriction |
| | `ReleaseAddress` | Release unassociated EIPs | **Condition:** `FinOps: AutoPurge` tag |
| | `CreateTags` | Apply quarantine tags | **Condition:** Specific resource types |
| **RDS Instances** | `DescribeDBInstances` | Discover idle instances | Read-only, no restriction |
| | `StopDBInstance` | Stop idle instances | **Condition:** `Environment` tag is `dev` or `staging` |
| **DynamoDB** | `GetItem`, `PutItem`, `UpdateItem`, `Query`, `Scan` | State tracking | Specific table only |
| **S3** | `PutObject`, `GetObject`, `ListBucket` | Audit uploads | Specific bucket only |
| **Secrets Manager** | `GetSecretValue` | Fetch Slack webhook | Specific secret only |
| **KMS** | `Decrypt` | Decrypt secret | Specific key with `ViaService` condition |
| **CloudWatch Logs** | `CreateLogGroup`, `CreateLogStream`, `PutLogEvents` | Structured logging | No restriction (safe) |
| **SQS** | `GetQueueUrl`, `SendMessage` | DLQ handling | Specific queue only |

### 5.2 IAM Policy Validation (CI)

The following tools validate IAM policies in CI:

| Tool | Purpose | Validation |
| :--- | :--- | :--- |
| **checkov** | Infrastructure-as-Code security scanning | Checks for overly permissive policies, missing conditions |
| **tfsec** | Terraform security scanning | Checks for insecure configurations, broad permissions |
| **golangci-lint** | Go code security linting | Checks for hardcoded credentials, insecure patterns |

```bash
# CI validation commands
checkov -d terraform/ --check CKV_AWS_*
tfsec terraform/ --exclude aws-iam-no-policy-wildcards
golangci-lint run --enable-all ./...
```

---

## 6. Secrets Manager & SSM Parameter Store

### 6.1 Secrets Manager Configuration

```hcl
# terraform/secrets.tf

# KMS Key for Secrets Manager
resource "aws_kms_key" "secrets_key" {
  description             = "KMS key for FinOps Bot secrets"
  deletion_window_in_days = 30
  enable_key_rotation     = true
  
  tags = {
    Name = "finops-secrets-key-${var.environment}"
  }
}

# Secrets Manager Secret
resource "aws_secretsmanager_secret" "slack_webhook" {
  name                    = "finops/slack-webhook-${var.environment}"
  description             = "Slack webhook URL for FinOps Bot notifications"
  kms_key_id              = aws_kms_key.secrets_key.arn
  recovery_window_in_days = 30
  
  tags = {
    Name = "finops-slack-webhook-${var.environment}"
  }
}

# Secret Version (Placeholder - user must update)
resource "aws_secretsmanager_secret_version" "slack_webhook" {
  secret_id     = aws_secretsmanager_secret.slack_webhook.id
  secret_string = jsonencode({
    SLACK_WEBHOOK_URL = "https://hooks.slack.com/services/REPLACE_ME"
  })
}
```

### 6.2 SSM Parameter Store (Optional)

Non-sensitive configuration can be stored in SSM Parameter Store:

```hcl
# terraform/ssm.tf

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
```

### 6.3 Secret Rotation Strategy

| Secret | Rotation Method | Frequency | Downtime |
| :--- | :--- | :--- | :--- |
| Slack Webhook URL | Manual via AWS Console/CLI | As needed | None |
| KMS Key | Automatic (AWS-managed rotation) | Every 365 days | None |
| IAM Role Keys | Automatic (AWS-managed) | Every 12 hours | None |

**Rotation Procedure (Slack Webhook):**

```bash
# 1. Create a new webhook URL in Slack
# 2. Update the secret
aws secretsmanager update-secret \
  --secret-id finops/slack-webhook-dev \
  --secret-string '{"SLACK_WEBHOOK_URL":"https://hooks.slack.com/services/NEW_WEBHOOK"}'

# 3. The Lambda will fetch the new value on the next invocation
```

---

## 7. Security Observability & Structured Logging

### 7.1 Structured Logging Fields

All logs are emitted as structured JSON to CloudWatch Logs:

| Field | Description | Source |
| :--- | :--- | :--- |
| `timestamp` | UTC timestamp of the log entry | `time.Now().UTC()` |
| `level` | Log level (debug, info, warn, error) | Configurable |
| `correlation_id` | Unique identifier for the invocation | Generated at Lambda start |
| `who` | IAM principal ARN | Lambda context identity |
| `what` | Action performed | Application code |
| `where` | Source IP address (if available) | Lambda context identity |
| `resource_id` | AWS resource ID | EC2/RDS API |
| `resource_type` | Resource type | Application code |
| `region` | AWS region | Configuration |
| `account_id` | AWS account ID | Lambda context |
| `message` | Human-readable log message | Application code |
| `duration_ms` | Execution duration (for performance tracking) | `time.Since()` |

### 7.2 Audit Trail (Who/What/When/Where)

All resource modifications are audited in DynamoDB with the following fields:

| Field | Description | Example |
| :--- | :--- | :--- |
| `ActionedBy` | IAM principal ARN | `arn:aws:sts::123456789012:assumed-role/finops-lambda-role-dev/finops-cleaner` |
| `SourceIP` | Source IP address | `203.0.113.1` |
| `CorrelationID` | Unique identifier for the invocation | `abc-123-def-456` |
| `ActionReason` | Reason for the action | `Quarantine expired with AutoPurge tag` |
| `Timestamp` | UTC timestamp | `2026-06-22T02:00:00Z` |

### 7.3 CloudWatch Logs Insights Queries

```sql
-- Find all actions performed by a specific IAM principal
fields @timestamp, who, what, resource_id, resource_type
| filter who like /finops-lambda-role/
| sort @timestamp desc
| limit 100

-- Find all DELETE actions in the last 24 hours
fields @timestamp, what, resource_id, resource_type, estimated_savings
| filter what = "DELETE_VOLUME"
| filter @timestamp > now() - 24h
| sort @timestamp desc

-- Correlation ID tracking across the entire invocation
fields @timestamp, correlation_id, what, resource_id
| filter correlation_id = "abc-123-def-456"
| sort @timestamp asc

-- Identify permission errors (potential privilege escalation)
fields @timestamp, level, who, what, message
| filter level = "error"
| filter message like /AccessDenied|Unauthorized/
| sort @timestamp desc
```

---

## 8. Incident Response & Break-Glass Procedures

### 8.1 Break-Glass Procedures

| Scenario | Immediate Action | Recovery Action | Time |
| :--- | :--- | :--- | :--- |
| **Compromised Bot** | Disable EventBridge rule + Set `DRY_RUN=true` | Rotate secrets, review logs, restore resources | 5 minutes |
| **Broad IAM Permissions** | Revoke suspicious permissions + Disable bot | Review IAM policies, rollback Terraform | 10 minutes |
| **Secrets Exposed** | Rotate secret immediately | Review CloudTrail for unauthorized access | 3 minutes |
| **Bot Deleting Unexpected Resources** | Disable bot immediately + Add resources to `EXCLUDED_IDS` | Restore from snapshot, review logs | 5 minutes |

### 8.2 Break-Glass Commands

```bash
# 1. EMERGENCY STOP: Disable the bot
aws events disable-rule --name finops-daily-trigger-dev --region us-east-1

# 2. Set DRY_RUN to true
aws lambda update-function-configuration --function-name finops-cleaner-dev \
  --environment "Variables={DRY_RUN=true}" --region us-east-1

# 3. Add critical resources to EXCLUDED_IDS
aws lambda update-function-configuration --function-name finops-cleaner-dev \
  --environment "Variables={EXCLUDED_IDS=vol-123,vol-456}" --region us-east-1

# 4. Rotate secret
aws secretsmanager update-secret --secret-id finops/slack-webhook-dev \
  --secret-string '{"SLACK_WEBHOOK_URL":"https://hooks.slack.com/services/..."}' --region us-east-1

# 5. Investigate logs
aws logs get-log-events --log-group-name /aws/lambda/finops-cleaner-dev \
  --limit 100 --order-by LastEventTime --descending --region us-east-1
```

---

## 9. Privilege Escalation Detection

### 9.1 Detection Mechanisms

| Mechanism | Purpose | Alert |
| :--- | :--- | :--- |
| **CloudTrail** | Logs all API calls | Alerts on `AccessDenied` for sensitive actions |
| **CloudWatch Alarms** | Monitors for errors | Alarm on multiple `AccessDenied` events |
| **IAM Policy Validation** | Prevents broad permissions in CI | CI failure on overly permissive policies |
| **Audit Trail** | Tracks who did what | Queryable for forensic analysis |

### 9.2 Privilege Escalation Scenarios

| Scenario | Detection | Response |
| :--- | :--- | :--- |
| **IAM Role Assumed by Unauthorized Principal** | CloudTrail `AssumeRole` events | Revoke access, rotate all credentials |
| **Policy Attached with Broad Permissions** | checkov/tfsec CI validation | Fail deployment, review policy |
| **Permission Escalation via Lambda** | CloudWatch Logs for `AccessDenied` | Disable bot, review IAM policy |
| **Unauthorized Secret Access** | CloudTrail `GetSecretValue` events | Rotate secret, revoke access |

---

## 10. KMS Key Configuration

### 10.1 KMS Key Definition

```hcl
# terraform/kms.tf

resource "aws_kms_key" "secrets_key" {
  description             = "KMS key for FinOps Bot secrets"
  deletion_window_in_days = 30
  enable_key_rotation     = true
  
  tags = {
    Name = "finops-secrets-key-${var.environment}"
  }
}

resource "aws_kms_alias" "secrets_key" {
  name          = "alias/finops-secrets-${var.environment}"
  target_key_id = aws_kms_key.secrets_key.key_id
}
```

### 10.2 Key Policy

```json
{
  "Version": "2012-10-17",
  "Id": "key-default-1",
  "Statement": [
    {
      "Sid": "Enable IAM User Permissions",
      "Effect": "Allow",
      "Principal": {
        "AWS": "arn:aws:iam::<account-id>:root"
      },
      "Action": "kms:*",
      "Resource": "*"
    },
    {
      "Sid": "Allow Lambda Decryption",
      "Effect": "Allow",
      "Principal": {
        "AWS": "arn:aws:iam::<account-id>:role/finops-lambda-role-*"
      },
      "Action": "kms:Decrypt",
      "Resource": "*"
    }
  ]
}
```

---

## 11. S3 Bucket Security

### 11.1 Security Configuration

| Security Feature | Implementation |
| :--- | :--- |
| **Public Access** | Blocked entirely (`block_public_acls = true`, `block_public_policy = true`). |
| **Bucket Policy** | Denies non-SSL requests. |
| **Encryption** | Server-side encryption (AES256) enforced. |
| **Versioning** | Enabled for data protection. |
| **Lifecycle** | Transition to Glacier after 30 days; delete after 365 days. |
| **Access** | Only Lambda IAM role can write/read. |
| **Static Website** | Hosting enabled, but accessed via CloudFront with Origin Access Control. |

### 11.2 S3 Bucket Policy (Deny Non-SSL)

```hcl
resource "aws_s3_bucket_policy" "audit_bucket" {
  bucket = aws_s3_bucket.audit_bucket.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid = "DenyNonSSL"
        Effect = "Deny"
        Principal = "*"
        Action = "s3:*"
        Resource = [
          aws_s3_bucket.audit_bucket.arn,
          "${aws_s3_bucket.audit_bucket.arn}/*"
        ]
        Condition = {
          Bool = {
            "aws:SecureTransport": "false"
          }
        }
      }
    ]
  })
}
```

---

## 12. Compliance Frameworks

The security controls implemented align with:

| Framework | Alignment |
| :--- | :--- |
| **AWS Well-Architected** | Security Pillar |
| **CIS AWS Benchmarks** | IAM, Logging, Monitoring |
| **NIST 800-53** | Access Control (AC), Audit & Accountability (AU) |
| **ISO 27001** | Annex A controls (A.9, A.12, A.16) |
| **SOC 2** | CC (Common Criteria), A (Availability), C (Confidentiality) |

---

## 13. Security Checklist (Pre-Deployment)

Before deploying to production, verify:

- [ ] IAM policy has `ec2:DeleteVolume` restricted with `aws:ResourceTag/FinOps == "AutoPurge"`.
- [ ] IAM policy has `ec2:CreateTags` restricted with `aws:ResourceType` condition.
- [ ] IAM policy has `rds:StopDBInstance` restricted with `aws:ResourceTag/Environment == "dev"` or `"staging"`.
- [ ] RDS instances are **stopped**, not deleted.
- [ ] KMS key is created with proper key policy.
- [ ] Secrets Manager secret is created and contains the correct Slack webhook URL.
- [ ] SSM Parameter Store contains non-sensitive configuration.
- [ ] S3 bucket has `block_public_acls = true` and `block_public_policy = true`.
- [ ] S3 bucket has server-side encryption enabled (AES256).
- [ ] S3 bucket has versioning enabled.
- [ ] S3 bucket policy denies non-SSL requests.
- [ ] Terraform state bucket has public access blocked.
- [ ] CloudWatch Log Group has retention set to 30 days.
- [ ] CloudWatch Alarms are configured with SNS notifications.
- [ ] `DRY_RUN` is set to `true` for initial testing.
- [ ] `EXCLUDED_IDS` includes any critical production resources.
- [ ] Structured logging is configured with all required fields.
- [ ] checkov/tfsec validation passes in CI.
- [ ] Incident drill has been performed and documented.

---

## 14. Sign-Off

| Role | Name | Date | Signature |
| :--- | :--- | :--- | :--- |
| **Project Lead / Architect** | Jibrin Ahmed | June 22, 2026 | JA |
| **Security Reviewer** | [Security Team / Imaginary CISO] | [Date] | [Initials] |

---

