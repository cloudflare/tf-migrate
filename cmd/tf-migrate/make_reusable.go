package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

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
		Args: cobra.NoArgs,
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
			if !check && !hasPrev {
				return fmt.Errorf("no %s in %s. Run with --check after the step 1 apply, before step 2", accesspolicyphase.CheckFile, dir)
			}
			skip := map[string]bool{}
			for _, a := range skipApps {
				skip[a] = true
			}
			if !check {
				// Step 2 already decided which applications are phased.
				prevSkip := map[string]bool{}
				for _, a := range prev.SkippedApps {
					prevSkip[a] = true
					skip[a] = true
				}
				for _, a := range skipApps {
					if !prevSkip[a] {
						return fmt.Errorf("--skip-app %s: an application can only be skipped with --check, before step 2", a)
					}
				}
			}

			result := accesspolicyphase.CheckResult{Checked: map[string]string{}, Skipped: map[string]string{}, Converted: map[string]string{}}
			incomplete := 0
			complete := map[string]accesspolicyphase.RecordedPolicy{}
			for _, ac := range c.CheckApps(rec, skip) {
				switch {
				case ac.Skipped:
					result.SkippedApps = append(result.SkippedApps, ac.AppID)
					for _, k := range ac.Keys {
						result.Skipped[k] = rec[k].ID
					}
					_, _ = fmt.Fprintf(out, "%s: skipped (--skip-app), %d policies stay on the manual path\n", ac.AppID, len(ac.Keys))
				case ac.Complete():
					for _, k := range ac.Keys {
						result.Checked[k] = rec[k].ID
						complete[k] = rec[k]
					}
				default:
					incomplete++
					_, _ = fmt.Fprint(out, ac.Describe())
				}
			}
			if !check {
				// Step 2 acted on the --check result; keep it. An application
				// that became incomplete since gets no marks, so step 4 waits.
				convertible := map[string]accesspolicyphase.RecordedPolicy{}
				for k, p := range complete {
					if prev.Checked[k] == p.ID {
						convertible[k] = p
					}
				}
				complete = convertible
				result.Checked, result.Skipped, result.SkippedApps = prev.Checked, prev.Skipped, prev.SkippedApps
			}
			for k, id := range prev.Converted {
				if _, ok := complete[k]; ok && result.Checked[k] == id {
					result.Converted[k] = id
				}
			}
			if incomplete > 0 {
				_, _ = fmt.Fprintln(out, "Nothing was changed for these applications. Do one of these, then re-run:")
				_, _ = fmt.Fprintln(out, "  - import the policy into this configuration")
				_, _ = fmt.Fprintln(out, "  - delete it in Cloudflare if it isn't needed")
				_, _ = fmt.Fprintln(out, "  - skip the application with --skip-app <id> (it stays on the manual path)")
			}

			failed := 0
			if check {
				_, _ = fmt.Fprintf(out, "%d policies on complete applications\n", len(result.Checked))
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
			if !cfg.dryRun {
				if err := accesspolicyphase.WriteCheck(dir, result); err != nil {
					return err
				}
				_, _ = fmt.Fprintf(out, "wrote %s\n", filepath.Join(dir, accesspolicyphase.CheckFile))
			}
			switch {
			case incomplete > 0 && failed > 0:
				return fmt.Errorf("%d application(s) incomplete, %d policies failed", incomplete, failed)
			case incomplete > 0:
				return fmt.Errorf("%d application(s) incomplete", incomplete)
			case failed > 0:
				return fmt.Errorf("%d of %d policies failed; fix and re-run (converted ones are skipped)", failed, len(complete))
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
