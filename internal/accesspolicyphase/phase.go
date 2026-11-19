// Package accesspolicyphase drives the phased, non-destructive migration of
// v4 application-scoped Access policies (cloudflare_access_policy with
// application_id) to v5 reusable policies.
//
// v5 has no application-scoped policy, and only the v4 provider can drop the
// old state entry. Recreating the policy changes its ID and briefly detaches
// it, so the migration keeps every policy ID instead:
//
//  1. Record (v4 apply): tf-migrate writes RecordFile into each directory with
//     app-scoped policies. It stores each policy's id, application_id,
//     account_id and precedence in terraform_data state and an output.
//  2. Removed (v4 apply): tf-migrate comments the policies out, adds
//     removed { lifecycle { destroy = false } } for each and freezes the
//     record so its values survive in state.
//  3. Between: `tf-migrate make-access-policies-reusable` converts each
//     recorded policy to a reusable one through the API (same ID).
//  4. v5: tf-migrate restores the policies as cloudflare_zero_trust_access_policy
//     with an import block whose ID is read from the record, and lists them
//     on the application's policies attribute.
//
// tf-migrate cannot read state, so the step is read from RecordFile.
package accesspolicyphase

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// Name is the local, terraform_data resource and output name of the record.
const Name = "tf_migrate_access_policies"

// RecordFile is the file tf-migrate writes the record into.
const RecordFile = "tf-migrate-access-policies.tf"

// State is the step a directory is at.
type State int

const (
	// NoRecord: the record has not been written.
	NoRecord State = iota
	// Recorded: the record is written and reads the live policies.
	Recorded
	// Frozen: the policies were removed from state; the record keeps the values.
	Frozen
)

func (s State) String() string {
	return [...]string{"no record", "recorded", "frozen"}[s]
}

var policyTypes = map[string]bool{
	"cloudflare_access_policy":            true,
	"cloudflare_zero_trust_access_policy": true,
}

var appTypes = map[string]bool{
	"cloudflare_access_application":            true,
	"cloudflare_zero_trust_access_application": true,
}

// Policy is an application-scoped policy the phased flow can migrate.
type Policy struct {
	Type       string // v4 resource type
	Name       string // resource name, also the record key
	App        string // name of the application resource it is attached to
	AccountID  string // account_id expression source; for a zone policy, read from the record
	ZoneID     string // zone_id expression source, zone policies only
	Precedence string // precedence expression source
	ForEach    bool   // the policy uses for_each; the record holds one entry per key
}

// RecordRef is the record expression for p (a map of instances when ForEach).
func (p Policy) RecordRef() string {
	return fmt.Sprintf("terraform_data.%s.output.%s", Name, p.Name)
}

// Inspect reports whether block is an application-scoped policy the phased
// flow supports. When it is app-scoped but unsupported, reason says why.
func Inspect(block *hclwrite.Block) (p Policy, ok bool, reason string) {
	if block.Type() != "resource" || len(block.Labels()) != 2 || !policyTypes[block.Labels()[0]] {
		return p, false, ""
	}
	body := block.Body()
	appAttr := body.GetAttribute("application_id")
	if appAttr == nil {
		return p, false, ""
	}
	p.Type, p.Name = block.Labels()[0], block.Labels()[1]
	app, isRef := appReference(appAttr)
	if isRef {
		p.App = app
	}
	switch {
	case !isRef:
		return p, false, "application_id is not a reference to an application resource in the same module"
	case body.GetAttribute("count") != nil:
		return p, false, "uses count"
	case body.GetAttribute("account_id") == nil && body.GetAttribute("zone_id") == nil:
		return p, false, "has neither account_id nor zone_id"
	case body.GetAttribute("precedence") == nil:
		return p, false, "has no precedence"
	}
	p.ForEach = body.GetAttribute("for_each") != nil
	if a := body.GetAttribute("account_id"); a != nil {
		p.AccountID = exprSource(a)
	} else {
		// v5 policies are account-level. v4 can't set account_id next to
		// zone_id, so the record looks the zone's account up.
		p.ZoneID = exprSource(body.GetAttribute("zone_id"))
		if p.ForEach && strings.Contains(p.ZoneID, "each.") {
			return p, false, "uses for_each with a per-instance zone_id"
		}
		p.AccountID = p.RecordRef() + ".account_id"
		if p.ForEach {
			p.AccountID = p.RecordRef() + "[each.key].account_id"
		}
	}
	p.Precedence = exprSource(body.GetAttribute("precedence"))
	return p, true, ""
}

