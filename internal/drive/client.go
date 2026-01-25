package drive

import (
	"context"
	"fmt"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"

	"github.com/open-cli-collective/google-readonly/internal/auth"
)

// Client wraps the Google Drive API service
type Client struct {
	Service *drive.Service
}

// NewClient creates a new Drive client with OAuth2 authentication
func NewClient(ctx context.Context) (*Client, error) {
	client, err := auth.GetHTTPClient(ctx)
	if err != nil {
		return nil, err
	}

	srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("unable to create Drive service: %w", err)
	}

	return &Client{
		Service: srv,
	}, nil
}

// fileFields defines the fields to request from the Drive API
const fileFields = "id,name,mimeType,size,createdTime,modifiedTime,parents,owners,webViewLink,shared"

// ListFiles returns files matching the query
func (c *Client) ListFiles(query string, pageSize int64) ([]*File, error) {
	call := c.Service.Files.List().
		Fields("files(" + fileFields + ")").
		OrderBy("modifiedTime desc")

	if query != "" {
		call = call.Q(query)
	}
	if pageSize > 0 {
		call = call.PageSize(pageSize)
	}

	resp, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	files := make([]*File, 0, len(resp.Files))
	for _, f := range resp.Files {
		files = append(files, ParseFile(f))
	}
	return files, nil
}

// GetFile retrieves a single file by ID
func (c *Client) GetFile(fileID string) (*File, error) {
	f, err := c.Service.Files.Get(fileID).
		Fields(fileFields).
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get file: %w", err)
	}
	return ParseFile(f), nil
}
