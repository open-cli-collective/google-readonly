// Package appidentity declares google-readonly's CLI identity: the config/
// keyring directory name, default credential ref, product name, and the exact
// OAuth scope set (with human descriptions) this CLI requests. It is the single
// place that defines what makes this binary "gro" as opposed to any other CLI
// built on github.com/open-cli-collective/google-cli-common.
//
// main registers this identity via config.Register before the command tree
// runs; the architecture test asserts the scope set stays non-destructive.
package appidentity

import (
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/people/v1"

	"github.com/open-cli-collective/google-cli-common/config"
)

// Scopes is the OAuth scope set gro requests. Gmail uses the modify scope for
// non-destructive organization (label, archive, star, mark read/unread) — a
// superset of readonly. Calendar uses readonly (list metadata) plus events
// (RSVP/color). Contacts uses the full scope for group management and starring.
// Profile is required for `gro me` and init verification. There is deliberately
// NO gmail.settings.* or https://mail.google.com/ here: gro never manages
// filters and can never permanently delete. The architecture test enforces that
// every scope below is on the non-destructive allowlist.
var Scopes = []string{
	gmail.GmailModifyScope,
	calendar.CalendarReadonlyScope,
	calendar.CalendarEventsScope,
	people.ContactsScope,
	people.UserinfoProfileScope,
	drive.DriveReadonlyScope,
	drive.DriveMetadataScope,
}

// ScopeDescriptions maps each requested scope URL to a human-friendly
// description, shown by the init wizard and the scope-drift re-auth prompt.
var ScopeDescriptions = map[string]string{
	gmail.GmailModifyScope:         "Gmail Modify — read messages, plus label, archive, star, and mark read/unread. No send or delete access.",
	gmail.GmailReadonlyScope:       "Gmail Read-Only — read messages and metadata.",
	calendar.CalendarReadonlyScope: "Calendar Read-Only — read calendars and events.",
	calendar.CalendarEventsScope:   "Calendar Events — read and update events (RSVP, color). No calendar settings access.",
	people.ContactsScope:           "Contacts — read contacts and groups, plus manage group membership and starring.",
	people.ContactsReadonlyScope:   "Contacts Read-Only — read contacts and groups.",
	people.UserinfoProfileScope:    "Profile — read the authenticated user's name and email address (required for 'gro me').",
	drive.DriveReadonlyScope:       "Drive Read-Only — read files and metadata.",
	drive.DriveMetadataScope:       "Drive Metadata — read and update file metadata (star/unstar). No file content write access.",
}

// Identity is gro's config.Identity, registered once at startup.
func Identity() config.Identity {
	return config.Identity{
		DirName:           "google-readonly",
		DefaultRef:        "google-readonly/default",
		ProductName:       "gro",
		Scopes:            Scopes,
		ScopeDescriptions: ScopeDescriptions,
		// Symmetric with grw: gro init can also adopt grw's OAuth client JSON
		// if grw was set up first. Tokens remain separate per keyring namespace.
		SiblingDirNames: []string{"google-readwrite"},
	}
}
