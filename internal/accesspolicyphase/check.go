package accesspolicyphase

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// CheckFile is written next to RecordFile by
// `make-access-policies-reusable --check` and updated by the convert run.
const CheckFile = "tf-migrate-access-policies-check.json"

// CheckResult is the content of CheckFile. Keys are record keys: a policy
// name, or name["key"] for a for_each instance.
type CheckResult struct {
	// Checked holds the policies of complete applications: every policy the
	// API lists on the application is in the record. Step 2 only runs when
	// it covers every recorded policy.
	Checked map[string]string `json:"checked"`
	// SkippedApps are application IDs passed with --skip-app. Their policies
	// stay on the manual path.
	SkippedApps []string `json:"skipped_apps,omitempty"`
	// Skipped holds the record keys of the skipped applications' policies.
	Skipped map[string]string `json:"skipped,omitempty"`
	// Converted holds the policies the convert run made reusable (or found
	// reusable). Step 4 only runs when it covers Checked.
	Converted map[string]string `json:"converted,omitempty"`
}

// ReadCheck reads dir's CheckFile. ok is false when there is none.
func ReadCheck(dir string) (r CheckResult, ok bool, err error) {
	data, err := os.ReadFile(filepath.Join(dir, CheckFile))
	if os.IsNotExist(err) {
		return r, false, nil
	}
	if err != nil {
		return r, false, err
	}
	if err := json.Unmarshal(data, &r); err != nil {
		return r, false, fmt.Errorf("parse %s: %w", filepath.Join(dir, CheckFile), err)
	}
	return r, true, nil
}

// WriteCheck writes r to dir's CheckFile.
func WriteCheck(dir string, r CheckResult) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, CheckFile), append(data, '\n'), 0644)
}

// BaseName returns the policy resource name of a record key.
func BaseName(key string) string {
	if i := strings.IndexByte(key, '['); i >= 0 {
		return key[:i]
	}
	return key
}

// HasName reports whether some key in m belongs to the policy resource name.
func HasName(m map[string]string, name string) bool {
	for k := range m {
		if BaseName(k) == name {
			return true
		}
	}
	return false
}

// skippedNames returns the policy resource names dir's CheckFile skips.
func skippedNames(dir string) map[string]bool {
	r, ok, err := ReadCheck(dir)
	if !ok || err != nil {
		return nil
	}
	out := map[string]bool{}
	for k := range r.Skipped {
		out[BaseName(k)] = true
	}
	return out
}

// AttachedPolicy is one policy the API lists on an application.
type AttachedPolicy struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Decision   string `json:"decision"`
	Precedence int    `json:"precedence"`
}

// AppCheck is the completeness result for one application.
type AppCheck struct {
	AppID, AppName string
	Keys           []string         // record keys on this application
	Extra          []AttachedPolicy // attached in Cloudflare, not in the record
	Missing        []string         // record keys whose ID is not attached
	Skipped        bool
	Err            error
}

// Complete reports whether the application's record matches Cloudflare.
func (a AppCheck) Complete() bool {
	return a.Err == nil && len(a.Extra) == 0 && len(a.Missing) == 0
}

