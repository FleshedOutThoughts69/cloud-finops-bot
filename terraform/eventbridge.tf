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
# Health Check Schedule - Every 5 Minutes
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