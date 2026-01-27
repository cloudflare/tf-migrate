// env.go manages environment variable loading and validation for e2e tests.
//
// This file provides centralized management of environment variables required
// for testing, including Cloudflare credentials, R2 access keys, and test
// configuration. It ensures all required variables are present before tests
// run and provides clear error messages for missing configuration.
package e2e

import (
	"fmt"
	"os"
)

// E2EEnv holds all environment variables used by e2e tests
type E2EEnv struct {
	AccountID              string
	ZoneID                 string
	Domain                 string
	Email                  string
	APIKey                 string
	R2AccessKeyID          string
	R2SecretAccessKey      string
	CrowdstrikeClientID    string
	CrowdstrikeClientSecret string
	CrowdstrikeAPIURL      string
	CrowdstrikeCustomerID  string
}

// LoadEnv loads environment variables and validates required ones
func LoadEnv(required []string) (*E2EEnv, error) {
	env := &E2EEnv{
		AccountID:              os.Getenv("CLOUDFLARE_ACCOUNT_ID"),
		ZoneID:                 os.Getenv("CLOUDFLARE_ZONE_ID"),
		Domain:                 os.Getenv("CLOUDFLARE_DOMAIN"),
		Email:                  os.Getenv("CLOUDFLARE_EMAIL"),
		APIKey:                 os.Getenv("CLOUDFLARE_API_KEY"),
		R2AccessKeyID:          os.Getenv("CLOUDFLARE_R2_ACCESS_KEY_ID"),
		R2SecretAccessKey:      os.Getenv("CLOUDFLARE_R2_SECRET_ACCESS_KEY"),
		CrowdstrikeClientID:    os.Getenv("CLOUDFLARE_CROWDSTRIKE_CLIENT_ID"),
		CrowdstrikeClientSecret: os.Getenv("CLOUDFLARE_CROWDSTRIKE_CLIENT_SECRET"),
		CrowdstrikeAPIURL:      os.Getenv("CLOUDFLARE_CROWDSTRIKE_API_URL"),
		CrowdstrikeCustomerID:  os.Getenv("CLOUDFLARE_CROWDSTRIKE_CUSTOMER_ID"),
	}

	// Validate required variables
	for _, varName := range required {
		var value string
		switch varName {
		case "CLOUDFLARE_ACCOUNT_ID":
			value = env.AccountID
		case "CLOUDFLARE_ZONE_ID":
			value = env.ZoneID
		case "CLOUDFLARE_DOMAIN":
			value = env.Domain
		case "CLOUDFLARE_EMAIL":
			value = env.Email
		case "CLOUDFLARE_API_KEY":
			value = env.APIKey
		case "CLOUDFLARE_R2_ACCESS_KEY_ID":
			value = env.R2AccessKeyID
		case "CLOUDFLARE_R2_SECRET_ACCESS_KEY":
			value = env.R2SecretAccessKey
		case "CLOUDFLARE_CROWDSTRIKE_CLIENT_ID":
			value = env.CrowdstrikeClientID
		case "CLOUDFLARE_CROWDSTRIKE_CLIENT_SECRET":
			value = env.CrowdstrikeClientSecret
		case "CLOUDFLARE_CROWDSTRIKE_API_URL":
			value = env.CrowdstrikeAPIURL
		case "CLOUDFLARE_CROWDSTRIKE_CUSTOMER_ID":
			value = env.CrowdstrikeCustomerID
		default:
			return nil, fmt.Errorf("unknown environment variable: %s", varName)
		}

		if value == "" {
			return nil, fmt.Errorf("%s environment variable is required", varName)
		}
	}

	return env, nil
}

// Common sets of required environment variables
var (
	// EnvForInit requires variables for init command
	EnvForInit = []string{
		"CLOUDFLARE_ACCOUNT_ID",
		"CLOUDFLARE_ZONE_ID",
		"CLOUDFLARE_DOMAIN",
	}

	// EnvForBackend requires variables for backend configuration
	EnvForBackend = []string{
		"CLOUDFLARE_ACCOUNT_ID",
		"CLOUDFLARE_ZONE_ID",
		"CLOUDFLARE_DOMAIN",
		"CLOUDFLARE_API_KEY",
		"CLOUDFLARE_EMAIL",
		"CLOUDFLARE_R2_ACCESS_KEY_ID",
		"CLOUDFLARE_R2_SECRET_ACCESS_KEY",
	}

	// EnvForRunner requires variables for e2e test runner (email is optional, shown for info only)
	EnvForRunner = []string{
		"CLOUDFLARE_ACCOUNT_ID",
		"CLOUDFLARE_ZONE_ID",
		"CLOUDFLARE_DOMAIN",
	}

	// EnvForBootstrap requires variables for bootstrap command
	EnvForBootstrap = []string{
		"CLOUDFLARE_ACCOUNT_ID",
		"CLOUDFLARE_R2_ACCESS_KEY_ID",
		"CLOUDFLARE_R2_SECRET_ACCESS_KEY",
	}

	// EnvForClean requires variables for clean command
	EnvForClean = []string{
		"CLOUDFLARE_R2_ACCESS_KEY_ID",
		"CLOUDFLARE_R2_SECRET_ACCESS_KEY",
	}
)
