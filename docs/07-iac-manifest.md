# Infrastructure as Code (IaC) Manifest: Cloud FinOps Bot

**Document:** 07
**Version:** 3.1 (Go Edition - Fully Patched)
**Author:** Jibrin Ahmed
**Date:** June 22, 2026
**Status:** Final

---

## 1. Document Purpose

This document defines the **complete Terraform configuration** for the Cloud FinOps Bot. It covers:

- **AWS Provider Configuration:** Region, authentication, and backend setup.
- **Resource Inventory:** All AWS resources required for the bot.
- **Variable Definitions:** Inputs for customization with validations.
- **IAM Least-Privilege:** Explicit policies with conditions and resource restrictions.
- **Secrets Management:** KMS, Secrets Manager, and SSM Parameter Store.
- **Security Observability:** Structured logging and audit trail support.
- **Health Check:** Dedicated Lambda for health monitoring.
- **SLO Monitoring:** CloudWatch dashboards and alarms.
- **Data Retention:** Lifecycle policies for S3 and CloudWatch.
- **Outputs:** Useful values after deployment.
- **Backend Configuration:** Remote state storage with locking.
- **Build Automation:** Automatic Go binary compilation before deployment.

**Audience:**
- **DevOps Engineers:** Deployment and operations.
- **Security Engineers:** Validation of infrastructure security.
- **Technical Reviewers:** Infrastructure architecture validation.
- **Future Employers:** Demonstrates IaC expertise and security awareness.

---

## 2. Resource Inventory

The following AWS resources will be provisioned by Terraform:

| Resource Type | Resource Name | Purpose |
| :--- | :--- | :--- |
| **KMS Key** | `finops-secrets-key` | Encryption key for Secrets Manager. |
| **Secrets Manager Secret** | `finops/slack-webhook` | Stores the Slack webhook URL. |
| **SSM Parameters** | `/finops/*` | Non-sensitive configuration values. |
| **IAM Role (Main)** | `finops-lambda-role` | Lambda execution role for main bot. |
| **IAM Role (Health)** | `finops-health-role` | Lambda execution role for health check. |
| **IAM Policy** | `finops-lambda-policy` | Least-privilege permissions for Lambda. |
| **Lambda Function (Main)** | `finops-cleaner` | Go-based FinOps Bot. |
| **Lambda Function (Health)** | `finops-health-check` | Health check Lambda. |
| **Lambda Permission (Main)** | `finops-lambda-permission` | Allows EventBridge to invoke the Lambda. |
| **Lambda Permission (Health)** | `finops-health-permission` | Allows EventBridge to invoke health check. |
| **EventBridge Rule (Main)** | `finops-daily-trigger` | Schedules the bot daily at 2:00 AM. |
| **EventBridge Rule (Health)** | `finops-health-trigger` | Schedules health check every 5 minutes. |
| **EventBridge Target (Main)** | `finops-lambda-target` | Targets the Lambda from the EventBridge rule. |
| **EventBridge Target (Health)** | `finops-health-target` | Targets health check Lambda. |
| **DynamoDB Table** | `FinOps-State` | State tracking with audit fields. |
| **S3 Bucket** | `finops-audit-<env>-<account-id>` | Audit logs and static HTML dashboard. |
| **S3 Bucket Public Access Block** | `finops-audit-block` | Blocks public access to the S3 bucket. |
| **S3 Bucket Ownership Controls** | `finops-audit-ownership` | Enforces bucket ownership. |
| **S3 Bucket Encryption** | `finops-audit-encryption` | Server-side encryption for audit logs. |
| **S3 Bucket Versioning** | `finops-audit-versioning` | Enables versioning for data protection. |
| **S3 Bucket Website Configuration** | `finops-audit-website` | Enables static website hosting. |
| **S3 Bucket Lifecycle Configuration** | `finops-audit-lifecycle` | Lifecycle rules for cost optimization. |
| **S3 Bucket CORS Configuration** | `finops-audit-cors` | CORS rules for dashboard access. |
| **SQS Queue** | `finops-dlq` | Dead-letter queue for failed Lambda invocations. |
| **CloudWatch Log Group (Main)** | `/aws/lambda/finops-cleaner` | Lambda execution logs (structured). |
| **CloudWatch Log Group (Health)** | `/aws/lambda/finops-health-check` | Health check logs. |
| **CloudWatch Dashboard** | `FinOps-Bot-Dashboard` | Operational dashboard. |
| **CloudWatch Alarm (Errors)** | `finops-lambda-error-alarm` | Alerts on Lambda errors. |
| **CloudWatch Alarm (Duration)** | `finops-lambda-duration-alarm` | Alerts on Lambda duration. |
| **CloudWatch Alarm (DLQ)** | `finops-dlq-alarm` | Alerts on DLQ depth. |
| **CloudWatch Alarm (Health)** | `finops-health-check-alarm` | Alerts on health check failure. |
| **CloudWatch Alarm (Invocations)** | `finops-invocation-alarm` | Alerts on low invocation count. |
| **CloudWatch Alarm (Throttles)** | `finops-throttle-alarm` | Alerts on Lambda throttles. |
| **SNS Topic** | `finops-alerts` | Notification topic for alarms. |
| **SNS Topic Subscription** | `finops-alerts-email` | Email subscription for alerts. |
| **Terraform Backend (Manual)** | `s3://finops-terraform-state-<account-id>` | Remote state storage. |
| **Terraform Backend (Manual)** | `dynamodb://terraform-locks` | State locking. |

