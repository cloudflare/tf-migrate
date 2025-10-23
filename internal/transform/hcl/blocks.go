// Package hcl provides utilities for transforming HCL blocks and their structure
// during Terraform provider migrations.
package hcl

import (
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/cloudflare/tf-migrate/internal/hcl"
)

// RenameResourceType renames a resource from oldType to newType.
// Returns true if the resource type was renamed, false otherwise.
//
// Example - Renaming legacy resource type:
//
// Before:
//   resource "cloudflare_record" "example" {
//     zone_id = "abc123"
//     name    = "test"
//     type    = "A"
//     value   = "192.0.2.1"
//   }
//
// After calling RenameResourceType(block, "cloudflare_record", "cloudflare_dns_record"):
//   resource "cloudflare_dns_record" "example" {
//     zone_id = "abc123"
//     name    = "test"
//     type    = "A"
//     value   = "192.0.2.1"
//   }
func RenameResourceType(block *hclwrite.Block, oldType, newType string) bool {
	labels := block.Labels()
	if len(labels) >= 2 && labels[0] == oldType {
		labels[0] = newType
		block.SetLabels(labels)
		return true
	}
	return false
}

// GetResourceType returns the resource type from a resource block
func GetResourceType(block *hclwrite.Block) string {
	if block.Type() != "resource" {
		return ""
	}
	labels := block.Labels()
	if len(labels) >= 1 {
		return labels[0]
	}
	return ""
}

// GetResourceName returns the resource name from a resource block
func GetResourceName(block *hclwrite.Block) string {
	if block.Type() != "resource" {
		return ""
	}
	labels := block.Labels()
	if len(labels) >= 2 {
		return labels[1]
	}
	return ""
}

// ConvertBlocksToAttribute converts all blocks of a certain type to an object attribute.
// The preProcess function is called on each block before conversion (can be nil).
//
// Example - Converting data blocks to attribute for CAA records:
//
// Before:
//   resource "cloudflare_dns_record" "caa" {
//     zone_id = "abc123"
//     name    = "example.com"
//     type    = "CAA"
//     
//     data {
//       flags   = "0"
//       tag     = "issue"
//       content = "letsencrypt.org"
//     }
//   }
//
// After calling ConvertBlocksToAttribute(body, "data", "data", preProcess):
//   resource "cloudflare_dns_record" "caa" {
//     zone_id = "abc123"
//     name    = "example.com"
//     type    = "CAA"
//     data = {
//       flags = "0"
//       tag   = "issue"
//       value = "letsencrypt.org"  # Renamed by preProcess
//     }
//   }
func ConvertBlocksToAttribute(body *hclwrite.Body, blockType, attrName string, preProcess func(*hclwrite.Block)) {
	var blocksToRemove []*hclwrite.Block
	
	for _, block := range body.Blocks() {
		if block.Type() != blockType {
			continue
		}
		
		// Apply preprocessing if provided
		if preProcess != nil {
			preProcess(block)
		}
		
		// Convert block to object tokens
		objTokens := hcl.BuildObjectFromBlock(block)
		body.SetAttributeRaw(attrName, objTokens)
		blocksToRemove = append(blocksToRemove, block)
	}
	
	// Remove the converted blocks
	for _, block := range blocksToRemove {
		body.RemoveBlock(block)
	}
}

// HoistAttributeFromBlock copies an attribute from a nested block to the parent body.
// Returns true if the attribute was hoisted, false otherwise.
//
// Example - Hoisting priority from SRV record data block:
//
// Before:
//   resource "cloudflare_dns_record" "srv" {
//     zone_id = "abc123"
//     name    = "_sip._tcp"
//     type    = "SRV"
//     
//     data {
//       priority = 10
//       weight   = 60
//       port     = 5060
//       target   = "sipserver.example.com"
//     }
//   }
//
// After calling HoistAttributeFromBlock(body, "data", "priority"):
//   resource "cloudflare_dns_record" "srv" {
//     zone_id  = "abc123"
//     name     = "_sip._tcp"
//     type     = "SRV"
//     priority = 10  # Hoisted from data block
//     
//     data {
//       weight = 60
//       port   = 5060
//       target = "sipserver.example.com"
//     }
//   }
func HoistAttributeFromBlock(parentBody *hclwrite.Body, blockType, attrName string) bool {
	for _, block := range parentBody.Blocks() {
		if block.Type() != blockType {
			continue
		}
		if block.Body().GetAttribute(attrName) != nil {
			// Only hoist if parent doesn't already have this attribute
			if parentBody.GetAttribute(attrName) == nil {
				hcl.CopyAttribute(block.Body(), parentBody, attrName)
				return true
			}
		}
	}
	return false
}

// HoistAttributesFromBlock copies multiple attributes from nested blocks to parent
func HoistAttributesFromBlock(parentBody *hclwrite.Body, blockType string, attrNames ...string) int {
	hoisted := 0
	for _, attrName := range attrNames {
		if HoistAttributeFromBlock(parentBody, blockType, attrName) {
			hoisted++
		}
	}
	return hoisted
}

// FindBlockByType finds the first block of a given type
func FindBlockByType(body *hclwrite.Body, blockType string) *hclwrite.Block {
	for _, block := range body.Blocks() {
		if block.Type() == blockType {
			return block
		}
	}
	return nil
}

// FindBlocksByType finds all blocks of a given type
func FindBlocksByType(body *hclwrite.Body, blockType string) []*hclwrite.Block {
	var blocks []*hclwrite.Block
	for _, block := range body.Blocks() {
		if block.Type() == blockType {
			blocks = append(blocks, block)
		}
	}
	return blocks
}

// RemoveBlocksByType removes all blocks of a given type
func RemoveBlocksByType(body *hclwrite.Body, blockType string) int {
	blocks := FindBlocksByType(body, blockType)
	for _, block := range blocks {
		body.RemoveBlock(block)
	}
	return len(blocks)
}

// ProcessBlocksOfType applies a function to all blocks of a given type
func ProcessBlocksOfType(body *hclwrite.Body, blockType string, processor func(*hclwrite.Block) error) error {
	for _, block := range body.Blocks() {
		if block.Type() == blockType {
			if err := processor(block); err != nil {
				return err
			}
		}
	}
	return nil
}

// ConvertSingleBlockToAttribute converts the first block of a type to an attribute
// This is useful when a resource changes from having a single block to an attribute
func ConvertSingleBlockToAttribute(body *hclwrite.Body, blockType, attrName string) bool {
	block := FindBlockByType(body, blockType)
	if block == nil {
		return false
	}
	
	objTokens := hcl.BuildObjectFromBlock(block)
	body.SetAttributeRaw(attrName, objTokens)
	body.RemoveBlock(block)
	return true
}