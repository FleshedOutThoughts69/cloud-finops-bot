resource "aws_secretsmanager_secret" "slack_webhook" {
  name                    = "finops/slack-webhook-${var.environment}"
  description             = "Slack webhook URL for FinOps Bot notifications"
  kms_key_id              = aws_kms_key.secrets_key.arn
  recovery_window_in_days = 30

  tags = {
    Name = "finops-slack-webhook-${var.environment}"
  }
}

resource "aws_secretsmanager_secret_version" "slack_webhook" {
  secret_id     = aws_secretsmanager_secret.slack_webhook.id
  secret_string = jsonencode({
    SLACK_WEBHOOK_URL = "https://hooks.slack.com/services/REPLACE_ME"
  })
}

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