---

## 3. Terraform Configuration

### 3.1 Provider Configuration (`terraform/main.tf`)

```hcl
# terraform/main.tf

terraform {
  required_version = ">= 1.5.0"
  
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    archive = {
      source  = "hashicorp/archive"
      version = "~> 2.4"
    }
    null = {
      source  = "hashicorp/null"
      version = "~> 3.2"
    }
  }
  
  backend "s3" {
    bucket         = "finops-terraform-state-123456789012" # Replace with your account ID
    key            = "finops-bot/terraform.tfstate"
    region         = "us-east-1"
    dynamodb_table = "terraform-locks"
    encrypt        = true
  }
}

provider "aws" {
  region = var.aws_region
  
  default_tags {
    tags = {
      Project     = "finops-bot"
      Environment = var.environment
      ManagedBy   = "terraform"
      SecurityLevel = "high"
    }
  }
}

# Get current account ID for dynamic resource naming
data "aws_caller_identity" "current" {}
```

---

### 3.2 Variables (`terraform/variables.tf`)

```hcl
# terraform/variables.tf

variable "aws_region" {
  description = "AWS region for deployment"
  type        = string
  default     = "us-east-1"
}

variable "environment" {
  description = "Environment name (dev, staging, prod)"
  type        = string
  default     = "dev"
  
  validation {
    condition     = contains(["dev", "staging", "prod"], var.environment)
    error_message = "Environment must be dev, staging, or prod."
  }
}

variable "dry_run" {
  description = "Enable dry-run mode (true = no deletions)"
  type        = bool
  default     = true
}

variable "cost_per_gb" {
  description = "Cost per GB-month for EBS volumes"
  type        = number
  default     = 0.08
}

variable "excluded_ids" {
  description = "Comma-separated list of resource IDs to exclude"
  type        = string
  default     = ""
}

variable "regions" {
  description = "Comma-separated list of AWS regions to scan"
  type        = string
  default     = "us-east-1,us-west-2,eu-west-1"
}

variable "quarantine_days" {
  description = "Number of days to quarantine resources before deletion"
  type        = number
  default     = 7
  validation {
    condition     = var.quarantine_days >= 1
    error_message = "Quarantine days must be at least 1."
  }
}

variable "snapshot_retention_days" {
  description = "Number of days to retain snapshots"
  type        = number
  default     = 30
  validation {
    condition     = var.snapshot_retention_days >= 1
    error_message = "Snapshot retention days must be at least 1."
  }
}

variable "snapshots_to_keep" {
  description = "Number of most recent snapshots to keep per volume"
  type        = number
  default     = 3
  validation {
    condition     = var.snapshots_to_keep >= 0
    error_message = "Snapshots to keep must be 0 or greater."
  }
}

variable "rds_stop_age_days" {
  description = "Minimum age in days for RDS instances to be considered for stopping"
  type        = number
  default     = 7
  validation {
    condition     = var.rds_stop_age_days >= 1
    error_message = "RDS stop age days must be at least 1."
  }
}

variable "log_level" {
  description = "Logging verbosity"
  type        = string
  default     = "info"
  validation {
    condition     = contains(["debug", "info", "warn", "error"], var.log_level)
    error_message = "Log level must be one of: debug, info, warn, error."
  }
}

variable "slack_channel" {
  description = "Slack channel for notifications"
  type        = string
  default     = ""
}

variable "s3_report_bucket" {
  description = "S3 bucket for audit reports and dashboard"
  type        = string
  default     = ""
  validation {
    condition     = can(regex("^[a-z0-9-]+$", var.s3_report_bucket)) || var.s3_report_bucket == ""
    error_message = "S3 bucket name must contain only lowercase letters, numbers, and hyphens."
  }
}

variable "s3_report_prefix" {
  description = "Prefix for S3 objects"
  type        = string
  default     = "audit/"
}

variable "timezone" {
  description = "Timezone for logging and reports"
  type        = string
  default     = "UTC"
}

variable "secrets_manager_arn" {
  description = "ARN of the Secrets Manager secret containing Slack webhook URL"
  type        = string
  default     = ""
  validation {
    condition     = can(regex("^arn:aws:secretsmanager:", var.secrets_manager_arn)) || var.secrets_manager_arn == ""
    error_message = "Secrets Manager ARN must be a valid ARN starting with 'arn:aws:secretsmanager:'."
  }
}

variable "sns_topic_arn" {
  description = "ARN of the SNS topic for alerts"
  type        = string
  default     = ""
  validation {
    condition     = can(regex("^arn:aws:sns:", var.sns_topic_arn)) || var.sns_topic_arn == ""
    error_message = "SNS topic ARN must be a valid ARN starting with 'arn:aws:sns:'."
  }
}

variable "enable_rds_savings" {
  description = "Enable RDS savings calculations and stopping"
  type        = bool
  default     = true
}

variable "lambda_memory_size" {
  description = "Memory size for Lambda function in MB"
  type        = number
  default     = 256
  validation {
    condition     = contains([128, 256, 512, 1024, 2048, 4096, 6144, 8192, 10240], var.lambda_memory_size)
    error_message = "Lambda memory size must be one of the supported values: 128, 256, 512, 1024, 2048, 4096, 6144, 8192, 10240."
  }
}

variable "lambda_timeout" {
  description = "Timeout for Lambda function in seconds"
  type        = number
  default     = 300
  validation {
    condition     = var.lambda_timeout >= 1 && var.lambda_timeout <= 900
    error_message = "Lambda timeout must be between 1 and 900 seconds."
  }
}

variable "health_check_interval" {
  description = "Interval for health check in minutes"
  type        = number
  default     = 5
}
```

