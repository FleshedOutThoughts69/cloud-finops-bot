resource "aws_sns_topic" "alerts" {
  count = var.sns_topic_arn == "" ? 1 : 0

  name = "finops-alerts-${var.environment}"

  tags = {
    Name = "finops-alerts-${var.environment}"
  }
}

resource "aws_sns_topic_subscription" "alerts_email" {
  count = var.sns_topic_arn == "" ? 1 : 0

  topic_arn = aws_sns_topic.alerts[0].arn
  protocol  = "email"
  endpoint  = "your-email@example.com" # Replace with your email
}