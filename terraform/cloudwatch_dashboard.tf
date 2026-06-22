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
            ["AWS/Lambda", "Errors", "FunctionName", aws_lambda_function.health_check.function_name],
            ["AWS/Lambda", "Duration", "FunctionName", aws_lambda_function.health_check.function_name],
            ["AWS/Lambda", "Invocations", "FunctionName", aws_lambda_function.health_check.function_name]
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