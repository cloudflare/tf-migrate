package authenticated_origin_pulls_certificate

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles migration of authenticated origin pulls certificate resources from v4 to v5
// The v4 cloudflare_authenticated_origin_pulls_certificate resource with a type field
// is split into two separate resources in v5:
// - cloudflare_authenticated_origin_pulls_certificate (for type="per-zone")
// - cloudflare_authenticated_origin_pulls_hostname_certificate (for type="per-hostname")
type V4ToV5Migrator struct {
	mu sync.Mutex
	// perHostname records every resource that was type="per-hostname", keyed by
	// directory and name, with the file that declares it, so references to it
	// can be renamed in global postprocessing.
	perHostname map[perHostnameKey]string
}

type perHostnameKey struct {
	dir  string
	name string
}

const (
	perZoneType     = "cloudflare_authenticated_origin_pulls_certificate"
	perHostnameType = "cloudflare_authenticated_origin_pulls_hostname_certificate"
)

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{
		perHostname: make(map[perHostnameKey]string),
	}
	internal.RegisterMigrator("cloudflare_authenticated_origin_pulls_certificate", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Returns empty string because this resource routes to TWO different types based on type field:
	// - type="per-hostname" → cloudflare_authenticated_origin_pulls_hostname_certificate (via moved blocks in TransformConfig)
	// - type="per-zone" (or default) → cloudflare_authenticated_origin_pulls_certificate (no type change)
	// The actual type is determined dynamically in TransformConfig based on config attributes.
	return ""
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_authenticated_origin_pulls_certificate"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed
	return content
}

// GetInstanceRenames implements transform.InstanceRenamer. Only per-hostname
// certificates change type, so references are renamed per resource rather than
// per type, and only in the directory that declares the resource.
func (m *V4ToV5Migrator) GetInstanceRenames() []transform.InstanceRename {
	m.mu.Lock()
	defer m.mu.Unlock()
	renames := make([]transform.InstanceRename, 0, len(m.perHostname))
	for k, file := range m.perHostname {
		renames = append(renames, transform.InstanceRename{
			OldType: perZoneType,
			NewType: perHostnameType,
			Name:    k.name,
			Dir:     k.dir,
			File:    file,
		})
	}
	sort.Slice(renames, func(a, b int) bool {
		if renames[a].Dir != renames[b].Dir {
			return renames[a].Dir < renames[b].Dir
		}
		return renames[a].Name < renames[b].Name
	})
	return renames
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()
	resourceName := tfhcl.GetResourceName(block)

	// Determine target resource type by reading from config
	var targetType string
	var typeFromState string

	typeAttr := body.GetAttribute("type")
	if typeAttr != nil {
		typeFromState = tfhcl.ExtractStringFromAttribute(typeAttr)
		if _, ok := tfhcl.JoinQuotedLiteral(typeAttr.Expr().BuildTokens(nil)); !ok {
			// A variable or expression can't be resolved here, so the resource
			// is treated as per-zone. Without a warning, a per-hostname
			// certificate configured this way would silently keep the wrong type.
			typeFromState = ""
			expr := strings.TrimSpace(string(typeAttr.Expr().BuildTokens(nil).Bytes()))
			ctx.Diagnostics = append(ctx.Diagnostics, &hcl.Diagnostic{
				Severity: hcl.DiagWarning,
				Summary: fmt.Sprintf(
					"Action required: could not resolve type on %s.%s", perZoneType, resourceName),
				Detail: fmt.Sprintf(
					"type = %s is not a string literal, so this certificate was migrated as per-zone "+
						"(%s). If it is per-hostname, rename the resource to %s and add:\n\n"+
						"  moved {\n    from = %s.%s\n    to   = %s.%s\n  }\n\n"+
						"and update every reference to it.",
					expr, perZoneType, perHostnameType,
					perZoneType, resourceName, perHostnameType, resourceName),
			})
		}
	}

	// Determine target type based on type value
	if typeFromState == "per-hostname" {
		targetType = perHostnameType
		// Record this resource so references to it are renamed in global postprocessing.
		dir := ""
		if ctx.FilePath != "" {
			dir = filepath.Clean(filepath.Dir(ctx.FilePath))
		}
		m.mu.Lock()
		m.perHostname[perHostnameKey{dir: dir, name: resourceName}] = ctx.FilePath
		m.mu.Unlock()
	} else {
		// Default to per-zone (includes "per-zone", empty, or any other value)
		targetType = perZoneType
	}

	// Rename the resource type
	tfhcl.RenameResourceType(block,
		"cloudflare_authenticated_origin_pulls_certificate",
		targetType)

	// Remove type field from v5 configuration (not present in v5 schemas)
	tfhcl.RemoveAttributes(body, "type")

	// Generate moved block for per-hostname resources (per-zone keeps same name)
	var blocks []*hclwrite.Block
	blocks = append(blocks, block)

	if typeFromState == "per-hostname" {
		movedBlock := tfhcl.CreateMovedBlock(
			"cloudflare_authenticated_origin_pulls_certificate."+resourceName,
			"cloudflare_authenticated_origin_pulls_hostname_certificate."+resourceName,
		)
		blocks = append(blocks, movedBlock)
	}

	return &transform.TransformResult{
		Blocks:         blocks,
		RemoveOriginal: true, // Must be true for blocks to be added
	}, nil
}
