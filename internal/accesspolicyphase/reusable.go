package accesspolicyphase

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// DefaultTimeout bounds each API request when Client.HTTP is nil.
const DefaultTimeout = 30 * time.Second

// RecordedPolicy is one entry of `terraform output -json tf_migrate_access_policies`.
type RecordedPolicy struct {
	ID            string `json:"id"`
	ApplicationID string `json:"application_id"`
	AccountID     string `json:"account_id"`
	ZoneID        string `json:"zone_id,omitempty"` // zone apps convert through the zone endpoint
}

// ParseRecord parses `terraform output -json tf_migrate_access_policies`.
// A for_each policy's entry is a map of instances, returned as name["key"].
func ParseRecord(data []byte) (map[string]RecordedPolicy, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse record: %w", err)
	}
	rec := map[string]RecordedPolicy{}
	for name, msg := range raw {
		var probe map[string]json.RawMessage
		if err := json.Unmarshal(msg, &probe); err != nil {
			return nil, fmt.Errorf("parse record entry %q: %w", name, err)
		}
		if _, single := probe["id"]; single {
			var p RecordedPolicy
			if err := json.Unmarshal(msg, &p); err != nil {
				return nil, fmt.Errorf("parse record entry %q: %w", name, err)
			}
			rec[name] = p
			continue
		}
		var instances map[string]RecordedPolicy
		if err := json.Unmarshal(msg, &instances); err != nil {
			return nil, fmt.Errorf("parse record entry %q: %w", name, err)
		}
		for k, p := range instances {
			rec[fmt.Sprintf("%s[%q]", name, k)] = p
		}
	}
	for k, p := range rec {
		if p.ID == "" || p.ApplicationID == "" || p.AccountID == "" {
			return nil, fmt.Errorf("record entry %q is missing id, application_id or account_id", k)
		}
	}
	return rec, nil
}

// Client calls the Cloudflare API.
type Client struct {
	BaseURL string // e.g. https://api.cloudflare.com/client/v4
	HTTP    *http.Client
	Auth    func(*http.Request)
}

// Outcome is what MakeReusable did for one policy.
type Outcome struct {
	Key, ID string
	Result  string // "converted", "already reusable", "would convert", "failed"
	Err     error
}

type apiPolicy struct {
	Success bool `json:"success"`
	Result  struct {
		ID       string `json:"id"`
		Reusable bool   `json:"reusable"`
	} `json:"result"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// MakeReusable converts each recorded policy to a reusable policy, one at a
// time, verifying each. It skips policies that are already reusable, so it
// can be re-run. dryRun only checks.
func (c *Client) MakeReusable(rec map[string]RecordedPolicy, dryRun bool) []Outcome {
	keys := make([]string, 0, len(rec))
	for k := range rec {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var out []Outcome
	for _, k := range keys {
		p := rec[k]
		o := Outcome{Key: k, ID: p.ID}
		reusable, err := c.isReusable(p)
		switch {
		case err != nil:
			o.Result, o.Err = "failed", err
		case reusable:
			o.Result = "already reusable"
		case dryRun:
			o.Result = "would convert"
		default:
			o.Result, o.Err = c.convert(p)
		}
		out = append(out, o)
	}
	return out
}

// isReusable reads the policy through the account-level endpoint, which
// only serves reusable policies (404 for an app-scoped one).
func (c *Client) isReusable(p RecordedPolicy) (bool, error) {
	status, body, err := c.do(http.MethodGet, fmt.Sprintf("/accounts/%s/access/policies/%s", url.PathEscape(p.AccountID), url.PathEscape(p.ID)))
	if err != nil {
		return false, err
	}
	if status == http.StatusNotFound {
		return false, nil
	}
	if status != http.StatusOK {
		return false, apiError(status, body)
	}
	var r apiPolicy
	if err := json.Unmarshal(body, &r); err != nil {
		return false, err
	}
	return r.Result.Reusable, nil
}

func (c *Client) convert(p RecordedPolicy) (string, error) {
	scope := "/accounts/" + url.PathEscape(p.AccountID)
	if p.ZoneID != "" {
		scope = "/zones/" + url.PathEscape(p.ZoneID)
	}
	status, body, err := c.do(http.MethodPut, fmt.Sprintf("%s/access/apps/%s/policies/%s/make_reusable", scope, url.PathEscape(p.ApplicationID), url.PathEscape(p.ID)))
	if err != nil {
		return "failed", err
	}
	if status != http.StatusOK && status != http.StatusConflict {
		return "failed", apiError(status, body)
	}
	var r apiPolicy
	if status == http.StatusOK {
		if err := json.Unmarshal(body, &r); err == nil && r.Result.ID != "" && r.Result.ID != p.ID {
			return "failed", fmt.Errorf("API returned a different policy ID %s", r.Result.ID)
		}
	}
	reusable, err := c.isReusable(p)
	if err != nil {
		return "failed", fmt.Errorf("verify: %w", err)
	}
	if !reusable {
		return "failed", fmt.Errorf("verify: policy is still not reusable")
	}
	if status == http.StatusConflict {
		return "already reusable", nil
	}
	return "converted", nil
}

func (c *Client) do(method, path string) (int, []byte, error) {
	req, err := http.NewRequest(method, strings.TrimRight(c.BaseURL, "/")+path, nil)
	if err != nil {
		return 0, nil, err
	}
	if c.Auth != nil {
		c.Auth(req)
	}
	hc := c.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: DefaultTimeout}
	}
	resp, err := hc.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	return resp.StatusCode, body, err
}

func apiError(status int, body []byte) error {
	var r apiPolicy
	if json.Unmarshal(body, &r) == nil && len(r.Errors) > 0 {
		msgs := make([]string, len(r.Errors))
		for i, e := range r.Errors {
			msgs[i] = e.Message
		}
		return fmt.Errorf("HTTP %d: %s", status, strings.Join(msgs, "; "))
	}
	return fmt.Errorf("HTTP %d", status)
}
