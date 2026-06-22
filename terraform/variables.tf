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
}

variable "sns_topic_arn" {
  description = "ARN of the SNS topic for alerts"
  type        = string
  default     = ""
}

variable "enable_rds_savings" {
  description = "Enable RDS savings calculations and stopping"
  type        = bool
  default     = true
}

variable "lambda_memory_size" {
  description = "Memory size for main Lambda function in MB"
  type        = number
  default     = 256
  validation {
    condition     = contains([128, 256, 512, 1024, 2048, 4096, 6144, 8192, 10240], var.lambda_memory_size)
    error_message = "Lambda memory size must be one of: 128, 256, 512, 1024, 2048, 4096, 6144, 8192, 10240."
  }
}

variable "lambda_timeout" {
  description = "Timeout for main Lambda function in seconds"
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

# ──────────────────────────────────────────────────────────────
# Feature Toggles
# ──────────────────────────────────────────────────────────────

variable "enable_ec2_discovery" {
  description = "Enable EC2 resource discovery (requires working EC2 API). Set to 'true' or 'false'."
  type        = string
  default     = "false"
  validation {
    condition     = contains(["true", "false"], var.enable_ec2_discovery)
    error_message = "enable_ec2_discovery must be 'true' or 'false'."
  }
}

variable "enable_slack" {
  description = "Enable Slack notifications. Set to 'true' or 'false'."
  type        = string
  default     = "true"
  validation {
    condition     = contains(["true", "false"], var.enable_slack)
    error_message = "enable_slack must be 'true' or 'false'."
  }
}

variable "enable_s3" {
  description = "Enable S3 audit report upload. Set to 'true' or 'false'."
  type        = string
  default     = "true"
  validation {
    condition     = contains(["true", "false"], var.enable_s3)
    error_message = "enable_s3 must be 'true' or 'false'."
  }
}

variable "enable_dashboard" {
  description = "Enable dashboard generation and upload. Set to 'true' or 'false'."
  type        = string
  default     = "true"
  validation {
    condition     = contains(["true", "false"], var.enable_dashboard)
    error_message = "enable_dashboard must be 'true' or 'false'."
  }
}