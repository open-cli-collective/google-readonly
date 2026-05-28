// Package refreshcmd implements `gro refresh` — the top-level cache
// control surface per cli-common/docs/working-with-state.md §4.6.
package refreshcmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	clicache "github.com/open-cli-collective/cli-common/cache"
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/google-readonly/internal/cache"
	"github.com/open-cli-collective/google-readonly/internal/drive"
)

// validResources is the closed set of resource names the gro cache exposes
// today. Kept local to refreshcmd (not exported from internal/cache) so the
// command's validation surface lives next to the command. Single-resource
// today; add an entry here when a second cached resource appears.
var validResources = []string{"drives"}

// DriveLister is the narrow seam refreshcmd needs from a Drive client — only
// ListSharedDrives. Matches the signature in internal/cmd/drive/output.go's
// DriveClient interface so the production factory satisfies it directly.
type DriveLister interface {
	ListSharedDrives(ctx context.Context, pageSize int64) ([]*drive.SharedDrive, error)
}

// ClientFactory constructs a DriveLister for the refresh path. The --status
// branch never invokes it; the invariant test in refresh_test.go pins that.
type ClientFactory func(ctx context.Context) (DriveLister, error)

// defaultClientFactory wraps internal/drive.NewClient so that init/auth
// errors surface through the same path as every other gro command.
func defaultClientFactory(ctx context.Context) (DriveLister, error) {
	return drive.NewClient(ctx)
}

// NewCommand registers `gro refresh` with the production client factory.
func NewCommand() *cobra.Command {
	return newCommandWithDeps(defaultClientFactory)
}

// newCommandWithDeps is the test seam.
func newCommandWithDeps(newClient ClientFactory) *cobra.Command {
	var statusOnly bool

	cmd := &cobra.Command{
		Use:   "refresh [resources...]",
		Short: "Refresh gro's local cache",
		Long: `Refresh gro's local cache of Google API metadata.

With no arguments, refreshes every cacheable resource. With resource names,
refreshes only those. With --status, reports freshness without fetching.

Today the only cached resource is "drives" (shared-drive name → ID lookup).`,
		Example: `  # Refresh everything
  gro refresh

  # Refresh a specific resource
  gro refresh drives

  # Show freshness without fetching
  gro refresh --status`,
		Args:      cobra.OnlyValidArgs,
		ValidArgs: validResources,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd.Context(), cmd.OutOrStdout(), args, statusOnly, newClient)
		},
	}

	cmd.Flags().BoolVar(&statusOnly, "status", false, "Print cache freshness; no network calls")
	return cmd
}

func run(ctx context.Context, stdout io.Writer, args []string, statusOnly bool, newClient ClientFactory) error {
	selected := args
	if len(selected) == 0 {
		selected = validResources
	}

	if statusOnly {
		return runStatus(stdout, selected)
	}
	return runRefresh(ctx, stdout, selected, newClient)
}

func runStatus(stdout io.Writer, selected []string) error {
	c, err := cache.New()
	if err != nil {
		return fmt.Errorf("initializing cache: %w", err)
	}

	_, _ = fmt.Fprintln(stdout, "RESOURCE | FETCHED_AT | AGE | TTL | STATUS")
	now := time.Now().UTC()
	for _, name := range selected {
		switch name {
		case "drives":
			fetchedAt, ttl, status, err := c.DrivesStatus()
			if err != nil {
				return err
			}
			at := "-"
			age := "-"
			if !fetchedAt.IsZero() {
				at = fetchedAt.UTC().Format(time.RFC3339)
				age = clicache.Age(fetchedAt, now)
			}
			_, _ = fmt.Fprintf(stdout, "%s | %s | %s | %s | %s\n", name, at, age, ttl, status)
		default:
			return fmt.Errorf("unknown resource: %s", name)
		}
	}
	return nil
}

func runRefresh(ctx context.Context, stdout io.Writer, selected []string, newClient ClientFactory) error {
	c, err := cache.New()
	if err != nil {
		return fmt.Errorf("initializing cache: %w", err)
	}

	client, err := newClient(ctx)
	if err != nil {
		return err
	}

	var failures []string
	for _, name := range selected {
		switch name {
		case "drives":
			count, err := refreshDrives(ctx, client, c)
			if err != nil {
				failures = append(failures, fmt.Sprintf("%s: %v", name, err))
				continue
			}
			_, _ = fmt.Fprintf(stdout, "Refreshing %s... %d entries - cache updated at %s\n",
				name, count, time.Now().UTC().Format(time.RFC3339))
		default:
			return fmt.Errorf("unknown resource: %s", name)
		}
	}

	if len(failures) > 0 {
		return errors.New(joinFailures(failures))
	}
	return nil
}

func refreshDrives(ctx context.Context, client DriveLister, c *cache.Cache) (int, error) {
	drives, err := client.ListSharedDrives(ctx, 100)
	if err != nil {
		return 0, fmt.Errorf("listing shared drives: %w", err)
	}
	cached := make([]*cache.CachedDrive, len(drives))
	for i, d := range drives {
		cached[i] = &cache.CachedDrive{ID: d.ID, Name: d.Name}
	}
	if err := c.SetDrives(cached); err != nil {
		return 0, err
	}
	return len(drives), nil
}

func joinFailures(failures []string) string {
	if len(failures) == 1 {
		return "refresh failed - " + failures[0]
	}
	out := fmt.Sprintf("refresh failed (%d resources):", len(failures))
	for _, f := range failures {
		out += "\n  - " + f
	}
	return out
}
