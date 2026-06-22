# ──────────────────────────────────────────────────────────────
# Build Main Lambda Binary
# ──────────────────────────────────────────────────────────────

resource "null_resource" "build_lambda" {
  triggers = {
    always_run = timestamp()
  }

  provisioner "local-exec" {
    command = "GOOS=linux GOARCH=amd64 go build -o ../bootstrap cmd/main.go"
    working_dir = path.module
  }
}

data "archive_file" "lambda_zip" {
  depends_on = [null_resource.build_lambda]
  type        = "zip"
  source_file = "../bootstrap"
  output_path = "lambda.zip"
}

# ──────────────────────────────────────────────────────────────
# Main Lambda Function
# ──────────────────────────────────────────────────────────────

resource "aws_lambda_function" "finops_cleaner" {
  depends_on = [
    aws_iam_role_policy_attachment.lambda_policy_attachment,
    aws_cloudwatch_log_group.lambda_logs
  ]

  filename         = data.archive_file.lambda_zip.output_path
  source_code_hash = data.archive_file.lambda_zip.output_base64sha256
  function_name    = "finops-cleaner-${var.environment}"
  role             = aws_iam_role.lambda_role.arn
  handler          = "bootstrap"
  runtime          = "provided.al2"
  timeout          = var.lambda_timeout
  memory_size      = var.lambda_memory_size

  reserved_concurrent_executions = 1

  environment {
    variables = {
      # Core configuration
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
      DYNAMODB_TABLE          = aws_dynamodb_table.state_table.name

      # Feature toggles (added from variables)
      ENABLE_EC2_DISCOVERY = var.enable_ec2_discovery
      ENABLE_SLACK         = var.enable_slack
      ENABLE_S3            = var.enable_s3
      ENABLE_DASHBOARD     = var.enable_dashboard
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
# Build Health Check Lambda Binary
# ──────────────────────────────────────────────────────────────

resource "null_resource" "build_health" {
  triggers = {
    always_run = timestamp()
  }

  provisioner "local-exec" {
    command = "GOOS=linux GOARCH=amd64 go build -o ../health_bootstrap cmd/health_check.go"
    working_dir = path.module
  }
}

data "archive_file" "health_zip" {
  depends_on = [null_resource.build_health]
  type        = "zip"
  source_file = "../health_bootstrap"
  output_path = "health.zip"
}

# ──────────────────────────────────────────────────────────────
# Health Check Lambda Function
# ──────────────────────────────────────────────────────────────

resource "aws_lambda_function" "health_check" {
  depends_on = [
    aws_iam_role_policy_attachment.health_policy_attachment
  ]

  filename         = data.archive_file.health_zip.output_path
  source_code_hash = data.archive_file.health_zip.output_base64sha256
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
      DYNAMODB_TABLE      = aws_dynamodb_table.state_table.name
      SSM_PARAM_PATH      = "/finops/${var.environment}/regions"
    }
  }

  tags = {
    Name = "finops-health-check-${var.environment}"
  }
}

# ──────────────────────────────────────────────────────────────
# Lambda Permissions
# ──────────────────────────────────────────────────────────────

resource "aws_lambda_permission" "eventbridge_invoke" {
  statement_id  = "AllowEventBridgeInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.finops_cleaner.function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.daily_trigger.arn
}

resource "aws_lambda_permission" "health_invoke" {
  statement_id  = "AllowEventBridgeHealthInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.health_check.function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.health_trigger.arn
}

# ──────────────────────────────────────────────────────────────
# Dead-Letter Queue Configuration
# ──────────────────────────────────────────────────────────────

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