// Package zerotrustsettings restores settings unavailable in the v4 provider
// from an API snapshot into explicit v5 configuration. It does not modify state
// or change the provider's normal handling of omitted configuration.
package zerotrustsettings

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

// Snapshot maps local v5 resource addresses to their pre-migration GET response.
// Each response may be either the API result object or the full API envelope.
// Run restoration in each module directory using addresses local to that module.
type Snapshot map[string]json.RawMessage

var fields = map[string][]string{
	"cloudflare_zero_trust_organization": {
		"deny_unmatched_requests", "deny_unmatched_requests_exempted_zone_names",
		"mfa_config", "mfa_piv_key_requirements", "mfa_required_for_all_apps",
	},
	"cloudflare_zero_trust_gateway_settings": {"host_selector", "sandbox"},
	"cloudflare_zero_trust_device_settings": {
		"external_emergency_signal_enabled", "external_emergency_signal_url",
		"external_emergency_signal_fingerprint", "external_emergency_signal_interval",
	},
}

// Restore returns only changed files, after validating the entire input. Existing
// configuration, including explicit false, null, and empty values, always wins.
func Restore(files map[string][]byte, snapshot Snapshot) (map[string][]byte, error) {
	if len(snapshot) == 0 {
		return nil, fmt.Errorf("Zero Trust settings snapshot is empty")
	}
	parsed := make(map[string]*hclwrite.File, len(files))
	blocks := make(map[string]*hclwrite.Block)
	owners := make(map[string]string)
	for filename, content := range files {
		file, diags := hclwrite.ParseConfig(content, filename, hcl.InitialPos)
		if diags.HasErrors() {
			return nil, fmt.Errorf("parse %s: %s", filename, diags.Error())
		}
		parsed[filename] = file
		for _, block := range file.Body().Blocks() {
			if block.Type() != "resource" || len(block.Labels()) != 2 {
				continue
			}
			address := strings.Join(block.Labels(), ".")
			if blocks[address] != nil {
				return nil, fmt.Errorf("duplicate resource %s", address)
			}
			blocks[address], owners[address] = block, filename
		}
	}
	addresses := make([]string, 0, len(snapshot))
	for address := range snapshot {
		addresses = append(addresses, address)
	}
	sort.Strings(addresses)
	changed := make(map[string]bool)
	for _, address := range addresses {
		block := blocks[address]
		if block == nil {
			return nil, fmt.Errorf("snapshot resource %s has no matching v5 resource block; use local addresses in the migrated module directory", address)
		}
		resourceType := block.Labels()[0]
		allowed, supported := fields[resourceType]
		if !supported {
			return nil, fmt.Errorf("snapshot resource %s is not a supported Zero Trust settings resource", address)
		}
		body := block.Body()
		if body.GetAttribute("count") != nil || body.GetAttribute("for_each") != nil {
			return nil, fmt.Errorf("%s uses count or for_each: restore its settings per instance explicitly instead of copying a single API response to all instances", address)
		}
		values, err := resultObject(snapshot[address])
		if err != nil {
			return nil, fmt.Errorf("snapshot resource %s: %w", address, err)
		}
		if resourceType == "cloudflare_zero_trust_gateway_settings" {
			var settings map[string]json.RawMessage
			if err := json.Unmarshal(values["settings"], &settings); err != nil || settings == nil {
				return nil, fmt.Errorf("snapshot resource %s: expected a settings object", address)
			}
			values = settings
		}
		attrs := make([]hclwrite.ObjectAttrTokens, 0, len(allowed))
		for _, name := range allowed {
			raw, present := values[name]
			if !present || string(raw) == "null" {
				continue
			}
			// cty produces HCL literals with correct escaping, including ${...}
			// in strings. JSON numbers retain their precision.
			typ, err := ctyjson.ImpliedType(raw)
			if err != nil {
				return nil, fmt.Errorf("snapshot resource %s field %s: %w", address, name, err)
			}
			value, err := ctyjson.Unmarshal(raw, typ)
			if err != nil {
				return nil, fmt.Errorf("snapshot resource %s field %s: %w", address, name, err)
			}
			if resourceType == "cloudflare_zero_trust_gateway_settings" {
				attrs = append(attrs, hclwrite.ObjectAttrTokens{Name: hclwrite.TokensForIdentifier(name), Value: hclwrite.TokensForValue(value)})
			} else if body.GetAttribute(name) == nil {
				body.SetAttributeValue(name, value)
				changed[owners[address]] = true
			}
		}
		if len(attrs) > 0 {
			updated, err := restoreGateway(body, attrs)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", address, err)
			}
			changed[owners[address]] = changed[owners[address]] || updated
		}
	}
	output := make(map[string][]byte, len(changed))
	for filename, updated := range changed {
		if updated {
			output[filename] = hclwrite.Format(parsed[filename].Bytes())
		}
	}
	return output, nil
}

func resultObject(raw json.RawMessage) (map[string]json.RawMessage, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil || object == nil {
		return nil, fmt.Errorf("expected an API result object or response envelope")
	}
	if success, present := object["success"]; present && string(success) != "true" {
		return nil, fmt.Errorf("API snapshot response was unsuccessful")
	}
	if result, present := object["result"]; present {
		object = nil
		if err := json.Unmarshal(result, &object); err != nil || object == nil {
			return nil, fmt.Errorf("API response result must be an object")
		}
	}
	return object, nil
}

func restoreGateway(body *hclwrite.Body, restored []hclwrite.ObjectAttrTokens) (bool, error) {
	attr := body.GetAttribute("settings")
	if attr == nil {
		body.SetAttributeRaw("settings", hclwrite.TokensForObject(restored))
		return true, nil
	}
	tokens := attr.Expr().BuildTokens(nil)
	expr, diags := hclsyntax.ParseExpression(tokens.Bytes(), "settings", hcl.InitialPos)
	if diags.HasErrors() {
		return false, fmt.Errorf("parse settings: %s", diags.Error())
	}
	object, ok := expr.(*hclsyntax.ObjectConsExpr)
	if !ok {
		return false, fmt.Errorf("settings is an expression; add host_selector and sandbox explicitly to its source object before applying v5")
	}
	// Keep the original tokens to retain expressions and comments verbatim.
	start, end := -1, -1
	for i, token := range tokens {
		if token.Type == hclsyntax.TokenOBrace && start == -1 {
			start = i
		}
		if token.Type == hclsyntax.TokenCBrace {
			end = i
		}
	}
	existing := make(map[string]bool, len(object.Items))
	for _, item := range object.Items {
		key, d := item.KeyExpr.Value(nil)
		if d.HasErrors() || !key.IsKnown() || key.Type() != cty.String {
			return false, fmt.Errorf("settings contains a nonliteral key; restore its security settings explicitly")
		}
		existing[key.AsString()] = true
	}
	var additions hclwrite.Tokens
	for _, field := range restored {
		if !existing[string(field.Name.Bytes())] {
			if len(additions) == 0 {
				additions = append(additions, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})
			}
			additions = append(additions, field.Name...)
			additions = append(additions, &hclwrite.Token{Type: hclsyntax.TokenEqual, Bytes: []byte("=")})
			additions = append(additions, field.Value...)
			additions = append(additions, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})
		}
	}
	if len(additions) == 0 {
		return false, nil
	}
	if start < 0 || end <= start {
		return false, fmt.Errorf("cannot locate settings object")
	}
	merged := append(hclwrite.Tokens{}, tokens[:end]...)
	merged = append(merged, additions...)
	merged = append(merged, tokens[end:]...)
	body.SetAttributeRaw("settings", merged)
	return true, nil
}
