package initcmd

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"google.golang.org/api/googleapi"

	"github.com/open-cli-collective/google-readonly/internal/gmail"
	"github.com/open-cli-collective/google-readonly/internal/keychain"
)

var noVerify bool

// NewCommand returns the init command
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Set up Google API authentication",
		Long: `Guided setup for Google API OAuth authentication.

This command walks you through the OAuth flow with clear instructions.
After setup, you can use commands like 'gro mail search', 'gro mail read', etc.

Prerequisites:
  1. Create a Google Cloud project at https://console.cloud.google.com
  2. Enable the Gmail API (and Calendar/Drive APIs for future features)
  3. Create OAuth 2.0 credentials (Desktop app type)
  4. Download the credentials JSON file
  5. Save it to ~/.config/google-readonly/credentials.json`,
		Args: cobra.NoArgs,
		RunE: runInit,
	}

	cmd.Flags().BoolVar(&noVerify, "no-verify", false, "Skip connectivity verification after setup")

	return cmd
}

func runInit(cmd *cobra.Command, args []string) error {
	// Step 1: Check for credentials.json
	credPath, err := gmail.GetCredentialsPath()
	if err != nil {
		return fmt.Errorf("failed to get credentials path: %w", err)
	}

	shortPath := gmail.ShortenPath(credPath)
	if _, err := os.Stat(credPath); os.IsNotExist(err) {
		fmt.Println("Credentials file not found.")
		fmt.Println()
		printCredentialsInstructions(shortPath)
		return fmt.Errorf("credentials file not found at %s", shortPath)
	}
	fmt.Printf("Credentials: %s\n", shortPath)

	// Step 2: Load OAuth config
	config, err := gmail.GetOAuthConfig()
	if err != nil {
		return fmt.Errorf("failed to load OAuth config: %w", err)
	}

	// Step 3: Check if already authenticated
	if keychain.HasStoredToken() {
		fmt.Println("Token:       Found")
		fmt.Println()

		if !noVerify {
			err := verifyConnectivity()
			if err == nil {
				return nil
			}

			// Check if this is an auth error (401)
			if isAuthError(err) {
				fmt.Println()
				fmt.Println("Your OAuth token appears to be expired or revoked.")

				if promptReauth() {
					fmt.Println()
					fmt.Println("Clearing old token...")
					if delErr := keychain.DeleteToken(); delErr != nil {
						return fmt.Errorf("failed to clear token: %w", delErr)
					}
					// Fall through to OAuth flow below
				} else {
					fmt.Println()
					fmt.Println("You can manually clear the token with: gro config clear")
					return err
				}
			} else {
				// Non-auth error, just return it
				return err
			}
		} else {
			fmt.Println("Setup complete! Try: gro mail search \"is:unread\"")
			return nil
		}
	}

	// Step 4: Guide through OAuth flow
	fmt.Println("Token:       Not found - starting OAuth flow")
	fmt.Println()

	authURL := gmail.GetAuthURL(config)
	fmt.Println("Open this URL in your browser:")
	fmt.Println()
	fmt.Println(authURL)
	fmt.Println()
	fmt.Println("After clicking 'Allow', your browser will redirect to a localhost URL.")
	fmt.Println("This will show an error - that's expected!")
	fmt.Println()
	fmt.Println("Copy the ENTIRE URL from your browser's address bar and paste it here,")
	fmt.Println("or just paste the 'code' parameter value:")
	fmt.Println()
	fmt.Print("> ")

	// Read the full line (code may contain special characters)
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	code := extractAuthCode(input)
	if code == "" {
		return fmt.Errorf("no authorization code found in input")
	}

	// Step 5: Exchange code for token
	fmt.Println()
	fmt.Println("Exchanging authorization code...")

	ctx := context.Background()
	token, err := gmail.ExchangeAuthCode(ctx, config, code)
	if err != nil {
		return fmt.Errorf("failed to exchange authorization code: %w", err)
	}

	// Step 6: Save token
	if err := keychain.SetToken(token); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}
	fmt.Printf("Token saved to: %s\n", keychain.GetStorageBackend())

	// Step 7: Verify connectivity (unless --no-verify)
	if !noVerify {
		fmt.Println()
		return verifyConnectivity()
	}

	fmt.Println()
	fmt.Println("Setup complete! Try: gro mail search \"is:unread\"")
	return nil
}

