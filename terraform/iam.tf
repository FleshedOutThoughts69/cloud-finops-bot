# ──────────────────────────────────────────────────────────────
# IAM Role for Main Lambda
# ──────────────────────────────────────────────────────────────

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

resource "aws_iam_policy" "lambda_policy" {
  name        = "finops-lambda-policy-${var.environment}"
  description = "Least-privilege permissions for FinOps Bot Lambda"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
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
      {
        Sid = "RDSReadOnly"
        Effect = "Allow"
        Action = ["rds:DescribeDBInstances"]
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
      {
        Sid = "SecretsManagerPermissions"
        Effect = "Allow"
        Action = ["secretsmanager:GetSecretValue"]
        Resource = aws_secretsmanager_secret.slack_webhook.arn
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
        Sid = "SSMParameterStore"
        Effect = "Allow"
        Action = [
          "ssm:GetParameter",
          "ssm:GetParameters",
          "ssm:GetParametersByPath"
        ]
        Resource = "arn:aws:ssm:${var.aws_region}:${data.aws_caller_identity.current.account_id}:parameter/finops/${var.environment}/*"
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
      },
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

resource "aws_iam_role_policy_attachment" "lambda_policy_attachment" {
  role       = aws_iam_role.lambda_role.name
  policy_arn = aws_iam_policy.lambda_policy.arn
}

# ──────────────────────────────────────────────────────────────
# IAM Role for Health Check Lambda
# ──────────────────────────────────────────────────────────────

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

resource "aws_iam_policy" "health_policy" {
  name        = "finops-health-policy-${var.environment}"
  description = "Minimal permissions for FinOps Bot Health Check"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid = "DynamoDBRead"
        Effect = "Allow"
        Action = ["dynamodb:DescribeTable"]
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
        Action = ["secretsmanager:DescribeSecret"]
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

resource "aws_iam_role_policy_attachment" "health_policy_attachment" {
  role       = aws_iam_role.health_role.name
  policy_arn = aws_iam_policy.health_policy.arn
}