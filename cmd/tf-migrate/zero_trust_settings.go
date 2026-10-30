package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/cloudflare/tf-migrate/internal/zerotrustsettings"
	"github.com/spf13/cobra"
)

func newPreserveZeroTrustSettingsCommand(cfg *config) *cobra.Command {
	var snapshotPath string
	cmd := &cobra.Command{
		Use:   "preserve-zero-trust-settings",
		Short: "Copy pre-migration Zero Trust API settings into explicit v5 HCL",
		Long: `After migrating HCL, restore settings unavailable in v4 from a JSON snapshot.
The snapshot maps local v5 resource addresses to their pre-migration GET responses.
Existing explicit configuration wins, including explicit null, false and empty values.
Only fields the v4 provider could not express are copied, including nested ones such
as settings.block_page.mode; a nested field merges into its parent object when that
object is already configured, and creates it when it is not.
Run in each module directory before the first v5 apply; no state or API is changed.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			data, err := os.ReadFile(snapshotPath)
			if err != nil {
				return fmt.Errorf("read Zero Trust settings snapshot: %w", err)
			}
			var snapshot zerotrustsettings.Snapshot
			if err := json.Unmarshal(data, &snapshot); err != nil {
				return fmt.Errorf("parse Zero Trust settings snapshot: %w", err)
			}
			directory := cfg.configDir
			if directory == "" {
				directory = "."
			}
			paths, err := filepath.Glob(filepath.Join(directory, "*.tf"))
			if err != nil {
				return err
			}
			files := make(map[string][]byte, len(paths))
			for _, path := range paths {
				files[path], err = os.ReadFile(path)
				if err != nil {
					return err
				}
			}
			updated, err := zerotrustsettings.Restore(files, snapshot)
			if err != nil {
				return err
			}
			paths = paths[:0]
			for path := range updated {
				paths = append(paths, path)
			}
			sort.Strings(paths)
			for _, path := range paths {
				if cfg.dryRun {
					fmt.Fprintf(cmd.OutOrStdout(), "# %s\n%s\n", path, updated[path])
					continue
				}
				if err := os.WriteFile(path, updated[path], 0600); err != nil {
					return fmt.Errorf("write %s: %w", path, err)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Restored migration settings in %s\n", path)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&snapshotPath, "snapshot", "", "JSON file mapping local v5 resource addresses to pre-migration API responses")
	_ = cmd.MarkFlagRequired("snapshot")
	return cmd
}
