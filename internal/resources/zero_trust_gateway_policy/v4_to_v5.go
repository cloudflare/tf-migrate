package zero_trust_gateway_policy

import (
	"fmt"
	"hash/crc32"
	"regexp"
	"strings"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	"github.com/cloudflare/tf-migrate/internal/transform/state"
)

// V4ToV5Migrator handles migration of Zero Trust Gateway Policy resources from v4 to v5
type V4ToV5Migrator struct {
	oldType string
	newType string
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{
		oldType: "cloudflare_teams_rule",
		newType: "cloudflare_zero_trust_gateway_policy",
	}
	// Register with OLD resource name
	internal.RegisterMigrator("cloudflare_teams_rule", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Return NEW resource type
	return m.newType
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == m.oldType
}

// Preprocess handles complex string-level transformations
func (m *V4ToV5Migrator) Preprocess(content string) string {
	// Process each resource block separately to avoid mixing data
	resourcePattern := regexp.MustCompile(`(?ms)(resource\s+"cloudflare_teams_rule"\s+"[^"]+"\s*\{(?:[^{}]|\{[^}]*\})*\})`)
	
	content = resourcePattern.ReplaceAllStringFunc(content, func(resourceBlock string) string {
		// Rename resource type
		resourceBlock = strings.Replace(resourceBlock, `resource "cloudflare_teams_rule"`, `resource "cloudflare_zero_trust_gateway_policy"`, 1)
		
		// Convert rule_settings block syntax to attribute syntax (required for SingleNestedAttribute in v5)
		// Replace "rule_settings {" with "rule_settings = {"
		ruleSettingsPattern := regexp.MustCompile(`(\s+)rule_settings\s*\{`)
		resourceBlock = ruleSettingsPattern.ReplaceAllString(resourceBlock, "${1}rule_settings = {")
		
		// Convert all nested blocks within rule_settings to attribute syntax (v5 uses SingleNestedAttribute)
		// These blocks become attributes with object values
		nestedBlocks := []string{
			"audit_ssh",
			"biso_admin_controls",
			"block_page",
			"check_session",
			"dns_resolvers",
			"egress",
			"insecure_disable_dnssec_validation",
			"l4override",
			"notification_settings",
			"payload_log",
			"untrusted_cert",
		}
		
		for _, block := range nestedBlocks {
			// Replace "block_name {" with "block_name = {" within rule_settings
			pattern := regexp.MustCompile(`(\s+)` + block + `\s*\{`)
			resourceBlock = pattern.ReplaceAllString(resourceBlock, "${1}" + block + " = {")
		}
		
		// Handle nested field renames in rule_settings
		// Rename block_page_reason to block_reason
		blockReasonPattern := regexp.MustCompile(`(\s+)block_page_reason(\s*=)`)
		resourceBlock = blockReasonPattern.ReplaceAllString(resourceBlock, "${1}block_reason${2}")
		
		// Rename notification_settings.message to notification_settings.msg
		// Need to be careful to only rename within notification_settings block
		// Pattern now matches "notification_settings = {" after conversion
		notificationPattern := regexp.MustCompile(`(?ms)(notification_settings\s*=\s*\{[^}]*?)(\s+)message(\s*=)`)
		resourceBlock = notificationPattern.ReplaceAllString(resourceBlock, "${1}${2}msg${3}")
		
		// Handle BISO admin control renames
		// Map disable_* to short codes (v1 style)
		// Pattern now matches "biso_admin_controls = {" after conversion
		bisoPattern := regexp.MustCompile(`(?ms)(biso_admin_controls\s*=\s*\{[^}]*?\})`)
		resourceBlock = bisoPattern.ReplaceAllStringFunc(resourceBlock, func(bisoBlock string) string {
			// Rename fields within BISO block
			bisoBlock = strings.Replace(bisoBlock, "disable_printing", "dp", -1)
			bisoBlock = strings.Replace(bisoBlock, "disable_copy_paste", "dcp", -1)
			bisoBlock = strings.Replace(bisoBlock, "disable_download", "dd", -1)
			bisoBlock = strings.Replace(bisoBlock, "disable_keyboard", "dk", -1)
			bisoBlock = strings.Replace(bisoBlock, "disable_upload", "du", -1)
			// Remove disable_clipboard_redirection (v1 only, deprecated)
			// Need to handle both with and without trailing comma
			clipboardPattern := regexp.MustCompile(`\s*disable_clipboard_redirection\s*=\s*\w+\s*,?\s*\n`)
			bisoBlock = clipboardPattern.ReplaceAllString(bisoBlock, "\n")
			return bisoBlock
		})
		
		return resourceBlock
	})
	
	return content
}

// TransformConfig handles HCL-level transformations
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// Resource type already renamed in preprocessing
	// Note: description field is now optional (was required)
	// Note: precedence field is now optional+computed (was required)
	// These don't need explicit handling - just preserve the values
	
	// No additional HCL transformations needed
	// Field renames were handled in preprocessing
	
	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// TransformState handles state JSON transformations
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, instance gjson.Result, resourcePath string) (string, error) {
	instanceStr := instance.String()
	attrs := instance.Get("attributes")
	
	if !attrs.Exists() {
		// Even for invalid instances, set schema_version
		instanceStr, _ = sjson.Set(instanceStr, "schema_version", 0)
		return instanceStr, nil
	}
	
	// Convert all numeric fields from int to float64
	// precedence field - v4 transformed this value, we need to reverse it
	if precedence := attrs.Get("precedence"); precedence.Exists() {
		// Get the rule name to calculate the hash
		ruleName := ""
		if name := attrs.Get("name"); name.Exists() {
			ruleName = name.String()
		}
		
		// v4 used: provided*1000 + hash(name)%1000
		// To reverse: (apiValue - hash(name)%1000) / 1000
		apiPrecedence := precedence.Int()
		if apiPrecedence > 0 && ruleName != "" {
			// Calculate the hash the same way v4 did
			nameHash := hashCodeString(ruleName)
			// Reverse the transformation
			originalPrecedence := (apiPrecedence - int64(nameHash)%1000) / 1000
			instanceStr, _ = sjson.Set(instanceStr, "attributes.precedence", float64(originalPrecedence))
		} else {
			// If we can't reverse it, just convert to float64
			value := state.ConvertToFloat64(precedence)
			instanceStr, _ = sjson.Set(instanceStr, "attributes.precedence", value)
		}
	}
	
	// version field (computed)
	if version := attrs.Get("version"); version.Exists() {
		value := state.ConvertToFloat64(version)
		instanceStr, _ = sjson.Set(instanceStr, "attributes.version", value)
	}
	
	// Handle nested rule_settings transformations
	// v4 stores rule_settings as array with MaxItems:1, v5 expects single object
	if ruleSettings := attrs.Get("rule_settings"); ruleSettings.Exists() {
		// Process rule_settings array and convert to single object for v5
		if ruleSettings.IsArray() {
			if len(ruleSettings.Array()) == 0 {
				// Empty array, convert to empty object
				instanceStr, _ = sjson.Set(instanceStr, "attributes.rule_settings", map[string]interface{}{})
			} else {
				// Get the first (and only) element from the array
				settings := ruleSettings.Array()[0]
			
			// Set it as a single object instead of array
			instanceStr, _ = sjson.Set(instanceStr, "attributes.rule_settings", settings.Value())
			
			// Now process the single object
			basePath := "attributes.rule_settings"
			
			// Rename block_page_reason to block_reason
			if blockReason := settings.Get("block_page_reason"); blockReason.Exists() {
				instanceStr, _ = sjson.Set(instanceStr, basePath+".block_reason", blockReason.Value())
				instanceStr, _ = sjson.Delete(instanceStr, basePath+".block_page_reason")
			}
			
			// Handle notification_settings (also convert from array to object if needed)
			if notifSettings := settings.Get("notification_settings"); notifSettings.Exists() {
				if notifSettings.IsArray() {
					if len(notifSettings.Array()) > 0 {
						// Convert notification_settings from array to object
						notif := notifSettings.Array()[0]
						instanceStr, _ = sjson.Set(instanceStr, basePath+".notification_settings", notif.Value())
						notifPath := basePath + ".notification_settings"
						
						// Rename message to msg
						if message := notif.Get("message"); message.Exists() {
							instanceStr, _ = sjson.Set(instanceStr, notifPath+".msg", message.Value())
							instanceStr, _ = sjson.Delete(instanceStr, notifPath+".message")
						}
					} else {
						instanceStr, _ = sjson.Delete(instanceStr, basePath+".notification_settings")
					}
				}
			}
				
			// Handle BISO admin controls (convert from array to object)
			if bisoControls := settings.Get("biso_admin_controls"); bisoControls.Exists() {
				if bisoControls.IsArray() {
					if len(bisoControls.Array()) > 0 {
						biso := bisoControls.Array()[0]
						instanceStr, _ = sjson.Set(instanceStr, basePath+".biso_admin_controls", biso.Value())
						bisoPath := basePath + ".biso_admin_controls"
						
						// Rename disable_* fields to short codes
						if field := biso.Get("disable_printing"); field.Exists() {
							instanceStr, _ = sjson.Set(instanceStr, bisoPath+".dp", field.Value())
							instanceStr, _ = sjson.Delete(instanceStr, bisoPath+".disable_printing")
						}
						if field := biso.Get("disable_copy_paste"); field.Exists() {
							instanceStr, _ = sjson.Set(instanceStr, bisoPath+".dcp", field.Value())
							instanceStr, _ = sjson.Delete(instanceStr, bisoPath+".disable_copy_paste")
						}
						if field := biso.Get("disable_download"); field.Exists() {
							instanceStr, _ = sjson.Set(instanceStr, bisoPath+".dd", field.Value())
							instanceStr, _ = sjson.Delete(instanceStr, bisoPath+".disable_download")
						}
						if field := biso.Get("disable_keyboard"); field.Exists() {
							instanceStr, _ = sjson.Set(instanceStr, bisoPath+".dk", field.Value())
							instanceStr, _ = sjson.Delete(instanceStr, bisoPath+".disable_keyboard")
						}
						if field := biso.Get("disable_upload"); field.Exists() {
							instanceStr, _ = sjson.Set(instanceStr, bisoPath+".du", field.Value())
							instanceStr, _ = sjson.Delete(instanceStr, bisoPath+".disable_upload")
						}
						
						// Remove disable_clipboard_redirection (deprecated)
						instanceStr, _ = sjson.Delete(instanceStr, bisoPath+".disable_clipboard_redirection")
					} else {
						instanceStr, _ = sjson.Delete(instanceStr, basePath+".biso_admin_controls")
					}
				}
			}
				
			// Handle DNS resolvers (convert from array to object and handle port conversions)
			if dnsResolvers := settings.Get("dns_resolvers"); dnsResolvers.Exists() {
				if dnsResolvers.IsArray() {
					if len(dnsResolvers.Array()) > 0 {
						resolver := dnsResolvers.Array()[0]
						instanceStr, _ = sjson.Set(instanceStr, basePath+".dns_resolvers", resolver.Value())
						resolverPath := basePath + ".dns_resolvers"
						
						// Convert IPv4 ports
						if ipv4Array := resolver.Get("ipv4"); ipv4Array.Exists() && ipv4Array.IsArray() {
							for k, ipv4 := range ipv4Array.Array() {
								if port := ipv4.Get("port"); port.Exists() {
									value := state.ConvertToFloat64(port)
									instanceStr, _ = sjson.Set(instanceStr, fmt.Sprintf("%s.ipv4.%d.port", resolverPath, k), value)
								}
							}
						}
						
						// Convert IPv6 ports
						if ipv6Array := resolver.Get("ipv6"); ipv6Array.Exists() && ipv6Array.IsArray() {
							for k, ipv6 := range ipv6Array.Array() {
								if port := ipv6.Get("port"); port.Exists() {
									value := state.ConvertToFloat64(port)
									instanceStr, _ = sjson.Set(instanceStr, fmt.Sprintf("%s.ipv6.%d.port", resolverPath, k), value)
								}
							}
						}
					} else {
						instanceStr, _ = sjson.Delete(instanceStr, basePath+".dns_resolvers")
					}
				}
			}
				
			// Handle L4 override (convert from array to object and handle port conversion)
			if l4override := settings.Get("l4override"); l4override.Exists() {
				if l4override.IsArray() {
					if len(l4override.Array()) > 0 {
						l4 := l4override.Array()[0]
						instanceStr, _ = sjson.Set(instanceStr, basePath+".l4override", l4.Value())
						if port := l4.Get("port"); port.Exists() {
							value := state.ConvertToFloat64(port)
							instanceStr, _ = sjson.Set(instanceStr, basePath+".l4override.port", value)
						}
					} else {
						instanceStr, _ = sjson.Delete(instanceStr, basePath+".l4override")
					}
				}
			}
			
			// Handle check_session (convert from array to object and handle duration conversion)
			if checkSession := settings.Get("check_session"); checkSession.Exists() {
				if checkSession.IsArray() {
					if len(checkSession.Array()) > 0 {
						session := checkSession.Array()[0]
						instanceStr, _ = sjson.Set(instanceStr, basePath+".check_session", session.Value())
						if duration := session.Get("duration"); duration.Exists() {
							value := state.ConvertToFloat64(duration)
							instanceStr, _ = sjson.Set(instanceStr, basePath+".check_session.duration", value)
						}
					} else {
						instanceStr, _ = sjson.Delete(instanceStr, basePath+".check_session")
					}
				}
			}
			
			// Handle egress (convert from array to object if needed)
			if egress := settings.Get("egress"); egress.Exists() {
				if egress.IsArray() {
					if len(egress.Array()) > 0 {
						instanceStr, _ = sjson.Set(instanceStr, basePath+".egress", egress.Array()[0].Value())
					} else {
						instanceStr, _ = sjson.Delete(instanceStr, basePath+".egress")
					}
				}
			}
			
			// Handle payload_log (convert from array to object if needed)
			if payloadLog := settings.Get("payload_log"); payloadLog.Exists() {
				if payloadLog.IsArray() {
					if len(payloadLog.Array()) > 0 {
						instanceStr, _ = sjson.Set(instanceStr, basePath+".payload_log", payloadLog.Array()[0].Value())
					} else {
						instanceStr, _ = sjson.Delete(instanceStr, basePath+".payload_log")
					}
				}
			}
			
			// Handle untrusted_cert (convert from array to object if needed)
			if untrustedCert := settings.Get("untrusted_cert"); untrustedCert.Exists() {
				if untrustedCert.IsArray() {
					if len(untrustedCert.Array()) > 0 {
						instanceStr, _ = sjson.Set(instanceStr, basePath+".untrusted_cert", untrustedCert.Array()[0].Value())
					} else {
						instanceStr, _ = sjson.Delete(instanceStr, basePath+".untrusted_cert")
					}
				}
			}
			
			// Handle audit_ssh (convert from array to object if needed)
			if auditSsh := settings.Get("audit_ssh"); auditSsh.Exists() {
				if auditSsh.IsArray() {
					if len(auditSsh.Array()) > 0 {
						instanceStr, _ = sjson.Set(instanceStr, basePath+".audit_ssh", auditSsh.Array()[0].Value())
					} else {
						// Empty array, remove it (v5 doesn't accept empty arrays)
						instanceStr, _ = sjson.Delete(instanceStr, basePath+".audit_ssh")
					}
				}
			}
			
			// Handle block_page (convert from array to object if needed)
			if blockPage := settings.Get("block_page"); blockPage.Exists() {
				if blockPage.IsArray() {
					if len(blockPage.Array()) > 0 {
						instanceStr, _ = sjson.Set(instanceStr, basePath+".block_page", blockPage.Array()[0].Value())
					} else {
						instanceStr, _ = sjson.Delete(instanceStr, basePath+".block_page")
					}
				}
			}
			
			// Handle quarantine (convert from array to object if needed)
			if quarantine := settings.Get("quarantine"); quarantine.Exists() {
				if quarantine.IsArray() {
					if len(quarantine.Array()) > 0 {
						instanceStr, _ = sjson.Set(instanceStr, basePath+".quarantine", quarantine.Array()[0].Value())
					} else {
						instanceStr, _ = sjson.Delete(instanceStr, basePath+".quarantine")
					}
				}
			}
			
			// Handle redirect (convert from array to object if needed)
			if redirect := settings.Get("redirect"); redirect.Exists() {
				if redirect.IsArray() {
					if len(redirect.Array()) > 0 {
						instanceStr, _ = sjson.Set(instanceStr, basePath+".redirect", redirect.Array()[0].Value())
					} else {
						instanceStr, _ = sjson.Delete(instanceStr, basePath+".redirect")
					}
				}
			}
			
			// Handle resolve_dns_internally (convert from array to object if needed)
			if resolveDns := settings.Get("resolve_dns_internally"); resolveDns.Exists() {
				if resolveDns.IsArray() {
					if len(resolveDns.Array()) > 0 {
						instanceStr, _ = sjson.Set(instanceStr, basePath+".resolve_dns_internally", resolveDns.Array()[0].Value())
					} else {
						instanceStr, _ = sjson.Delete(instanceStr, basePath+".resolve_dns_internally")
					}
				}
			}
			}
		}
	}
	
	// Always set schema_version to 0 for v5
	instanceStr, _ = sjson.Set(instanceStr, "schema_version", 0)
	
	return instanceStr, nil
}

// hashCodeString replicates the v4 provider's hash function for rule names
// This is needed to reverse the precedence transformation
func hashCodeString(s string) int {
	v := int(crc32.ChecksumIEEE([]byte(s)))
	if v >= 0 {
		return v
	}
	if -v >= 0 {
		return -v
	}
	// v == MinInt
	return 0
}

