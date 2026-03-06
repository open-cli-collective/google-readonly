// Package gmail provides a client for the Gmail API.
package gmail

import (
	"context"
	"fmt"
	"sync"

	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"

	"github.com/open-cli-collective/google-readonly/internal/auth"
)

// Client wraps the Gmail API service
type Client struct {
	service      *gmail.Service
	userID       string
	labels       map[string]*gmail.Label
	labelsByName map[string]string // display name -> label ID
	labelsLoaded bool
	labelsMu     sync.RWMutex
}

// NewClient creates a new Gmail client with OAuth2 authentication
func NewClient(ctx context.Context) (*Client, error) {
	client, err := auth.GetHTTPClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading OAuth client: %w", err)
	}

	srv, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("creating Gmail service: %w", err)
	}

	return &Client{
		service: srv,
		userID:  "me",
	}, nil
}

// FetchLabels retrieves and caches all labels from the Gmail account
func (c *Client) FetchLabels(ctx context.Context) error {
	// Check with read lock first to avoid unnecessary API calls
	c.labelsMu.RLock()
	if c.labelsLoaded {
		c.labelsMu.RUnlock()
		return nil
	}
	c.labelsMu.RUnlock()

	// Acquire write lock and check again (double-check locking)
	c.labelsMu.Lock()
	defer c.labelsMu.Unlock()

	// Re-check after acquiring write lock
	if c.labelsLoaded {
		return nil
	}

	resp, err := c.service.Users.Labels.List(c.userID).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("fetching labels: %w", err)
	}

	c.labels = make(map[string]*gmail.Label)
	c.labelsByName = make(map[string]string)
	for _, label := range resp.Labels {
		c.labels[label.Id] = label
		c.labelsByName[label.Name] = label.Id
	}
	c.labelsLoaded = true

	return nil
}

// GetLabelName resolves a label ID to its display name
func (c *Client) GetLabelName(labelID string) string {
	c.labelsMu.RLock()
	defer c.labelsMu.RUnlock()

	if label, ok := c.labels[labelID]; ok {
		return label.Name
	}
	return labelID
}

// GetLabels returns all cached labels
func (c *Client) GetLabels() []*gmail.Label {
	c.labelsMu.RLock()
	defer c.labelsMu.RUnlock()

	if !c.labelsLoaded {
		return nil
	}
	labels := make([]*gmail.Label, 0, len(c.labels))
	for _, label := range c.labels {
		labels = append(labels, label)
	}
	return labels
}

// GetLabelID resolves a label display name to its ID. Calls FetchLabels if needed.
func (c *Client) GetLabelID(ctx context.Context, name string) (string, error) {
	if err := c.FetchLabels(ctx); err != nil {
		return "", err
	}

	c.labelsMu.RLock()
	defer c.labelsMu.RUnlock()

	if id, ok := c.labelsByName[name]; ok {
		return id, nil
	}
	return "", fmt.Errorf("label %q not found", name)
}

// ModifyMessages modifies labels on one or more messages.
// Uses Messages.Modify for a single message; Messages.BatchModify for multiple.
// Gmail limits batch operations to 1000 IDs, so this method chunks automatically.
func (c *Client) ModifyMessages(ctx context.Context, ids []string, addLabels, removeLabels []string) error {
	if len(ids) == 0 {
		return nil
	}

	if len(ids) == 1 {
		req := &gmail.ModifyMessageRequest{
			AddLabelIds:    addLabels,
			RemoveLabelIds: removeLabels,
		}
		_, err := c.service.Users.Messages.Modify(c.userID, ids[0], req).Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("modifying message %s: %w", ids[0], err)
		}
		return nil
	}

	const batchSize = 1000
	for i := 0; i < len(ids); i += batchSize {
		end := i + batchSize
		if end > len(ids) {
			end = len(ids)
		}
		chunk := ids[i:end]

		req := &gmail.BatchModifyMessagesRequest{
			Ids:            chunk,
			AddLabelIds:    addLabels,
			RemoveLabelIds: removeLabels,
		}
		err := c.service.Users.Messages.BatchModify(c.userID, req).Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("batch modifying messages: %w", err)
		}
	}
	return nil
}

// Profile represents a Gmail user profile.
type Profile struct {
	EmailAddress  string
	MessagesTotal int64
	ThreadsTotal  int64
}

// GetProfile retrieves the authenticated user's profile
func (c *Client) GetProfile(ctx context.Context) (*Profile, error) {
	profile, err := c.service.Users.GetProfile(c.userID).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("getting profile: %w", err)
	}
	return &Profile{
		EmailAddress:  profile.EmailAddress,
		MessagesTotal: profile.MessagesTotal,
		ThreadsTotal:  profile.ThreadsTotal,
	}, nil
}