// extractAuthCode extracts the authorization code from user input.
// It accepts either a full localhost redirect URL or just the code value.
func extractAuthCode(input string) string {
	input = strings.TrimSpace(input)

	// If it looks like a URL, try to extract the code parameter
	if strings.HasPrefix(input, "http://localhost") || strings.HasPrefix(input, "https://localhost") {
		if u, err := url.Parse(input); err == nil {
			// Return the code if present, empty string if not
			// (e.g., URL has ?error=access_denied instead of ?code=...)
			return u.Query().Get("code")
		}
		// URL parsing failed, return empty
		return ""
	}

	// Otherwise treat as raw code
	return input
}

// verifyConnectivity tests the Gmail API connection
func verifyConnectivity() error {
	fmt.Println("Verifying Gmail API connection...")

	client, err := gmail.NewClient(context.Background())
	if err != nil {
		fmt.Println("  OAuth token: FAILED")
		return fmt.Errorf("failed to create client: %w", err)
	}
	fmt.Println("  OAuth token: OK")

	// Get profile to verify connectivity and get email address
	profile, err := client.GetProfile()
	if err != nil {
		fmt.Println("  Gmail API:   FAILED")
		return fmt.Errorf("failed to access Gmail API: %w", err)
	}
	fmt.Println("  Gmail API:   OK")
	fmt.Printf("  Messages:    %d total\n", profile.MessagesTotal)
	fmt.Println()
	fmt.Printf("Authenticated as: %s\n", profile.EmailAddress)
	fmt.Println()
	fmt.Println("Setup complete! Try: gro mail search \"is:unread\"")
	return nil
}

func printCredentialsInstructions(credPath string) {
	fmt.Println("To set up Google API credentials:")
	fmt.Println()
	fmt.Println("1. Go to https://console.cloud.google.com")
	fmt.Println("2. Create a new project (or select an existing one)")
	fmt.Println("3. Enable the Gmail API:")
	fmt.Println("   - Go to 'APIs & Services' > 'Library'")
	fmt.Println("   - Search for 'Gmail API' and enable it")
	fmt.Println("4. Create OAuth credentials:")
	fmt.Println("   - Go to 'APIs & Services' > 'Credentials'")
	fmt.Println("   - Click 'Create Credentials' > 'OAuth client ID'")
	fmt.Println("   - Select 'Desktop app' as application type")
	fmt.Println("   - Download the JSON file")
	fmt.Printf("5. Save the downloaded file to:\n   %s\n", credPath)
	fmt.Println()
	fmt.Println("Then run 'gro init' again.")
}

// isAuthError checks if an error is a Google API authentication error (HTTP 401)
func isAuthError(err error) bool {
	if err == nil {
		return false
	}

	// Check for googleapi.Error
	var apiErr *googleapi.Error
	if ok := errorAs(err, &apiErr); ok {
		return apiErr.Code == http.StatusUnauthorized
	}

	// Also check error message for common auth error patterns
	errStr := err.Error()
	return strings.Contains(errStr, "401") &&
		(strings.Contains(errStr, "Invalid Credentials") ||
			strings.Contains(errStr, "invalid_grant") ||
			strings.Contains(errStr, "Token has been expired or revoked"))
}

// errorAs is a wrapper for errors.As to make testing easier
var errorAs = func(err error, target interface{}) bool {
	switch t := target.(type) {
	case **googleapi.Error:
		for e := err; e != nil; {
			if apiErr, ok := e.(*googleapi.Error); ok {
				*t = apiErr
				return true
			}
			if unwrapper, ok := e.(interface{ Unwrap() error }); ok {
				e = unwrapper.Unwrap()
			} else {
				break
			}
		}
	}
	return false
}

// promptReauth asks the user if they want to re-authenticate
func promptReauth() bool {
	fmt.Print("Would you like to re-authenticate? [Y/n]: ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	input = strings.TrimSpace(strings.ToLower(input))
	// Empty input (just Enter) or "y" or "yes" means yes
	return input == "" || input == "y" || input == "yes"
}