// appReference returns NAME when attr is <application type>.NAME.id.
func appReference(attr *hclwrite.Attribute) (string, bool) {
	expr, diags := hclsyntax.ParseExpression([]byte(exprSource(attr)), "application_id", hcl.InitialPos)
	if diags.HasErrors() {
		return "", false
	}
	st, ok := expr.(*hclsyntax.ScopeTraversalExpr)
	if !ok || len(st.Traversal) != 3 || !appTypes[st.Traversal.RootName()] {
		return "", false
	}
	name, ok1 := st.Traversal[1].(hcl.TraverseAttr)
	id, ok2 := st.Traversal[2].(hcl.TraverseAttr)
	if !ok1 || !ok2 || id.Name != "id" {
		return "", false
	}
	return name.Name, true
}

func exprSource(attr *hclwrite.Attribute) string {
	return strings.TrimSpace(string(attr.Expr().BuildTokens(nil).Bytes()))
}

// StateOf reads the step a directory is at from its RecordFile.
func StateOf(dir string) State {
	content, err := os.ReadFile(filepath.Join(dir, RecordFile))
	if err != nil {
		return NoRecord
	}
	f, diags := hclwrite.ParseConfig(content, RecordFile, hcl.InitialPos)
	if diags.HasErrors() {
		return NoRecord
	}
	for _, b := range f.Body().Blocks() {
		if b.Type() == "resource" && len(b.Labels()) == 2 && b.Labels()[1] == Name {
			if b.Body().FirstMatchingBlock("lifecycle", nil) != nil {
				return Frozen
			}
			return Recorded
		}
	}
	return NoRecord
}

// Scan returns the app-scoped policies in dir's .tf files that the phased
// flow migrates, and a message for each one it leaves on the manual path.
//
// It decides per application, all or nothing: the v5 step writes the
// application's complete policies list, and a policy missing from that list
// would be detached on apply. So an application keeps the manual path when
// any of its policies is unsupported, when it already sets policies, or when
// some app-scoped policy in the module has an application_id that can't be
// resolved (it could belong to any application).
func Scan(dir string) ([]Policy, []string, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.tf"))
	if err != nil {
		return nil, nil, err
	}
	sort.Strings(paths)
	type candidate struct {
		p      Policy
		reason string // empty when supported
	}
	var cands []*candidate
	appsWithPolicies := map[string]bool{}
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, nil, err
		}
		f, diags := hclwrite.ParseConfig(content, filepath.Base(path), hcl.InitialPos)
		if diags.HasErrors() {
			continue
		}
		for _, b := range f.Body().Blocks() {
			if b.Type() == "resource" && len(b.Labels()) == 2 && appTypes[b.Labels()[0]] && b.Body().GetAttribute("policies") != nil {
				appsWithPolicies[b.Labels()[1]] = true
			}
			p, ok, reason := Inspect(b)
			if ok || reason != "" {
				cands = append(cands, &candidate{p: p, reason: reason})
			}
		}
	}

	names := map[string]int{}
	unresolved := ""
	for _, c := range cands {
		names[c.p.Name]++
		if c.p.App == "" && unresolved == "" {
			unresolved = c.p.Type + "." + c.p.Name
		}
	}
	for _, c := range cands {
		switch {
		case c.reason != "":
		case names[c.p.Name] > 1:
			c.reason = "another app-scoped policy has the same name"
		case appsWithPolicies[c.p.App]:
			c.reason = fmt.Sprintf("its application (%s) already sets policies", c.p.App)
		case unresolved != "":
			c.reason = fmt.Sprintf("%s has an application_id tf-migrate can't resolve, so it can't tell which application's policies list would be complete", unresolved)
		}
	}
	// All or nothing per application.
	blocked := map[string]string{}
	for _, c := range cands {
		if c.reason != "" && c.p.App != "" && blocked[c.p.App] == "" {
			blocked[c.p.App] = c.p.Type + "." + c.p.Name
		}
	}
	var policies []Policy
	var skipped []string
	for _, c := range cands {
		if c.reason == "" && blocked[c.p.App] != "" {
			c.reason = fmt.Sprintf("its application (%s) also has %s, which is not supported, and the application's policies list must be complete", c.p.App, blocked[c.p.App])
		}
		if c.reason == "" {
			policies = append(policies, c.p)
		} else {
			skipped = append(skipped, fmt.Sprintf("%s.%s: %s", c.p.Type, c.p.Name, c.reason))
		}
	}
	return policies, skipped, nil
}

