package drive

import (
	"context"
	"fmt"
	"io"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"

	"github.com/open-cli-collective/google-readonly/internal/auth"
)

// Client wraps the Google Drive API service
type Client struct {
	service *drive.Service
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
		service: srv,
	}, nil
}

// fileFields defines the fields to request from the Drive API
const fileFields = "id,name,mimeType,size,createdTime,modifiedTime,parents,owners,webViewLink,shared"

// ListFiles returns files matching the query
func (c *Client) ListFiles(query string, pageSize int64) ([]*File, error) {
	call := c.service.Files.List().
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
	f, err := c.service.Files.Get(fileID).
		Fields(fileFields).
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get file: %w", err)
	}
	return ParseFile(f), nil
}

// DownloadFile downloads a regular (non-Google Workspace) file
func (c *Client) DownloadFile(fileID string) ([]byte, error) {
	resp, err := c.service.Files.Get(fileID).Download()
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read file content: %w", err)
	}
	return data, nil
}

// ExportFile exports a Google Workspace file to the specified MIME type
func (c *Client) ExportFile(fileID string, mimeType string) ([]byte, error) {
	resp, err := c.service.Files.Export(fileID, mimeType).Download()
	if err != nil {
		return nil, fmt.Errorf("failed to export file: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read exported content: %w", err)
	}
	return data, nil
}
