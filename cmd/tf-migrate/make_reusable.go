package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"

	"github.com/cloudflare/tf-migrate/internal/accesspolicyphase"
)

func newMakeAccessPoliciesReusableCommand(cfg *config) *cobra.Command {
	var idsPath, baseURL string
	cmd := &cobra.Command{
		Use:   "make-access-policies-reusable",
		Short: "Convert recorded application-scoped Access policies to reusable policies (same IDs)",
		Long: `Step 3 of 4 of the Access policy migration. Run it after the step 2 apply.

Reads the record written by tf-migrate:
  terraform output -json ` + accesspolicyphase.Name + ` > ids.json

and calls make_reusable for each policy, one at a time. The policy keeps its ID,
stays attached to its application with the same precedence, and its conditions
don't change. Each conversion is verified. Policies that are already reusable are
skipped, so it is safe to re-run. --dry-run only checks.

Credentials: CLOUDFLARE_API_TOKEN (Access: Apps and Policies Edit), or
CLOUDFLARE_EMAIL and CLOUDFLARE_API_KEY.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
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
			failed := 0
			for _, o := range c.MakeReusable(rec, cfg.dryRun) {
				line := fmt.Sprintf("%-40s %s  %s", o.Key, o.ID, o.Result)
				if o.Err != nil {
					failed++
					line += ": " + o.Err.Error()
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), line)
			}
			if failed > 0 {
				return fmt.Errorf("%d of %d policies failed; fix and re-run (converted ones are skipped)", failed, len(rec))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&idsPath, "ids", "", "file with the JSON from terraform output -json "+accesspolicyphase.Name+" (for a module, "+accesspolicyphase.Name+"_<module>)")
	cmd.Flags().StringVar(&baseURL, "api-url", "https://api.cloudflare.com/client/v4", "Cloudflare API base URL")
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