var (
	cacheMu sync.Mutex
	cache   = map[string][]Policy{}
)

// PoliciesIn returns Scan(dir)'s policies, scanned once per directory per run.
// The first call must happen before any file in dir is rewritten; migrators
// call it when they see the first policy or application in the directory.
func PoliciesIn(dir string) []Policy {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if p, ok := cache[dir]; ok {
		return p
	}
	p, _, _ := Scan(dir)
	cache[dir] = p
	return p
}

// ResetCache clears PoliciesIn's cache (tests, or between phases).
func ResetCache() {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	cache = map[string][]Policy{}
}

// Find returns the supported policy typ.name in dir.
func Find(dir, typ, name string) (Policy, bool) {
	for _, p := range PoliciesIn(dir) {
		if p.Type == typ && p.Name == name {
			return p, true
		}
	}
	return Policy{}, false
}

// ForApp returns dir's supported policies attached to application appName,
// ordered by precedence when every precedence is a number, else source order.
func ForApp(dir, appName string) []Policy {
	var out []Policy
	for _, p := range PoliciesIn(dir) {
		if p.App == appName {
			out = append(out, p)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, errA := strconv.Atoi(out[i].Precedence)
		b, errB := strconv.Atoi(out[j].Precedence)
		return errA == nil && errB == nil && a < b
	})
	return out
}

// RenderRecord returns the step 1 record for policies.
func RenderRecord(policies []Policy) []byte {
	var b strings.Builder
	b.WriteString("# Written by tf-migrate (step 1 of the Access policy migration).\n")
	b.WriteString("# Records each application-scoped policy's ID so the v5 migration can import\n")
	b.WriteString("# it instead of recreating it. Don't edit; tf-migrate rewrites this file.\n")
	// One zone lookup per distinct zone_id expression, for zone policies.
	zones := map[string]string{}
	var zoneOrder []string
	for _, p := range policies {
		if p.ZoneID != "" && zones[p.ZoneID] == "" {
			zones[p.ZoneID] = fmt.Sprintf("%s_%d", Name, len(zoneOrder))
			zoneOrder = append(zoneOrder, p.ZoneID)
		}
	}
	b.WriteString("locals {\n  " + Name + " = {\n")
	for _, p := range policies {
		r := p.Type + "." + p.Name
		account := r + ".account_id"
		zone := ""
		if p.ZoneID != "" {
			account = "data.cloudflare_zone." + zones[p.ZoneID] + ".account_id"
			zone = "      zone_id        = " + r + ".zone_id\n"
		}
		if p.ForEach {
			account = strings.Replace(account, r+".", "i.", 1)
			zone = strings.Replace(zone, r+".", "i.", 1)
			fmt.Fprintf(&b, "    %s = { for k, i in %s : k => {\n      id             = i.id\n      application_id = i.application_id\n      account_id     = %s\n%s      precedence     = i.precedence\n    } }\n", p.Name, r, account, zone)
			continue
		}
		fmt.Fprintf(&b, "    %s = {\n      id             = %s.id\n      application_id = %s.application_id\n      account_id     = %s\n%s      precedence     = %s.precedence\n    }\n", p.Name, r, r, account, zone, r)
	}
	b.WriteString("  }\n}\n\n")
	for _, z := range zoneOrder {
		fmt.Fprintf(&b, "data \"cloudflare_zone\" %q {\n  zone_id = %s\n}\n\n", zones[z], z)
	}
	fmt.Fprintf(&b, "resource \"terraform_data\" %q {\n  input = local.%s\n}\n\n", Name, Name)
	fmt.Fprintf(&b, "output %q {\n  value = local.%s\n}\n", Name, Name)
	return hclwrite.Format([]byte(b.String()))
}

