resource "aws_dynamodb_table" "state_table" {
  name           = "FinOps-State-${var.environment}"
  billing_mode   = "PAY_PER_REQUEST"
  hash_key       = "ResourceId"
  range_key      = "AccountId"

  attribute {
    name = "ResourceId"
    type = "S"
  }

  attribute {
    name = "AccountId"
    type = "S"
  }

  attribute {
    name = "ActionTaken"
    type = "S"
  }

  attribute {
    name = "DeletionTimestamp"
    type = "N"
  }

  attribute {
    name = "QuarantineExpiry"
    type = "N"
  }

  attribute {
    name = "Region"
    type = "S"
  }

  attribute {
    name = "DeleteProtection"
    type = "BOOL"
  }

  attribute {
    name = "ActionedBy"
    type = "S"
  }

  global_secondary_index {
    name               = "ActionTaken-Index"
    hash_key           = "ActionTaken"
    range_key          = "DeletionTimestamp"
    projection_type    = "ALL"
  }

  global_secondary_index {
    name               = "QuarantineExpiry-Index"
    hash_key           = "ActionTaken"
    range_key          = "QuarantineExpiry"
    projection_type    = "ALL"
  }

  global_secondary_index {
    name               = "Region-Index"
    hash_key           = "Region"
    range_key          = "ResourceId"
    projection_type    = "ALL"
  }

  global_secondary_index {
    name               = "DeleteProtection-Index"
    hash_key           = "DeleteProtection"
    range_key          = "ResourceId"
    projection_type    = "ALL"
  }

  global_secondary_index {
    name               = "ActionedBy-Index"
    hash_key           = "ActionedBy"
    range_key          = "DeletionTimestamp"
    projection_type    = "ALL"
  }

  ttl {
    attribute_name = "ExpirationTimestamp"
    enabled        = true
  }

  point_in_time_recovery {
    enabled = true
  }

  timeouts {
    create = "10m"
    update = "10m"
    delete = "10m"
  }

  tags = {
    Name = "FinOps-State-${var.environment}"
  }
}