// CheckApps compares, per application, the policy IDs the API lists on it
// with the IDs the record holds for it. Step 4 writes the application's
// policies list from the record, so a policy attached in Cloudflare but
// missing from the record would be detached. skip lists application IDs to
// leave on the manual path; they are not called.
func (c *Client) CheckApps(rec map[string]RecordedPolicy, skip map[string]bool) []AppCheck {
	byApp := map[string][]string{}
	for k, p := range rec {
		byApp[p.ApplicationID] = append(byApp[p.ApplicationID], k)
	}
	apps := make([]string, 0, len(byApp))
	for a := range byApp {
		apps = append(apps, a)
	}
	sort.Strings(apps)
	var out []AppCheck
	for _, app := range apps {
		keys := byApp[app]
		sort.Strings(keys)
		ac := AppCheck{AppID: app, Keys: keys}
		if skip[app] {
			ac.Skipped = true
			out = append(out, ac)
			continue
		}
		p := rec[keys[0]]
		scope := "/accounts/" + url.PathEscape(p.AccountID)
		if p.ZoneID != "" {
			scope = "/zones/" + url.PathEscape(p.ZoneID)
		}
		attached, err := c.listAppPolicies(scope, app)
		if err != nil {
			ac.Err = err
			out = append(out, ac)
			continue
		}
		recorded := map[string]bool{}
		for _, k := range keys {
			recorded[rec[k].ID] = true
		}
		seen := map[string]bool{}
		for _, a := range attached {
			seen[a.ID] = true
			if !recorded[a.ID] {
				ac.Extra = append(ac.Extra, a)
			}
		}
		for _, k := range keys {
			if !seen[rec[k].ID] {
				ac.Missing = append(ac.Missing, k)
			}
		}
		if !ac.Complete() {
			ac.AppName = c.appName(scope, app)
		}
		out = append(out, ac)
	}
	return out
}

// listAppPolicies lists every policy attached to an application, following
// pagination.
func (c *Client) listAppPolicies(scope, app string) ([]AttachedPolicy, error) {
	perPage := c.PerPage
	if perPage <= 0 {
		perPage = 50
	}
	var all []AttachedPolicy
	for page := 1; ; page++ {
		status, body, err := c.do(http.MethodGet, fmt.Sprintf("%s/access/apps/%s/policies?page=%d&per_page=%d", scope, url.PathEscape(app), page, perPage))
		if err != nil {
			return nil, err
		}
		if status != http.StatusOK {
			return nil, apiError(status, body)
		}
		var r struct {
			Result     []AttachedPolicy `json:"result"`
			ResultInfo *struct {
				TotalPages int `json:"total_pages"`
				TotalCount int `json:"total_count"`
			} `json:"result_info"`
		}
		if err := json.Unmarshal(body, &r); err != nil {
			return nil, fmt.Errorf("list policies of application %s: %w", app, err)
		}
		all = append(all, r.Result...)
		switch {
		case r.ResultInfo == nil || len(r.Result) == 0:
			return all, nil
		case r.ResultInfo.TotalPages > 0 && page >= r.ResultInfo.TotalPages:
			return all, nil
		case r.ResultInfo.TotalPages == 0 && (r.ResultInfo.TotalCount == 0 || len(all) >= r.ResultInfo.TotalCount):
			return all, nil
		}
		if page >= 1000 {
			return nil, fmt.Errorf("list policies of application %s: more than 1000 pages", app)
		}
	}
}

func (c *Client) appName(scope, app string) string {
	status, body, err := c.do(http.MethodGet, fmt.Sprintf("%s/access/apps/%s", scope, url.PathEscape(app)))
	if err != nil || status != http.StatusOK {
		return app
	}
	var r struct {
		Result struct {
			Name string `json:"name"`
		} `json:"result"`
	}
	if json.Unmarshal(body, &r) != nil || r.Result.Name == "" {
		return app
	}
	return r.Result.Name
}

// Describe renders an incomplete application for the user.
func (a AppCheck) Describe() string {
	var b strings.Builder
	switch {
	case a.Err != nil:
		fmt.Fprintf(&b, "%s: could not list its policies: %v\n", a.AppID, a.Err)
		return b.String()
	case len(a.Extra) > 0:
		fmt.Fprintf(&b, "%s (%s): %d policies in Cloudflare, %d in your Terraform record\n", a.AppName, a.AppID, len(a.Extra)+len(a.Keys)-len(a.Missing), len(a.Keys))
		for _, p := range a.Extra {
			fmt.Fprintf(&b, "  not in Terraform: %q (%s, precedence %d, id %s)\n", p.Name, p.Decision, p.Precedence, p.ID)
		}
	}
	for _, k := range a.Missing {
		fmt.Fprintf(&b, "%s (%s): %s is in the record but not attached in Cloudflare\n", a.AppName, a.AppID, k)
	}
	return b.String()
}