---

### 3.3 KMS Key (`terraform/kms.tf`)

```hcl
# terraform/kms.tf

# KMS Key for Secrets Manager
resource "aws_kms_key" "secrets_key" {
  description             = "KMS key for FinOps Bot secrets"
  deletion_window_in_days = 30
  enable_key_rotation     = true
  
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid = "Enable IAM User Permissions"
        Effect = "Allow"
        Principal = {
          AWS = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:root"
        }
        Action = "kms:*"
        Resource = "*"
      },
      {
        Sid = "Allow Lambda Decryption"
        Effect = "Allow"
        Principal = {
          AWS = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:role/finops-lambda-role-${var.environment}"
        }
        Action = "kms:Decrypt"
        Resource = "*"
      }
    ]
  })
  
  tags = {
    Name = "finops-secrets-key-${var.environment}"
  }
}

# KMS Key Alias
resource "aws_kms_alias" "secrets_key" {
  name          = "alias/finops-secrets-${var.environment}"
  target_key_id = aws_kms_key.secrets_key.key_id
}
```

---

### 3.4 Secrets Manager & SSM (`terraform/secrets.tf`)

```hcl
# terraform/secrets.tf

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

# SSM Parameter Store - Non-sensitive configuration
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

---

### 3.5 IAM Role and Policy (`terraform/iam.tf`)

```hcl
# terraform/iam.tf

# IAM Role for Main Lambda
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

# IAM Role for Health Check Lambda
resource "aws_iam_role" "health_role" {
  name = "finops-health-role-${var.environment}"
  
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

# IAM Policy - Least Privilege (Main Lambda)
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
      # SSM: Get non-sensitive configuration
      # ──────────────────────────────────────────────────────
      {
        Sid = "SSMParameterStore"
        Effect = "Allow"
        Action = [
          "ssm:GetParameter",
          "ssm:GetParameters",
          "ssm:GetParametersByPath"
        ]
        Resource = "arn:aws:ssm:${var.aws_region}:${data.aws_caller_identity.current.account_id}:parameter/finops/${var.environment}/*"
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
      # CloudWatch Metrics: Health check metrics
      # ──────────────────────────────────────────────────────
      {
        Sid = "CloudWatchMetrics"
        Effect = "Allow"
        Action = [
          "cloudwatch:PutMetricData"
        ]
        Resource = "*"
        Condition = {
          StringEquals = {
            "cloudwatch:namespace" = "FinOpsBot"
          }
        }
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

# IAM Policy - Health Check Lambda (Minimal)
resource "aws_iam_policy" "health_policy" {
  name        = "finops-health-policy-${var.environment}"
  description = "Minimal permissions for FinOps Bot Health Check"
  
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid = "DynamoDBRead"
        Effect = "Allow"
        Action = ["dynamodb:GetItem"]
        Resource = aws_dynamodb_table.state_table.arn
      },
      {
        Sid = "S3Read"
        Effect = "Allow"
        Action = ["s3:HeadBucket"]
        Resource = aws_s3_bucket.audit_bucket.arn
      },
      {
        Sid = "SecretsManagerRead"
        Effect = "Allow"
        Action = ["secretsmanager:GetSecretValue"]
        Resource = aws_secretsmanager_secret.slack_webhook.arn
      },
      {
        Sid = "SSMRead"
        Effect = "Allow"
        Action = ["ssm:GetParameter"]
        Resource = "arn:aws:ssm:${var.aws_region}:${data.aws_caller_identity.current.account_id}:parameter/finops/${var.environment}/*"
      },
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
      {
        Sid = "CloudWatchMetrics"
        Effect = "Allow"
        Action = ["cloudwatch:PutMetricData"]
        Resource = "*"
        Condition = {
          StringEquals = {
            "cloudwatch:namespace" = "FinOpsBot"
          }
        }
      }
    ]
  })
}

# Attach the policies to the roles
resource "aws_iam_role_policy_attachment" "lambda_policy_attachment" {
  role       = aws_iam_role.lambda_role.name
  policy_arn = aws_iam_policy.lambda_policy.arn
}

resource "aws_iam_role_policy_attachment" "health_policy_attachment" {
  role       = aws_iam_role.health_role.name
  policy_arn = aws_iam_policy.health_policy.arn
}
```

---

### 3.6 Lambda Functions (`terraform/lambda.tf`)

```hcl
# terraform/lambda.tf

