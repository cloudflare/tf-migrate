package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/go-hclog"
)

func TestUpdateProviderVersionConstraint(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		targetVersion string
		expected      string
		shouldUpdate  bool
	}{
		{
			name: "replace simple version constraint",
			input: `terraform {
  required_providers {
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 4.0"
    }
  }
}`,
			targetVersion: "5.19.0-beta.4",
			expected: `terraform {
  required_providers {
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "5.19.0-beta.4"
    }
  }
}`,
			shouldUpdate: true,
		},
		{
			name: "replace complex version constraint",
			input: `terraform {
  required_providers {
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = ">= 4.0, < 5.0"
    }
  }
}`,
			targetVersion: "5.19.0-beta.4",
			expected: `terraform {
  required_providers {
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "5.19.0-beta.4"
    }
  }
}`,
			shouldUpdate: true,
		},
		{
			name: "replace version with different spacing",
			input: `terraform {
  required_providers {
    cloudflare = {
      source = "cloudflare/cloudflare"
      version = "~> 4.0"
    }
  }
}`,
			targetVersion: "5.19.0-beta.4",
			expected: `terraform {
  required_providers {
    cloudflare = {
      source = "cloudflare/cloudflare"
      version = "5.19.0-beta.4"
    }
  }
}`,
			shouldUpdate: true,
		},
		{
			name: "add version when not exists",
			input: `terraform {
  required_providers {
    cloudflare = {
      source = "cloudflare/cloudflare"
    }
  }
}`,
			targetVersion: "5.19.0-beta.4",
			expected: `terraform {
  required_providers {
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "5.19.0-beta.4"
    }
  }
}`,
			shouldUpdate: true, // We add version if not exists
		},
		{
			name: "do not modify other providers",
			input: `terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 4.0"
    }
  }
}`,
			targetVersion: "5.19.0-beta.4",
			expected: `terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "5.19.0-beta.4"
    }
  }
}`,
			shouldUpdate: true,
		},
		{
			name: "handle multiline provider block",
			input: `terraform {
  required_providers {
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 4.0"
    }
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}`,
			targetVersion: "5.19.0-beta.4",
			expected: `terraform {
  required_providers {
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "5.19.0-beta.4"
    }
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}`,
			shouldUpdate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tempDir := t.TempDir()

			// Write input file
			tfFile := filepath.Join(tempDir, "main.tf")
			err := os.WriteFile(tfFile, []byte(tt.input), 0644)
			if err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			// Create lock file to bypass version check
			lockFile := filepath.Join(tempDir, ".terraform.lock.hcl")
			lockContent := `provider "registry.terraform.io/cloudflare/cloudflare" {
  version     = "4.52.5"
  constraints = "~> 4.0"
}
`
			err = os.WriteFile(lockFile, []byte(lockContent), 0644)
			if err != nil {
				t.Fatalf("Failed to write lock file: %v", err)
			}

			// Create config
			cfg := config{
				configDir:             tempDir,
				sourceVersion:         "v4",
				targetVersion:         "v5",
				targetProviderVersion: tt.targetVersion,
				skipVersionCheck:      true,
			}

			// Create logger
			log := hclog.New(&hclog.LoggerOptions{
				Level:  hclog.Error,
				Output: os.Stderr,
			})

			// Call the function
			err = updateProviderVersionConstraint(log, cfg, nil)
			if err != nil {
				t.Fatalf("updateProviderVersionConstraint failed: %v", err)
			}

			// Read the result
			result, err := os.ReadFile(tfFile)
			if err != nil {
				t.Fatalf("Failed to read result file: %v", err)
			}

			if tt.shouldUpdate {
				if strings.TrimSpace(string(result)) != strings.TrimSpace(tt.expected) {
					t.Errorf("Version constraint not updated correctly.\nExpected:\n%s\n\nGot:\n%s", tt.expected, string(result))
				}

				// Verify the version constraint doesn't contain both old and new versions
				resultStr := string(result)
				if strings.Contains(resultStr, "~> 4.0") && strings.Contains(resultStr, tt.targetVersion) {
					t.Errorf("Version constraint contains both old and new versions: %s", resultStr)
				}

				// Verify only the target version is present
				if !strings.Contains(resultStr, `version = "`+tt.targetVersion+`"`) {
					t.Errorf("Version constraint doesn't contain target version %q: %s", tt.targetVersion, resultStr)
				}
			} else {
				// Should not have modified the file
				if strings.TrimSpace(string(result)) != strings.TrimSpace(tt.input) {
					t.Errorf("File was modified when it shouldn't have been.\nExpected:\n%s\n\nGot:\n%s", tt.input, string(result))
				}
			}
		})
	}
}
