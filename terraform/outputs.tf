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