# ──────────────────────────────────────────────────────────────
# Main Lambda Function
# ──────────────────────────────────────────────────────────────

# Build the Go binary before packaging
resource "null_resource" "build_lambda" {
  triggers = {
    always_run = timestamp()
  }
  
  provisioner "local-exec" {
    command = "GOOS=linux GOARCH=amd64 go build -o ../bootstrap ../cmd/main.go"
    working_dir = path.module
  }
}

# Archive the Lambda code
data "archive_file" "lambda_zip" {
  depends_on = [null_resource.build_lambda]
  type        = "zip"
  source_file = "../bootstrap"
  output_path = "lambda.zip"
}

# Main Lambda Function
resource "aws_lambda_function" "finops_cleaner" {
  depends_on = [
    aws_iam_role_policy_attachment.lambda_policy_attachment,
    aws_cloudwatch_log_group.lambda_logs
  ]
  
  filename         = data.archive_file.lambda_zip.output_path
  function_name    = "finops-cleaner-${var.environment}"
  role             = aws_iam_role.lambda_role.arn
  handler          = "bootstrap" # Custom runtime
  runtime          = "provided.al2"
  timeout          = var.lambda_timeout
  memory_size      = var.lambda_memory_size
  
  # NEW: Reserved concurrency to prevent race conditions
  reserved_concurrent_executions = 1
  
  environment {
    variables = {
      DRY_RUN                 = tostring(var.dry_run)
      COST_PER_GB             = tostring(var.cost_per_gb)
      EXCLUDED_IDS            = var.excluded_ids
      REGIONS                 = var.regions
      QUARANTINE_DAYS         = tostring(var.quarantine_days)
      SNAPSHOT_RETENTION_DAYS = tostring(var.snapshot_retention_days)
      SNAPSHOTS_TO_KEEP       = tostring(var.snapshots_to_keep)
      RDS_STOP_AGE_DAYS       = tostring(var.rds_stop_age_days)
      LOG_LEVEL               = var.log_level
      SLACK_CHANNEL           = var.slack_channel
      S3_REPORT_BUCKET        = aws_s3_bucket.audit_bucket.bucket
      S3_REPORT_PREFIX        = var.s3_report_prefix
      TZ                      = var.timezone
      SECRETS_MANAGER_ARN     = aws_secretsmanager_secret.slack_webhook.arn
      ENABLE_RDS_SAVINGS      = tostring(var.enable_rds_savings)
      ENVIRONMENT             = var.environment
    }
  }
  
  tracing_config {
    mode = "PassThrough"
  }
  
  tags = {
    Name = "finops-cleaner-${var.environment}"
  }
}

# ──────────────────────────────────────────────────────────────
# Health Check Lambda Function (NEW)
# ──────────────────────────────────────────────────────────────

# Build health check binary
resource "null_resource" "build_health" {
  triggers = {
    always_run = timestamp()
  }
  
  provisioner "local-exec" {
    command = "GOOS=linux GOARCH=amd64 go build -o ../health_bootstrap ../cmd/health_check.go"
    working_dir = path.module
  }
}

data "archive_file" "health_zip" {
  depends_on = [null_resource.build_health]
  type        = "zip"
  source_file = "../health_bootstrap"
  output_path = "health.zip"
}

# Health Check Lambda
resource "aws_lambda_function" "health_check" {
  depends_on = [
    aws_iam_role_policy_attachment.health_policy_attachment,
    aws_cloudwatch_log_group.health_logs
  ]
  
  filename         = data.archive_file.health_zip.output_path
  function_name    = "finops-health-check-${var.environment}"
  role             = aws_iam_role.health_role.arn
  handler          = "bootstrap"
  runtime          = "provided.al2"
  timeout          = 30
  memory_size      = 128
  reserved_concurrent_executions = 1
  
  environment {
    variables = {
      LOG_LEVEL           = var.log_level
      ENVIRONMENT         = var.environment
      SECRETS_MANAGER_ARN = aws_secretsmanager_secret.slack_webhook.arn
      S3_REPORT_BUCKET    = aws_s3_bucket.audit_bucket.bucket
    }
  }
  
  tags = {
    Name = "finops-health-check-${var.environment}"
  }
}

# ──────────────────────────────────────────────────────────────
# Lambda Permissions
# ──────────────────────────────────────────────────────────────

# Main Lambda Permission for EventBridge
resource "aws_lambda_permission" "eventbridge_invoke" {
  statement_id  = "AllowEventBridgeInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.finops_cleaner.function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.daily_trigger.arn
}

# Health Check Lambda Permission for EventBridge
resource "aws_lambda_permission" "health_invoke" {
  statement_id  = "AllowEventBridgeHealthInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.health_check.function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.health_trigger.arn
}

# Main Lambda Dead-Letter Queue Configuration
resource "aws_lambda_function_event_invoke_config" "finops_cleaner" {
  function_name = aws_lambda_function.finops_cleaner.function_name
  
  destination_config {
    on_failure {
      destination = aws_sqs_queue.dlq.arn
    }
  }
  
  maximum_event_age_in_seconds = 3600
  maximum_retry_attempts       = 2
}
```

---

### 3.7 EventBridge Rules (`terraform/eventbridge.tf`)

```hcl
# terraform/eventbridge.tf

