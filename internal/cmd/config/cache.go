package config

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/google-readonly/internal/cache"
	configpkg "github.com/open-cli-collective/google-readonly/internal/config"
	"github.com/open-cli-collective/google-readonly/internal/output"
)

func newCacheCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Manage cache settings",
		Long: `Manage cache settings for Drive metadata.

gro caches Drive metadata (like shared drive lists) to speed up repeated commands.
The cache TTL is configured during 'gro init' (default: 24 hours).`,
	}

	cmd.AddCommand(newCacheShowCommand())
	cmd.AddCommand(newCacheClearCommand())
	cmd.AddCommand(newCacheTTLCommand())

	return cmd
}

func newCacheShowCommand() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Display cache status",
		Long: `Display the current cache status including:
- Cache directory location
- Configured TTL
- Cached data status (when last updated, expiration)`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := configpkg.LoadConfig()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			c, err := cache.New(cfg.CacheTTLHours)
			if err != nil {
				return fmt.Errorf("failed to initialize cache: %w", err)
			}

			status, err := c.GetStatus()
			if err != nil {
				return fmt.Errorf("failed to get cache status: %w", err)
			}

			if jsonOutput {
				return output.JSONStdout(status)
			}

			fmt.Printf("Cache directory: %s\n", configpkg.ShortenPath(status.Dir))
			fmt.Printf("Cache TTL:       %d hours\n", status.TTLHours)
			fmt.Println()

			if status.DrivesCache != nil {
				fmt.Println("Shared Drives Cache:")
				fmt.Printf("  File:     %s\n", configpkg.ShortenPath(status.DrivesCache.Path))
				fmt.Printf("  Cached:   %s\n", status.DrivesCache.CachedAt.Local().Format("2006-01-02 15:04:05"))
				fmt.Printf("  Expires:  %s\n", status.DrivesCache.ExpiresAt.Local().Format("2006-01-02 15:04:05"))
				if status.DrivesCache.IsStale {
					fmt.Printf("  Status:   Stale (will refresh on next use)\n")
				} else {
					fmt.Printf("  Status:   Valid (%d drives cached)\n", status.DrivesCache.Count)
				}
			} else {
				fmt.Println("Shared Drives Cache: Not populated")
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output as JSON")

	return cmd
}

func newCacheClearCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "clear",
		Short: "Clear all cached data",
		Long:  `Remove all cached data. Cache will be repopulated on next use.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := configpkg.LoadConfig()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			c, err := cache.New(cfg.CacheTTLHours)
			if err != nil {
				return fmt.Errorf("failed to initialize cache: %w", err)
			}

			if err := c.Clear(); err != nil {
				return fmt.Errorf("failed to clear cache: %w", err)
			}

			fmt.Println("Cache cleared.")
			return nil
		},
	}
}

func newCacheTTLCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "ttl <hours>",
		Short: "Set cache TTL",
		Long: `Set the cache time-to-live in hours.

This affects how long cached data (like shared drive lists) is considered valid.
After the TTL expires, data will be refreshed from the API on next use.

Examples:
  gro config cache ttl 12     # Set TTL to 12 hours
  gro config cache ttl 48     # Set TTL to 48 hours`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ttl, err := strconv.Atoi(args[0])
			if err != nil || ttl <= 0 {
				return fmt.Errorf("invalid TTL value: must be a positive integer (hours)")
			}

			cfg, err := configpkg.LoadConfig()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			cfg.CacheTTLHours = ttl

			if err := configpkg.SaveConfig(cfg); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			fmt.Printf("Cache TTL set to %d hours.\n", ttl)
			return nil
		},
	}
}
