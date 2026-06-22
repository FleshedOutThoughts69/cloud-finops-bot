resource "aws_sqs_queue" "dlq" {
  name                       = "finops-dlq-${var.environment}"
  message_retention_seconds  = 1209600 # 14 days
  visibility_timeout_seconds = 30

  tags = {
    Name = "finops-dlq-${var.environment}"
  }
}