# ──────────────────────────────────────────────────────────────
# Main Bot Schedule - Daily at 2:00 AM
# ──────────────────────────────────────────────────────────────

resource "aws_cloudwatch_event_rule" "daily_trigger" {
  name                = "finops-daily-trigger-${var.environment}"
  description         = "Triggers FinOps Bot daily at 2:00 AM"
  schedule_expression = "cron(0 2 * * ? *)"
  
  lifecycle {
    create_before_destroy = true
  }
  
  tags = {
    Name = "finops-daily-trigger-${var.environment}"
  }
}

resource "aws_cloudwatch_event_target" "lambda_target" {
  rule      = aws_cloudwatch_event_rule.daily_trigger.name
  target_id = "finops-lambda-target"
  arn       = aws_lambda_function.finops_cleaner.arn
}

# ──────────────────────────────────────────────────────────────
# Health Check Schedule - Every 5 Minutes (NEW)
# ──────────────────────────────────────────────────────────────

resource "aws_cloudwatch_event_rule" "health_trigger" {
  name                = "finops-health-trigger-${var.environment}"
  description         = "Triggers Health Check every ${var.health_check_interval} minutes"
  schedule_expression = "rate(${var.health_check_interval} minutes)"
  
  lifecycle {
    create_before_destroy = true
  }
  
  tags = {
    Name = "finops-health-trigger-${var.environment}"
  }
}

resource "aws_cloudwatch_event_target" "health_target" {
  rule      = aws_cloudwatch_event_rule.health_trigger.name
  target_id = "finops-health-target"
  arn       = aws_lambda_function.health_check.arn
}
```

---

### 3.8 DynamoDB State Table (`terraform/dynamodb.tf`)

```hcl
# terraform/dynamodb.tf

# DynamoDB State Table with Audit Fields
resource "aws_dynamodb_table" "state_table" {
  name           = "FinOps-State-${var.environment}"
  billing_mode   = "PAY_PER_REQUEST"
  hash_key       = "ResourceId"
  range_key      = "AccountId"
  
  attribute {
    name = "ResourceId"
    type = "S"
  }
  
  attribute {
    name = "AccountId"
    type = "S"
  }
  
  attribute {
    name = "ActionTaken"
    type = "S"
  }
  
  attribute {
    name = "DeletionTimestamp"
    type = "N"
  }
  
  attribute {
    name = "QuarantineExpiry"
    type = "N"
  }
  
  attribute {
    name = "Region"
    type = "S"
  }
  
  attribute {
    name = "DeleteProtection"
    type = "BOOL"
  }
  
  attribute {
    name = "ActionedBy"
    type = "S"
  }
  
  # Global Secondary Indexes
  global_secondary_index {
    name               = "ActionTaken-Index"
    hash_key           = "ActionTaken"
    range_key          = "DeletionTimestamp"
    projection_type    = "ALL"
  }
  
  global_secondary_index {
    name               = "QuarantineExpiry-Index"
    hash_key           = "ActionTaken"
    range_key          = "QuarantineExpiry"
    projection_type    = "ALL"
  }
  
  global_secondary_index {
    name               = "Region-Index"
    hash_key           = "Region"
    range_key          = "ResourceId"
    projection_type    = "ALL"
  }
  
  global_secondary_index {
    name               = "DeleteProtection-Index"
    hash_key           = "DeleteProtection"
    range_key          = "ResourceId"
    projection_type    = "ALL"
  }
  
  global_secondary_index {
    name               = "ActionedBy-Index"
    hash_key           = "ActionedBy"
    range_key          = "DeletionTimestamp"
    projection_type    = "ALL"
  }
  
  ttl {
    attribute_name = "ExpirationTimestamp"
    enabled        = true
  }
  
  # Point-in-Time Recovery for DR
  point_in_time_recovery {
    enabled = true
  }
  
  timeouts {
    create = "10m"
    update = "10m"
    delete = "10m"
  }
  
  tags = {
    Name = "FinOps-State-${var.environment}"
  }
}
```

---

### 3.9 S3 Audit Bucket (`terraform/s3.tf`)

```hcl
# terraform/s3.tf

# S3 Audit Bucket - FIXED: Includes environment in name
resource "aws_s3_bucket" "audit_bucket" {
  bucket        = var.s3_report_bucket != "" ? var.s3_report_bucket : "finops-audit-${var.environment}-${data.aws_caller_identity.current.account_id}"
  force_destroy = var.environment != "prod"
  
  tags = {
    Name = "finops-audit-${var.environment}"
  }
}

