package drive

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/google-readonly/internal/cache"
	"github.com/open-cli-collective/google-readonly/internal/config"
	"github.com/open-cli-collective/google-readonly/internal/drive"
)

func newDrivesCommand() *cobra.Command {
	var (
		jsonOutput bool
		refresh    bool
	)

	cmd := &cobra.Command{
		Use:   "drives",
		Short: "List shared drives",
		Long: `List all Google Shared Drives accessible to you.

Results are cached locally for fast lookups. Use --refresh to force a refresh.

Examples:
  gro drive drives              # List shared drives (uses cache)
  gro drive drives --refresh    # Force refresh from API
  gro drive drives --json       # Output as JSON`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := newDriveClient(cmd.Context())
			if err != nil {
				return fmt.Errorf("creating Drive client: %w", err)
			}

			// Initialize cache
			ttl := config.GetCacheTTLHours()
			c, err := cache.New(ttl)
			if err != nil {
				return fmt.Errorf("initializing cache: %w", err)
			}

			var drives []*drive.SharedDrive

			// Try cache first unless refresh requested
			if !refresh {
				cached, err := c.GetDrives()
				if err != nil {
					return fmt.Errorf("reading cache: %w", err)
				}
				if cached != nil {
					// Convert from cache type to drive type
					drives = make([]*drive.SharedDrive, len(cached))
					for i, d := range cached {
						drives[i] = &drive.SharedDrive{
							ID:   d.ID,
							Name: d.Name,
						}
					}
				}
			}

			// Fetch from API if no cache hit
			if drives == nil {
				drives, err = client.ListSharedDrives(cmd.Context(), 100)
				if err != nil {
					return fmt.Errorf("listing shared drives: %w", err)
				}

				// Update cache
				cached := make([]*cache.CachedDrive, len(drives))
				for i, d := range drives {
					cached[i] = &cache.CachedDrive{
						ID:   d.ID,
						Name: d.Name,
					}
				}
				if err := c.SetDrives(cached); err != nil {
					// Non-fatal: warn but continue
					fmt.Fprintf(os.Stderr, "Warning: failed to update cache: %v\n", err)
				}
			}

			if len(drives) == 0 {
				if jsonOutput {
					fmt.Println("[]")
					return nil
				}
				fmt.Println("No shared drives found.")
				return nil
			}

			if jsonOutput {
				return printJSON(drives)
			}

			printSharedDrives(drives)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output results as JSON")
	cmd.Flags().BoolVar(&refresh, "refresh", false, "Force refresh from API (ignore cache)")

	return cmd
}

// printSharedDrives prints shared drives in a formatted table
func printSharedDrives(drives []*drive.SharedDrive) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tNAME")

	for _, d := range drives {
		_, _ = fmt.Fprintf(w, "%s\t%s\n", d.ID, d.Name)
	}

	_ = w.Flush()
}

// resolveDriveScope converts command flags to a DriveScope, resolving drive names via cache
func resolveDriveScope(ctx context.Context, client DriveClient, myDrive bool, driveFlag string) (drive.DriveScope, error) {
	// --my-drive flag
	if myDrive {
		return drive.DriveScope{MyDriveOnly: true}, nil
	}

	// No --drive flag: search all drives
	if driveFlag == "" {
		return drive.DriveScope{AllDrives: true}, nil
	}

	// --drive flag provided: resolve name or use as ID
	// If it looks like a drive ID (starts with 0A), use directly
	if looksLikeDriveID(driveFlag) {
		return drive.DriveScope{DriveID: driveFlag}, nil
	}

	// Try to resolve name via cache
	ttl := config.GetCacheTTLHours()
	c, err := cache.New(ttl)
	if err != nil {
		return drive.DriveScope{}, fmt.Errorf("initializing cache: %w", err)
	}

	cached, _ := c.GetDrives()
	if cached == nil {
		// Cache miss - fetch from API
		drives, err := client.ListSharedDrives(ctx, 100)
		if err != nil {
			return drive.DriveScope{}, fmt.Errorf("listing shared drives: %w", err)
		}

		// Update cache
		cached = make([]*cache.CachedDrive, len(drives))
		for i, d := range drives {
			cached[i] = &cache.CachedDrive{
				ID:   d.ID,
				Name: d.Name,
			}
		}
		_ = c.SetDrives(cached) // Ignore cache write errors
	}

	// Find by name (case-insensitive)
	nameLower := strings.ToLower(driveFlag)
	for _, d := range cached {
		if strings.ToLower(d.Name) == nameLower {
			return drive.DriveScope{DriveID: d.ID}, nil
		}
	}

	return drive.DriveScope{}, fmt.Errorf("shared drive not found: %s", driveFlag)
}

// looksLikeDriveID returns true if the string appears to be a Drive ID
// Shared drive IDs typically start with "0A"
func looksLikeDriveID(s string) bool {
	return len(s) > 10 && strings.HasPrefix(s, "0A")
}
