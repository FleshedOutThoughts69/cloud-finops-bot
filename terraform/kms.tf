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

resource "aws_kms_alias" "secrets_key" {
  name          = "alias/finops-secrets-${var.environment}"
  target_key_id = aws_kms_key.secrets_key.key_id
}