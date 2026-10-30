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
	// Entries may name a nested path with a dot. A nested field is merged into
	// its parent object when that object is already present in the migrated
	// configuration, and the parent is created when it is not.
	"cloudflare_zero_trust_gateway_settings": {
		"host_selector", "sandbox", "inspection", "max_ttl_secs",
		// block_page gained these in v5; the v4 provider could not express them,
		// so they exist only in the API response (BUGS-2159). read_only,
		// source_account and version are computed and are deliberately omitted.
		"block_page.mode", "block_page.include_context",
		"block_page.suppress_footer", "block_page.target_uri",
	},
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
		additions := newAdditions()
		for _, name := range allowed {
			parent, leaf, nested := strings.Cut(name, ".")

			source := values
			if nested {
				// The parent object is absent from the snapshot when the setting
				// was never configured live; there is then nothing to restore.
				raw, present := values[parent]
				if !present || string(raw) == "null" {
					continue
				}
				var object map[string]json.RawMessage
				if err := json.Unmarshal(raw, &object); err != nil || object == nil {
					return nil, fmt.Errorf("snapshot resource %s field %s: expected %s to be an object", address, name, parent)
				}
				source = object
			} else {
				leaf = parent
			}

			raw, present := source[leaf]
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
				target := additions
				if nested {
					target = additions.child(parent)
				}
				target.set(leaf, hclwrite.TokensForValue(value))
			} else if nested {
				return nil, fmt.Errorf("snapshot resource %s field %s: nested restoration is only supported for gateway settings", address, name)
			} else if body.GetAttribute(leaf) == nil {
				body.SetAttributeValue(leaf, value)
				changed[owners[address]] = true
			}
		}
		if !additions.empty() {
			updated, err := restoreGateway(body, additions)
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

// additionTree holds the values to merge into an object, keyed by field name and
// kept in insertion order so output is deterministic. A field is either a leaf
// value or a nested object of further additions, never both.
type additionTree struct {
	order    []string
	leaves   map[string]hclwrite.Tokens
	children map[string]*additionTree
}

func newAdditions() *additionTree {
	return &additionTree{leaves: map[string]hclwrite.Tokens{}, children: map[string]*additionTree{}}
}

func (t *additionTree) set(name string, value hclwrite.Tokens) {
	if _, seen := t.leaves[name]; !seen {
		t.order = append(t.order, name)
	}
	t.leaves[name] = value
}

func (t *additionTree) child(name string) *additionTree {
	if existing, ok := t.children[name]; ok {
		return existing
	}
	t.order = append(t.order, name)
	t.children[name] = newAdditions()
	return t.children[name]
}

func (t *additionTree) empty() bool { return len(t.order) == 0 }

// tokens renders the tree as a complete object, used when the target object does
// not exist in configuration yet.
func (t *additionTree) tokens() hclwrite.Tokens {
	attrs := make([]hclwrite.ObjectAttrTokens, 0, len(t.order))
	for _, name := range t.order {
		value, leaf := t.leaves[name]
		if !leaf {
			value = t.children[name].tokens()
		}
		attrs = append(attrs, hclwrite.ObjectAttrTokens{Name: hclwrite.TokensForIdentifier(name), Value: value})
	}
	return hclwrite.TokensForObject(attrs)
}

func restoreGateway(body *hclwrite.Body, additions *additionTree) (bool, error) {
	attr := body.GetAttribute("settings")
	if attr == nil {
		body.SetAttributeRaw("settings", additions.tokens())
		return true, nil
	}
	merged, changed, err := mergeObject(attr.Expr().BuildTokens(nil), additions, "settings")
	if err != nil || !changed {
		return false, err
	}
	body.SetAttributeRaw("settings", merged)
	return true, nil
}

// mergeObject adds missing fields to an object's token stream. The original
// tokens are reused so expressions and comments survive verbatim; configuration
// that is already present always wins, including explicit null and false.
func mergeObject(tokens hclwrite.Tokens, additions *additionTree, path string) (hclwrite.Tokens, bool, error) {
	expr, diags := hclsyntax.ParseExpression(tokens.Bytes(), path, hcl.InitialPos)
	if diags.HasErrors() {
		return nil, false, fmt.Errorf("parse %s: %s", path, diags.Error())
	}
	object, ok := expr.(*hclsyntax.ObjectConsExpr)
	if !ok {
		return nil, false, fmt.Errorf("%s is an expression; add %s explicitly to its source object before applying v5", path, strings.Join(additions.order, " and "))
	}
	existing := make(map[string]bool, len(object.Items))
	for _, item := range object.Items {
		key, d := item.KeyExpr.Value(nil)
		if d.HasErrors() || !key.IsKnown() || key.Type() != cty.String {
			return nil, false, fmt.Errorf("%s contains a nonliteral key; restore its security settings explicitly", path)
		}
		existing[key.AsString()] = true
	}

	// Merge into nested objects first. Each pass re-tokenises so the indices used
	// for the next splice stay valid.
	changed := false
	for _, name := range additions.order {
		child, nested := additions.children[name]
		if !nested || !existing[name] {
			continue
		}
		start, end, found := childValueTokens(tokens, name)
		if !found {
			return nil, false, fmt.Errorf("cannot locate %s.%s", path, name)
		}
		childMerged, childChanged, err := mergeObject(tokens[start:end], child, path+"."+name)
		if err != nil {
			return nil, false, err
		}
		if !childChanged {
			continue
		}
		spliced := append(hclwrite.Tokens{}, tokens[:start]...)
		spliced = append(spliced, childMerged...)
		spliced = append(spliced, tokens[end:]...)
		tokens = spliced
		changed = true
	}

	// Then append fields, and whole nested objects, that are absent entirely.
	var pending hclwrite.Tokens
	for _, name := range additions.order {
		if existing[name] {
			continue
		}
		value, leaf := additions.leaves[name]
		if !leaf {
			value = additions.children[name].tokens()
		}
		if len(pending) == 0 {
			pending = append(pending, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})
		}
		pending = append(pending, hclwrite.TokensForIdentifier(name)...)
		pending = append(pending, &hclwrite.Token{Type: hclsyntax.TokenEqual, Bytes: []byte("=")})
		pending = append(pending, value...)
		pending = append(pending, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})
	}
	if len(pending) == 0 {
		return tokens, changed, nil
	}

	start, end := -1, -1
	for i, token := range tokens {
		if token.Type == hclsyntax.TokenOBrace && start == -1 {
			start = i
		}
		if token.Type == hclsyntax.TokenCBrace {
			end = i
		}
	}
	if start < 0 || end <= start {
		return nil, false, fmt.Errorf("cannot locate %s object", path)
	}
	merged := append(hclwrite.Tokens{}, tokens[:end]...)
	merged = append(merged, pending...)
	merged = append(merged, tokens[end:]...)
	return merged, true, nil
}

