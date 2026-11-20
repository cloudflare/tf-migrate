package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cloudflare/tf-migrate/internal/accesspolicyphase"
)

func newMakeAccessPoliciesReusableCommand(cfg *config) *cobra.Command {
	var idsPath, baseURL string
	var check bool
	var skipApps []string
	cmd := &cobra.Command{
		Use:   "make-access-policies-reusable",
		Short: "Check, then convert, recorded application-scoped Access policies to reusable policies (same IDs)",
		Long: `Reads the record written by tf-migrate:
  terraform output -json ` + accesspolicyphase.Name + ` > ids.json

--check (after the step 1 apply, before step 2) is read-only. For each recorded
application it lists the policies attached in Cloudflare and compares their IDs
with the record. Step 4 writes the application's policies list from the record,
so a policy attached in Cloudflare but missing from the record (in another
module, behind a literal ID, created in the dashboard) would be detached. The
result goes to ` + accesspolicyphase.CheckFile + ` in --config-dir, next to the
record; step 2 runs only when it covers every recorded policy.

Without --check (step 3, after the step 2 apply) it repeats the check, then
calls make_reusable for each policy of a complete application, one at a time.
The policy keeps its ID, stays attached to its application with the same
precedence, and its conditions don't change. Each conversion is verified and
marked in ` + accesspolicyphase.CheckFile + `; step 4 runs only when every
checked policy is marked. Policies that are already reusable are skipped, so it
is safe to re-run. --dry-run only checks.

--skip-app <application ID> leaves an application on the manual path, for
example one with a policy you can't bring into this configuration.

Credentials: CLOUDFLARE_API_TOKEN (Access: Apps and Policies Edit; Read for
--check), or CLOUDFLARE_EMAIL and CLOUDFLARE_API_KEY.`,
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			dir := cfg.configDir
			if dir == "" {
				dir = "."
			}
			if _, err := os.Stat(filepath.Join(dir, accesspolicyphase.RecordFile)); err != nil {
				return fmt.Errorf("no %s in %s: pass --config-dir, the directory with the record (for a module, the module's directory)", accesspolicyphase.RecordFile, dir)
			}
			data, err := os.ReadFile(idsPath)
			if err != nil {
				return err
			}
			rec, err := accesspolicyphase.ParseRecord(data)
			if err != nil {
				return err
			}
			if len(rec) == 0 {
				// {} is what the output returns when step 2 was applied
				// without step 1: the policies may be out of state with no
				// IDs recorded.
				return fmt.Errorf("%s has no policies. Check that step 1 (the record) was applied: terraform state show terraform_data.%s", idsPath, accesspolicyphase.Name)
			}
			auth, err := cloudflareAuth()
			if err != nil {
				return err
			}
			c := &accesspolicyphase.Client{BaseURL: baseURL, Auth: auth}
			prev, hasPrev, err := accesspolicyphase.ReadCheck(dir)
			if err != nil {
				return err
			}
			// Once step 2 has run (record frozen), the set of phased policies
			// is fixed: the policies are out of state. A later --check only
			// reports, and the convert run never changes Checked or Skipped.
			frozen := accesspolicyphase.StateOf(dir) == accesspolicyphase.Frozen
			locked := !check || frozen
			switch {
			case frozen && !hasPrev:
				return fmt.Errorf("no %s in %s. Step 2 has run, so --check can't recreate it: restore it from git (git checkout -- %s), then re-run", accesspolicyphase.CheckFile, dir, filepath.Join(dir, accesspolicyphase.CheckFile))
			case locked && !hasPrev:
				return fmt.Errorf("no %s in %s. Run with --check after the step 1 apply, before step 2", accesspolicyphase.CheckFile, dir)
			}
			skip := map[string]bool{}
			for _, a := range prev.SkippedApps {
				skip[a] = true // carried forward: the skipped policies are no longer in the record
			}
			for _, a := range skipApps {
				if locked && !skip[a] {
					return fmt.Errorf("--skip-app %s: an application can only be skipped with --check, before step 2", a)
				}
				skip[a] = true
			}

			result := accesspolicyphase.CheckResult{Checked: map[string]string{}, Skipped: map[string]string{}, Converted: map[string]string{}}
			for k, id := range prev.Skipped {
				result.Skipped[k] = id
			}
			for a := range skip {
				result.SkippedApps = append(result.SkippedApps, a)
			}
			sort.Strings(result.SkippedApps)
			var incomplete, unchecked []accesspolicyphase.AppCheck
			complete := map[string]accesspolicyphase.RecordedPolicy{}
			for _, ac := range c.CheckApps(rec, skip) {
				switch {
				case ac.Skipped:
					for _, k := range ac.Keys {
						result.Skipped[k] = rec[k].ID
					}
					_, _ = fmt.Fprintf(out, "%s: skipped (--skip-app), %d policies stay on the manual path\n", ac.AppID, len(ac.Keys))
				case ac.Err != nil:
					unchecked = append(unchecked, ac)
				case ac.Complete():
					for _, k := range ac.Keys {
						result.Checked[k] = rec[k].ID
						complete[k] = rec[k]
					}
				default:
					incomplete = append(incomplete, ac)
					_, _ = fmt.Fprint(out, ac.Describe())
				}
			}
			if len(incomplete) > 0 {
				_, _ = fmt.Fprintln(out, "Nothing was changed for these applications. Do one of these, then re-run:")
				_, _ = fmt.Fprintln(out, "  - bring the policy into the record: define it next to its application (if it is in")
				_, _ = fmt.Fprintln(out, "    another module, move it into the application's module), or import it there")
				_, _ = fmt.Fprintln(out, "  - delete it in Cloudflare, or remove it from the application, if it isn't needed")
				if locked {
					_, _ = fmt.Fprintln(out, "Step 2 has run, so the application can't be skipped any more; step 4 waits until it is complete.")
				} else {
					_, _ = fmt.Fprintln(out, "  - skip the application with --skip-app <id> (it stays on the manual path)")
				}
			}
			for _, ac := range unchecked {
				_, _ = fmt.Fprintf(out, "%s: could not check: %v\n", ac.AppID, ac.Err)
			}
			if len(unchecked) > 0 {
				_, _ = fmt.Fprintf(out, "Could not check %d application(s). Nothing was changed for them and %s was not updated for them; re-run.\n", len(unchecked), accesspolicyphase.CheckFile)
			}

			if locked {
				// Keep step 2's decision. Marks stay for applications that
				// could not be checked; an incomplete application loses its
				// marks, so step 4 waits.
				result.Checked, result.Skipped, result.SkippedApps = prev.Checked, prev.Skipped, prev.SkippedApps
				convertible := map[string]accesspolicyphase.RecordedPolicy{}
				for k, p := range complete {
					if prev.Checked[k] == p.ID {
						convertible[k] = p
					}
				}
				complete = convertible
				for _, ac := range unchecked {
					for _, k := range ac.Keys {
						if id, ok := prev.Converted[k]; ok {
							result.Converted[k] = id
						}
					}
				}
			}
			for k, id := range prev.Converted {
				if _, ok := complete[k]; ok && result.Checked[k] == id {
					result.Converted[k] = id
				}
			}

			failed := 0
			if check {
				if locked {
					_, _ = fmt.Fprintf(out, "%d of %d checked policies on complete applications\n", len(complete), len(result.Checked))
				} else {
					_, _ = fmt.Fprintf(out, "%d policies on complete applications\n", len(complete))
				}
			} else {
				for _, o := range c.MakeReusable(complete, cfg.dryRun) {
					line := fmt.Sprintf("%-40s %s  %s", o.Key, o.ID, o.Result)
					if o.Err != nil {
						failed++
						line += ": " + o.Err.Error()
					} else if !cfg.dryRun {
						result.Converted[o.Key] = o.ID
					}
					_, _ = fmt.Fprintln(out, line)
				}
			}
			// A --check before step 2 that could not reach every application
			// leaves the file alone, so a transient error can't change which
			// policies step 2 takes.
			if !cfg.dryRun && (locked || len(unchecked) == 0) {
				if err := accesspolicyphase.WriteCheck(dir, result); err != nil {
					return err
				}
				_, _ = fmt.Fprintf(out, "wrote %s\n", filepath.Join(dir, accesspolicyphase.CheckFile))
			}
			var problems []string
			if len(incomplete) > 0 {
				problems = append(problems, fmt.Sprintf("%d application(s) incomplete", len(incomplete)))
			}
			if len(unchecked) > 0 {
				problems = append(problems, fmt.Sprintf("%d application(s) could not be checked", len(unchecked)))
			}
			if failed > 0 {
				problems = append(problems, fmt.Sprintf("%d of %d policies failed; fix and re-run (converted ones are skipped)", failed, len(complete)))
			}
			if len(problems) > 0 {
				return errors.New(strings.Join(problems, ", "))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&idsPath, "ids", "", "file with the JSON from terraform output -json "+accesspolicyphase.Name+" (for a module, "+accesspolicyphase.Name+"_<module>)")
	cmd.Flags().StringVar(&baseURL, "api-url", "https://api.cloudflare.com/client/v4", "Cloudflare API base URL")
	cmd.Flags().BoolVar(&check, "check", false, "only check that each recorded application's policies are all in the record (run before step 2)")
	cmd.Flags().StringSliceVar(&skipApps, "skip-app", nil, "application ID to leave on the manual path (repeatable)")
	_ = cmd.MarkFlagRequired("ids")
	return cmd
}

func cloudflareAuth() (func(*http.Request), error) {
	if t := os.Getenv("CLOUDFLARE_API_TOKEN"); t != "" {
		return func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+t) }, nil
	}
	email, key := os.Getenv("CLOUDFLARE_EMAIL"), os.Getenv("CLOUDFLARE_API_KEY")
	if email != "" && key != "" {
		return func(r *http.Request) {
			r.Header.Set("X-Auth-Email", email)
			r.Header.Set("X-Auth-Key", key)
		}, nil
	}
	return nil, fmt.Errorf("set CLOUDFLARE_API_TOKEN, or CLOUDFLARE_EMAIL and CLOUDFLARE_API_KEY")
}
