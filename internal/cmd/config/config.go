// Package config implements the gro config command and subcommands.
package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/google-readonly/internal/auth"
	"github.com/open-cli-collective/google-readonly/internal/gmail"
	"github.com/open-cli-collective/google-readonly/internal/keychain"
)

// NewCommand returns the config command with subcommands
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage gro configuration",
		Long:  "View and manage gro configuration and authentication status.",
	}

	cmd.AddCommand(newShowCommand())
	cmd.AddCommand(newTestCommand())
	cmd.AddCommand(newClearCommand())
	cmd.AddCommand(newCacheCommand())

	return cmd
}

func newShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Display configuration status",
		Long: `Display the current configuration status including:
- Credentials file location and status
- OAuth token storage location and status
- Token expiration time (if available)`,
		Args: cobra.NoArgs,
		RunE: runShow,
	}
}

func newTestCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "test",
		Short: "Test Gmail API connectivity",
		Long: `Test the Gmail API connection with current credentials.
This verifies that the OAuth token is valid and the API is accessible.`,
		Args: cobra.NoArgs,
		RunE: runTest,
	}
}

func newClearCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "clear",
		Short: "Remove stored OAuth token",
		Long: `Remove the stored OAuth token, forcing re-authentication on next use.

Note: This only removes the OAuth token (access/refresh tokens).
The credentials.json file (OAuth client config) is not removed.`,
		Args: cobra.NoArgs,
		RunE: runClear,
	}
}

func runShow(cmd *cobra.Command, _ []string) error {
	// Check credentials file
	credPath, err := auth.GetCredentialsPath()
	if err != nil {
		return fmt.Errorf("getting credentials path: %w", err)
	}

	credStatus := "OK"
	if _, err := os.Stat(credPath); os.IsNotExist(err) {
		credStatus = "Not found"
	}
	fmt.Printf("Credentials: %s (%s)\n", credPath, credStatus)

	// Check token
	tokenStatus := "Not found"
	var tokenExpiry string

	if keychain.HasStoredToken() {
		backend := keychain.GetStorageBackend()
		tokenStatus = string(backend)

		// Try to get token expiry
		if token, err := keychain.GetToken(); err == nil && !token.Expiry.IsZero() {
			if token.Expiry.Before(time.Now()) {
				tokenExpiry = fmt.Sprintf("Expired at %s", token.Expiry.Format(time.RFC3339))
			} else {
				remaining := time.Until(token.Expiry).Round(time.Minute)
				tokenExpiry = fmt.Sprintf("Expires %s (in %s)", token.Expiry.Format(time.RFC3339), remaining)
			}
		}
	}

	fmt.Printf("Token:       %s\n", tokenStatus)
	if tokenExpiry != "" {
		fmt.Printf("Expiry:      %s\n", tokenExpiry)
	}

	// Check if secure storage is being used
	if keychain.IsSecureStorage() {
		fmt.Println("Security:    Secure storage (system keychain)")
	} else if keychain.HasStoredToken() {
		fmt.Println("Security:    File storage (0600 permissions)")
	}

	// Show email if we can get it without triggering auth
	if keychain.HasStoredToken() && credStatus == "OK" {
		if client, err := gmail.NewClient(cmd.Context()); err == nil {
			if profile, err := client.GetProfile(); err == nil {
				fmt.Printf("Email:       %s\n", profile.EmailAddress)
			}
		}
	}

	// Show help if not fully configured
	if credStatus == "Not found" || tokenStatus == "Not found" {
		fmt.Println()
		fmt.Println("Run 'gro init' to complete setup.")
	}

	return nil
}

func runTest(cmd *cobra.Command, _ []string) error {
	fmt.Println("Testing Gmail API connection...")
	fmt.Println()

	// Check token exists
	if !keychain.HasStoredToken() {
		fmt.Println("  OAuth token: Not found")
		fmt.Println()
		fmt.Println("Run 'gro init' to authenticate.")
		return fmt.Errorf("no OAuth token found")
	}
	fmt.Println("  OAuth token: Found")

	// Try to create client (tests token validity)
	client, err := gmail.NewClient(cmd.Context())
	if err != nil {
		fmt.Println("  Token valid: FAILED")
		fmt.Println()
		fmt.Println("Token may be expired or revoked.")
		fmt.Println("Run 'gro config clear' then 'gro init' to re-authenticate.")
		return fmt.Errorf("creating client: %w", err)
	}
	fmt.Println("  Token valid: OK")

	// Test API access
	profile, err := client.GetProfile()
	if err != nil {
		fmt.Println("  Gmail API:   FAILED")
		return fmt.Errorf("accessing Gmail API: %w", err)
	}
	fmt.Println("  Gmail API:   OK")
	fmt.Printf("  Messages:    %d total\n", profile.MessagesTotal)

	fmt.Println()
	fmt.Printf("Authenticated as: %s\n", profile.EmailAddress)

	return nil
}

func runClear(_ *cobra.Command, _ []string) error {
	if !keychain.HasStoredToken() {
		fmt.Println("No OAuth token found to clear.")
		return nil
	}

	backend := keychain.GetStorageBackend()

	if err := keychain.DeleteToken(); err != nil {
		return fmt.Errorf("clearing token: %w", err)
	}

	fmt.Printf("Cleared OAuth token from %s.\n", backend)
	fmt.Println()
	fmt.Println("Note: credentials.json is not removed (contains OAuth client config, not user data).")
	fmt.Println("Run 'gro init' to re-authenticate.")

	return nil
}
