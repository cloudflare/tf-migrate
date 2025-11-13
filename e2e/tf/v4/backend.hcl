# Backend configuration for R2 remote state
# Account ID must be provided via TF_BACKEND_ACCOUNT_ID environment variable

# R2 endpoint - replace ACCOUNT_ID with actual value
# This file is a template and will be populated by scripts
endpoint = "https://ACCOUNT_ID.r2.cloudflarestorage.com"

bucket = "tf-migrate-e2e-state"
key    = "v4/terraform.tfstate"
region = "auto"

# R2 credentials (provided via environment variables)
# AWS_ACCESS_KEY_ID = R2 Access Key ID
# AWS_SECRET_ACCESS_KEY = R2 Secret Access Key

# State locking with DynamoDB-compatible API (if available)
# dynamodb_endpoint = "https://ACCOUNT_ID.r2.cloudflarestorage.com"
# dynamodb_table = "tf-migrate-state-lock"

# Skip AWS-specific validations
skip_credentials_validation = true
skip_region_validation      = true
skip_requesting_account_id  = true
skip_metadata_api_check     = true
skip_s3_checksum            = true

# Use path-style access for R2
force_path_style = true