// RenderFrozen returns the step 2 record: state keeps the recorded values.
func RenderFrozen() []byte {
	return hclwrite.Format([]byte(fmt.Sprintf(`# Written by tf-migrate (step 2 of the Access policy migration).
# The policies are no longer in config. ignore_changes keeps the recorded IDs in
# state; the v5 migration imports the policies by these IDs. Keep this file:
# v5 policies migrated from zone_id read their account_id from it.
resource "terraform_data" %q {
  input = {}
  lifecycle {
    ignore_changes = [input]
  }
}

output %q {
  value = terraform_data.%s.output
}
`, Name, Name, Name)))
}

// ImportID returns the v5 import ID expression source for p. For a for_each
// policy it is evaluated in an import block iterating over p.RecordRef().
func ImportID(p Policy) string {
	if p.ForEach {
		return `"${each.value.account_id}/${each.value.id}"`
	}
	acct := "${" + p.AccountID + "}"
	if s, err := strconv.Unquote(p.AccountID); err == nil && !strings.Contains(s, "${") {
		acct = s
	}
	return fmt.Sprintf(`"%s/${terraform_data.%s.output.%s.id}"`, acct, Name, p.Name)
}

// ModuleCall is a call of a local module whose directory has app-scoped
// policies. Import blocks are only allowed in the root module, so the
// caller's directory gets the imports and re-exports the module's record.
type ModuleCall struct {
	CallerDir string // directory of the file with the module block
	Name      string // module block label
}

var (
	modulesMu sync.Mutex
	modules   = map[string]ModuleCall{}
)

// SetModules records which policy directories are local modules, keyed by
// module directory.
func SetModules(m map[string]ModuleCall) {
	modulesMu.Lock()
	defer modulesMu.Unlock()
	modules = m
}

// ModuleOf returns the call of the module in dir, if dir is a local module.
func ModuleOf(dir string) (ModuleCall, bool) {
	modulesMu.Lock()
	defer modulesMu.Unlock()
	c, ok := modules[filepath.Clean(dir)]
	return c, ok
}

// CallerFile is the file in the caller's directory for module call c.
func CallerFile(c ModuleCall) string {
	return filepath.Join(c.CallerDir, "tf-migrate-access-policies-"+c.Name+".tf")
}

// CallerOutput is the root output re-exporting module call c's record.
func CallerOutput(c ModuleCall) string { return Name + "_" + c.Name }

// RenderCaller returns the caller's file for module call c: the re-exported
// record and, in step 3, an import block per policy.
func RenderCaller(c ModuleCall, policies []Policy, imports bool) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "# Written by tf-migrate (Access policy migration) for module.%s.\n", c.Name)
	b.WriteString("# Re-exports the module's policy record; in the v5 step, imports the policies\n# (import blocks are only allowed in the root module).\n")
	fmt.Fprintf(&b, "output %q {\n  value = module.%s.%s\n}\n", CallerOutput(c), c.Name, Name)
	if imports {
		for _, p := range policies {
			rec := fmt.Sprintf("module.%s.%s.%s", c.Name, Name, p.Name)
			to := fmt.Sprintf("module.%s.cloudflare_zero_trust_access_policy.%s", c.Name, p.Name)
			if p.ForEach {
				fmt.Fprintf(&b, "\nimport {\n  for_each = %s\n  to       = %s[each.key]\n  id       = \"${each.value.account_id}/${each.value.id}\"\n}\n", rec, to)
				continue
			}
			fmt.Fprintf(&b, "\nimport {\n  to = %s\n  id = \"${%s.account_id}/${%s.id}\"\n}\n", to, rec, rec)
		}
	}
	return hclwrite.Format([]byte(b.String()))
}
