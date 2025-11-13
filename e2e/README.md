# E2E Testing for tf-migrate

End-to-end testing environment for validating Terraform v4 to v5 migrations with real Cloudflare resources.

## What is this?

This directory lets you test the migration tool against real Terraform configurations before running it on production code. You can:

1. Set up v4 configs from integration test data
2. Run the migration tool to generate v5 configs
3. Deploy and validate both versions work with Cloudflare

## Quick Start

### First Time Setup

```bash
# 1. Configure your Cloudflare credentials
./scripts/init --account <your-account-id> --zone <your-zone-id>

# 2. Initialize and apply v4 resources
cd tf/v4
terraform init
terraform apply

# 3. Run migration to generate v5 configs
cd ../..
./scripts/migrate

# 4. Test v5 configs
cd migrated-v4_to_v5
terraform init
terraform plan  # Should show no changes
```

### Automated E2E Testing

Run the complete end-to-end test suite:

```bash
# Run full e2e test (from project root)
CLOUDFLARE_ACCOUNT_ID=<...> CLOUDFLARE_ZONE_ID=<...> ./scripts/run-e2e-tests
```

## Cleaning Up Resources

If you need to clean up orphaned test resources (resources that exist in Cloudflare but not in Terraform state), you can use the sweeper script from the Terraform provider repository:

```bash
# Clone the provider repo (if you haven't already)
git clone https://github.com/cloudflare/terraform-provider-cloudflare.git

# Run the sweeper for specific resource types
cd terraform-provider-cloudflare
./scripts/sweep --account $CLOUDFLARE_ACCOUNT_ID --zone $CLOUDFLARE_ZONE_ID --resource workers_kv_namespace

# You can sweep multiple resource types
./scripts/sweep --account $CLOUDFLARE_ACCOUNT_ID --zone $CLOUDFLARE_ZONE_ID --resource workers_kv_namespace --resource teams_list --resource teams_rule
```

Common resource types to clean up:
- `workers_kv_namespace` - Workers KV namespaces
- `teams_list` - Zero Trust lists
- `teams_rule` - Zero Trust Gateway rules
- `access_service_token` - Zero Trust Access service tokens
- `api_token` - API tokens
- `dns_record` - DNS records

**Note:** The sweeper is a destructive operation that will delete all resources of the specified type in the account/zone. Use with caution!