// childValueTokens returns the half-open token range holding the value of key
// within an object, so a nested object can be rewritten without disturbing the
// surrounding tokens.
func childValueTokens(tokens hclwrite.Tokens, key string) (int, int, bool) {
	depth, outer := 0, -1
	for i := 0; i < len(tokens); i++ {
		switch tokens[i].Type {
		case hclsyntax.TokenOBrace:
			depth++
			if outer == -1 {
				outer = depth
			}
			continue
		case hclsyntax.TokenCBrace:
			depth--
			continue
		}
		if depth != outer {
			continue
		}
		name, next := objectKeyAt(tokens, i)
		if name != key {
			continue
		}
		if next >= len(tokens) || tokens[next].Type != hclsyntax.TokenEqual {
			continue
		}
		start := next + 1
		for start < len(tokens) && tokens[start].Type == hclsyntax.TokenNewline {
			start++
		}
		if start >= len(tokens) || tokens[start].Type != hclsyntax.TokenOBrace {
			return 0, 0, false
		}
		inner := 0
		for j := start; j < len(tokens); j++ {
			switch tokens[j].Type {
			case hclsyntax.TokenOBrace:
				inner++
			case hclsyntax.TokenCBrace:
				inner--
				if inner == 0 {
					return start, j + 1, true
				}
			}
		}
		return 0, 0, false
	}
	return 0, 0, false
}

// objectKeyAt reads an object key at index i, accepting both bare identifiers and
// quoted names, and returns the key with the index just past it.
func objectKeyAt(tokens hclwrite.Tokens, i int) (string, int) {
	switch tokens[i].Type {
	case hclsyntax.TokenIdent:
		return string(tokens[i].Bytes), i + 1
	case hclsyntax.TokenOQuote:
		if i+2 < len(tokens) && tokens[i+1].Type == hclsyntax.TokenQuotedLit && tokens[i+2].Type == hclsyntax.TokenCQuote {
			return string(tokens[i+1].Bytes), i + 3
		}
	}
	return "", i + 1
}
