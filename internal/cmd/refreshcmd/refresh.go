// Package refreshcmd implements `gro refresh` — the top-level cache
// control surface per cli-common/docs/working-with-state.md §4.6.
package refreshcmd

import (
	"context"
	"encoding/json"
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
	var (
		statusOnly bool
		jsonOut    bool
	)

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
  gro refresh --status

  # Control-plane envelope (scripts)
  gro refresh --status --json`,
		Args:      cobra.OnlyValidArgs,
		ValidArgs: validResources,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd.Context(), cmd.OutOrStdout(), args, statusOnly, jsonOut, newClient)
		},
	}

	cmd.Flags().BoolVar(&statusOnly, "status", false, "Print cache freshness; no network calls")
	cmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "Emit a JSON control-plane envelope")
	return cmd
}

func run(ctx context.Context, stdout io.Writer, args []string, statusOnly, jsonOut bool, newClient ClientFactory) error {
	selected := args
	if len(selected) == 0 {
		selected = validResources
	}

	if statusOnly {
		return runStatus(stdout, selected, jsonOut)
	}
	return runRefresh(ctx, stdout, selected, jsonOut, newClient)
}

// statusEntry is the per-resource envelope element for --status --json.
type statusEntry struct {
	Resource  string    `json:"resource"`
	FetchedAt time.Time `json:"fetched_at,omitempty"`
	TTL       string    `json:"ttl"`
	Status    string    `json:"status"`
}

// refreshEntry is the per-resource envelope element for the refresh path.
type refreshEntry struct {
	Resource  string    `json:"resource"`
	Count     int       `json:"count"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
	Error     string    `json:"error,omitempty"`
}

func runStatus(stdout io.Writer, selected []string, jsonOut bool) error {
	c, err := cache.New()
	if err != nil {
		return fmt.Errorf("initializing cache: %w", err)
	}

	var now time.Time
	entries := make([]statusEntry, 0, len(selected))
	for _, name := range selected {
		// Only "drives" is in validResources today; cobra rejects anything else
		// before RunE, so there is no need for an in-loop default case.
		fetchedAt, ttl, status, statusNow, err := c.DrivesStatus()
		if err != nil {
			return err
		}
		now = statusNow
		entries = append(entries, statusEntry{
			Resource:  name,
			FetchedAt: fetchedAt,
			TTL:       ttl,
			Status:    status.String(),
		})
	}

	if jsonOut {
		return writeJSON(stdout, map[string]any{"resources": entries})
	}
	if _, err := fmt.Fprintln(stdout, "RESOURCE | FETCHED_AT | AGE | TTL | STATUS"); err != nil {
		return err
	}
	for _, e := range entries {
		at := "-"
		age := "-"
		if !e.FetchedAt.IsZero() {
			at = e.FetchedAt.UTC().Format(time.RFC3339)
			age = clicache.Age(e.FetchedAt, now)
		}
		if _, err := fmt.Fprintf(stdout, "%s | %s | %s | %s | %s\n", e.Resource, at, age, e.TTL, e.Status); err != nil {
			return err
		}
	}
	return nil
}

func runRefresh(ctx context.Context, stdout io.Writer, selected []string, jsonOut bool, newClient ClientFactory) error {
	c, err := cache.New()
	if err != nil {
		return fmt.Errorf("initializing cache: %w", err)
	}

	client, err := newClient(ctx)
	if err != nil {
		return err
	}

	entries := make([]refreshEntry, 0, len(selected))
	var firstErr error
	for _, name := range selected {
		count, err := refreshDrives(ctx, client, c)
		entry := refreshEntry{Resource: name, Count: count, UpdatedAt: time.Now().UTC()}
		if err != nil {
			entry.Error = err.Error()
			entry.UpdatedAt = time.Time{}
			if firstErr == nil {
				firstErr = fmt.Errorf("refreshing %s: %w", name, err)
			}
		}
		entries = append(entries, entry)
	}

	if jsonOut {
		if writeErr := writeJSON(stdout, map[string]any{"resources": entries}); writeErr != nil {
			return writeErr
		}
	} else {
		for _, e := range entries {
			if e.Error != "" {
				if _, err := fmt.Fprintf(stdout, "Refreshing %s failed: %s\n", e.Resource, e.Error); err != nil {
					return err
				}
				continue
			}
			if _, err := fmt.Fprintf(stdout, "Refreshing %s... %d entries - cache updated at %s\n",
				e.Resource, e.Count, e.UpdatedAt.Format(time.RFC3339)); err != nil {
				return err
			}
		}
	}

	return firstErr
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

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