# Block Public Access
resource "aws_s3_bucket_public_access_block" "audit_bucket" {
  bucket = aws_s3_bucket.audit_bucket.id
  
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# Ownership Controls
resource "aws_s3_bucket_ownership_controls" "audit_bucket" {
  bucket = aws_s3_bucket.audit_bucket.id
  rule {
    object_ownership = "BucketOwnerPreferred"
  }
}

# Server-Side Encryption
resource "aws_s3_bucket_server_side_encryption_configuration" "audit_bucket" {
  bucket = aws_s3_bucket.audit_bucket.id
  
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

# Versioning - Enabled for DR
resource "aws_s3_bucket_versioning" "audit_bucket" {
  bucket = aws_s3_bucket.audit_bucket.id
  versioning_configuration {
    status = "Enabled"
  }
}

# Website Configuration
resource "aws_s3_bucket_website_configuration" "audit_bucket" {
  bucket = aws_s3_bucket.audit_bucket.id
  
  index_document {
    suffix = "index.html"
  }
  
  error_document {
    key = "index.html"
  }
}

# Lifecycle Configuration - Data Retention Policy
resource "aws_s3_bucket_lifecycle_configuration" "audit_bucket" {
  bucket = aws_s3_bucket.audit_bucket.id
  
  rule {
    id     = "transition-to-glacier"
    status = "Enabled"
    
    transition {
      days          = 30
      storage_class = "GLACIER"
    }
  }
  
  rule {
    id     = "delete-after-one-year"
    status = "Enabled"
    
    expiration {
      days = 365
    }
  }
  
  rule {
    id     = "abort-incomplete-multipart-upload"
    status = "Enabled"
    
    abort_incomplete_multipart_upload {
      days_after_initiation = 7
    }
  }
}

# CORS Configuration
resource "aws_s3_bucket_cors_configuration" "audit_bucket" {
  bucket = aws_s3_bucket.audit_bucket.id

  cors_rule {
    allowed_headers = ["*"]
    allowed_methods = ["GET", "HEAD"]
    allowed_origins = ["https://dashboard.example.com"]
    max_age_seconds = 3000
  }
}

# Bucket Policy - Deny Non-SSL
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

### 3.10 SQS Dead-Letter Queue (`terraform/sqs.tf`)

```hcl
# terraform/sqs.tf

# SQS Dead-Letter Queue
resource "aws_sqs_queue" "dlq" {
  name                       = "finops-dlq-${var.environment}"
  message_retention_seconds  = 1209600 # 14 days
  visibility_timeout_seconds = 30
  
  tags = {
    Name = "finops-dlq-${var.environment}"
  }
}
```

---

### 3.11 CloudWatch Logs (`terraform/cloudwatch.tf`)

```hcl
# terraform/cloudwatch.tf

# ──────────────────────────────────────────────────────────────
# CloudWatch Log Groups
# ──────────────────────────────────────────────────────────────

# Main Lambda Logs - Structured Logging
resource "aws_cloudwatch_log_group" "lambda_logs" {
  name              = "/aws/lambda/${aws_lambda_function.finops_cleaner.function_name}"
  retention_in_days = 30 # Data Retention Policy
  
  tags = {
    Name = "finops-lambda-logs"
  }
}

# Health Check Lambda Logs
resource "aws_cloudwatch_log_group" "health_logs" {
  name              = "/aws/lambda/${aws_lambda_function.health_check.function_name}"
  retention_in_days = 30
  
  tags = {
    Name = "finops-health-logs"
  }
}

# ──────────────────────────────────────────────────────────────
# CloudWatch Alarms
# ──────────────────────────────────────────────────────────────

# Alarm for Lambda Errors
resource "aws_cloudwatch_metric_alarm" "lambda_error_alarm" {
  alarm_name          = "finops-lambda-error-alarm-${var.environment}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = "Errors"
  namespace           = "AWS/Lambda"
  period              = "300"
  statistic           = "Sum"
  threshold           = "0"
  alarm_description   = "FinOps Bot Lambda has errors"
  alarm_actions       = var.sns_topic_arn != "" ? [var.sns_topic_arn] : []
  
  dimensions = {
    FunctionName = aws_lambda_function.finops_cleaner.function_name
  }
}

# Alarm for Lambda Duration (SLO: < 300 seconds)
resource "aws_cloudwatch_metric_alarm" "lambda_duration_alarm" {
  alarm_name          = "finops-lambda-duration-alarm-${var.environment}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = "Duration"
  namespace           = "AWS/Lambda"
  period              = "300"
  statistic           = "p90"
  threshold           = "250" # 250 seconds (SLO: < 300s)
  alarm_description   = "FinOps Bot Lambda duration is high"
  alarm_actions       = var.sns_topic_arn != "" ? [var.sns_topic_arn] : []
  
  dimensions = {
    FunctionName = aws_lambda_function.finops_cleaner.function_name
  }
}

# Alarm for DLQ Depth
resource "aws_cloudwatch_metric_alarm" "dlq_alarm" {
  alarm_name          = "finops-dlq-alarm-${var.environment}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = "ApproximateNumberOfMessagesVisible"
  namespace           = "AWS/SQS"
  period              = "300"
  statistic           = "Average"
  threshold           = "1"
  alarm_description   = "FinOps Bot DLQ has messages - manual intervention required"
  alarm_actions       = var.sns_topic_arn != "" ? [var.sns_topic_arn] : []
  
  dimensions = {
    QueueName = aws_sqs_queue.dlq.name
  }
}

# NEW: Health Check Alarm
resource "aws_cloudwatch_metric_alarm" "health_check_alarm" {
  alarm_name          = "finops-health-check-alarm-${var.environment}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "3"
  metric_name         = "HealthCheckStatus"
  namespace           = "FinOpsBot"
  period              = "300"
  statistic           = "Average"
  threshold           = "0"
  alarm_description   = "FinOps Bot health check failed for 3 consecutive checks"
  alarm_actions       = var.sns_topic_arn != "" ? [var.sns_topic_arn] : []
}

# NEW: Alarm for Low Invocation Count (Bot not running)
resource "aws_cloudwatch_metric_alarm" "invocation_alarm" {
  alarm_name          = "finops-invocation-alarm-${var.environment}"
  comparison_operator = "LessThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "Invocations"
  namespace           = "AWS/Lambda"
  period              = "86400" # 24 hours
  statistic           = "Sum"
  threshold           = "1"
  alarm_description   = "FinOps Bot has not run in the last 24 hours"
  alarm_actions       = var.sns_topic_arn != "" ? [var.sns_topic_arn] : []
  
  dimensions = {
    FunctionName = aws_lambda_function.finops_cleaner.function_name
  }
}

# NEW: Alarm for Lambda Throttles
resource "aws_cloudwatch_metric_alarm" "throttle_alarm" {
  alarm_name          = "finops-throttle-alarm-${var.environment}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = "Throttles"
  namespace           = "AWS/Lambda"
  period              = "300"
  statistic           = "Sum"
  threshold           = "0"
  alarm_description   = "FinOps Bot Lambda is being throttled"
  alarm_actions       = var.sns_topic_arn != "" ? [var.sns_topic_arn] : []
  
  dimensions = {
    FunctionName = aws_lambda_function.finops_cleaner.function_name
  }
}
```

---

### 3.12 CloudWatch Dashboard (`terraform/cloudwatch_dashboard.tf`) - NEW

```hcl
# terraform/cloudwatch_dashboard.tf

# CloudWatch Dashboard for Operations
resource "aws_cloudwatch_dashboard" "finops_dashboard" {
  dashboard_name = "FinOps-Bot-Dashboard-${var.environment}"
  
  dashboard_body = jsonencode({
    widgets = [
      {
        type = "metric"
        properties = {
          metrics = [
            ["AWS/Lambda", "Errors", "FunctionName", aws_lambda_function.finops_cleaner.function_name],
            ["AWS/Lambda", "Duration", "FunctionName", aws_lambda_function.finops_cleaner.function_name],
            ["AWS/Lambda", "Invocations", "FunctionName", aws_lambda_function.finops_cleaner.function_name],
            ["AWS/Lambda", "Throttles", "FunctionName", aws_lambda_function.finops_cleaner.function_name],
            ["AWS/SQS", "ApproximateNumberOfMessagesVisible", "QueueName", aws_sqs_queue.dlq.name],
            ["AWS/DynamoDB", "SuccessfulRequestLatency", "TableName", aws_dynamodb_table.state_table.name],
            ["AWS/S3", "NumberOfObjects", "BucketName", aws_s3_bucket.audit_bucket.id],
            ["FinOpsBot", "HealthCheckStatus"]
          ]
          period = 300
          stat = "Average"
          title = "FinOps Bot Operational Dashboard - ${var.environment}"
          view = "timeSeries"
          stacked = false
          region = var.aws_region
        }
      },
      {
        type = "metric"
        properties = {
          metrics = [
            [ "AWS/Lambda", "Errors", "FunctionName", aws_lambda_function.health_check.function_name ],
            [ "AWS/Lambda", "Duration", "FunctionName", aws_lambda_function.health_check.function_name ],
            [ "AWS/Lambda", "Invocations", "FunctionName", aws_lambda_function.health_check.function_name ]
          ]
          period = 300
          stat = "Average"
          title = "Health Check Lambda Metrics"
          view = "timeSeries"
          stacked = false
          region = var.aws_region
        }
      }
    ]
  })
}
```

---

### 3.13 SNS Topic (`terraform/sns.tf`)

```hcl
# terraform/sns.tf

# SNS Topic for Alerts
resource "aws_sns_topic" "alerts" {
  count = var.sns_topic_arn == "" ? 1 : 0
  
  name = "finops-alerts-${var.environment}"
  
  tags = {
    Name = "finops-alerts-${var.environment}"
  }
}

# SNS Email Subscription
resource "aws_sns_topic_subscription" "alerts_email" {
  count = var.sns_topic_arn == "" ? 1 : 0
  
  topic_arn = aws_sns_topic.alerts[0].arn
  protocol  = "email"
  endpoint  = "your-email@example.com" # Replace with actual email
}
```

---

### 3.14 Outputs (`terraform/outputs.tf`)

```hcl
# terraform/outputs.tf

output "lambda_function_name" {
  description = "Name of the main Lambda function"
  value       = aws_lambda_function.finops_cleaner.function_name
}

output "lambda_arn" {
  description = "ARN of the main Lambda function"
  value       = aws_lambda_function.finops_cleaner.arn
}

output "health_check_function_name" {
  description = "Name of the health check Lambda function"
  value       = aws_lambda_function.health_check.function_name
}

output "dynamodb_table_name" {
  description = "Name of the DynamoDB state table"
  value       = aws_dynamodb_table.state_table.name
}

output "s3_bucket_name" {
  description = "Name of the S3 audit bucket"
  value       = aws_s3_bucket.audit_bucket.bucket
}

output "s3_bucket_website_url" {
  description = "URL of the static website (if configured)"
  value       = try(aws_s3_bucket_website_configuration.audit_bucket.website_endpoint, null)
}

output "dashboard_url" {
  description = "URL of the FinOps Bot dashboard"
  value       = "http://${aws_s3_bucket_website_configuration.audit_bucket.website_endpoint}"
}

output "dashboard_s3_uri" {
  description = "S3 URI of the dashboard"
  value       = "s3://${aws_s3_bucket.audit_bucket.bucket}/${var.s3_report_prefix}index.html"
}

output "cloudwatch_dashboard_name" {
  description = "Name of the CloudWatch dashboard"
  value       = aws_cloudwatch_dashboard.finops_dashboard.dashboard_name
}

output "sqs_queue_url" {
  description = "URL of the SQS Dead-Letter Queue"
  value       = aws_sqs_queue.dlq.url
}

output "eventbridge_rule_name" {
  description = "Name of the EventBridge rule"
  value       = aws_cloudwatch_event_rule.daily_trigger.name
}

output "iam_role_arn" {
  description = "ARN of the Lambda IAM role"
  value       = aws_iam_role.lambda_role.arn
}

output "sns_topic_arn" {
  description = "ARN of the SNS topic"
  value       = var.sns_topic_arn != "" ? var.sns_topic_arn : try(aws_sns_topic.alerts[0].arn, null)
}

output "secrets_manager_arn" {
  description = "ARN of the Secrets Manager secret"
  value       = aws_secretsmanager_secret.slack_webhook.arn
}

output "kms_key_arn" {
  description = "ARN of the KMS key"
  value       = aws_kms_key.secrets_key.arn
}
```

---

## 4. Backend Bootstrap

### 4.1 Manual Bootstrap Steps

Before the first Terraform `apply`, the remote state backend must be bootstrapped manually:

1. **Create the S3 bucket:**
   ```bash
   aws s3 mb s3://finops-terraform-state-<account-id> --region us-east-1
   ```

2. **Enable versioning:**
   ```bash
   aws s3api put-bucket-versioning \
     --bucket finops-terraform-state-<account-id> \
     --versioning-configuration Status=Enabled
   ```

3. **Block public access:**
   ```bash
   aws s3api put-public-access-block \
     --bucket finops-terraform-state-<account-id> \
     --public-access-block-configuration BlockPublicAcls=true,IgnorePublicAcls=true,BlockPublicPolicy=true,RestrictPublicBuckets=true
   ```

4. **Create the DynamoDB table for state locking:**
   ```bash
   aws dynamodb create-table \
     --table-name terraform-locks \
     --attribute-definitions AttributeName=LockID,AttributeType=S \
     --key-schema AttributeName=LockID,KeyType=HASH \
     --billing-mode PAY_PER_REQUEST
   ```

### 4.2 Backend Configuration

The backend is configured in the Terraform provider block:

```hcl
terraform {
  backend "s3" {
    bucket         = "finops-terraform-state-<account-id>"
    key            = "finops-bot/terraform.tfstate"
    region         = "us-east-1"
    dynamodb_table = "terraform-locks"
    encrypt        = true
  }
}
```

---

## 5. Terraform Variables File (`terraform/terraform.tfvars.example`)

```hcl
# terraform/terraform.tfvars.example

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

# Replace with your actual values
s3_report_bucket = "finops-audit-dev-123456789012"
secrets_manager_arn = "arn:aws:secretsmanager:us-east-1:123456789012:secret:finops/slack-webhook-abc123"
sns_topic_arn = "arn:aws:sns:us-east-1:123456789012:finops-alerts"

enable_rds_savings = true

lambda_memory_size = 256
lambda_timeout = 300

timezone = "America/New_York"

health_check_interval = 5
```

---

## 6. IAM Policy Validation (CI/CD)

### 6.1 checkov Validation

```bash
# Validate IAM policies with checkov
checkov -d terraform/ --check CKV_AWS_* --framework terraform

# Specific checks for IAM policies
checkov -d terraform/ --check CKV_AWS_109  # No wildcard in IAM policy
checkov -d terraform/ --check CKV_AWS_111  # IAM policy has no explicit deny
checkov -d terraform/ --check CKV_AWS_112  # IAM policy has no privileged actions with wildcard
```

### 6.2 tfsec Validation

```bash
# Validate Terraform security with tfsec
tfsec terraform/ --exclude aws-iam-no-policy-wildcards

# Specific checks
tfsec terraform/ --include aws-iam-no-policy-wildcards,aws-iam-no-policy-unrestricted
```

### 6.3 CI Pipeline Integration

```yaml
# FlowCI pipeline stage
- name: "IAM Policy Validation"
  steps:
    - command: checkov -d terraform/ --framework terraform
    - command: tfsec terraform/
```

---

## 7. Sign-Off

| Role | Name | Date | Signature |
| :--- | :--- | :--- | :--- |
| **Project Lead / Architect** | Jibrin Ahmed | June 22, 2026 | JA |
| **Security Reviewer** | [Security Team / Imaginary CISO] | [Date] | [Initials] |

---

