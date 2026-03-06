package drive

import (
	"context"

	driveapi "github.com/open-cli-collective/google-readonly/internal/drive"
)

// MockDriveClient is a configurable mock for DriveClient.
type MockDriveClient struct {
	ListFilesFunc          func(ctx context.Context, query string, pageSize int64) ([]*driveapi.File, error)
	ListFilesWithScopeFunc func(ctx context.Context, query string, pageSize int64, scope driveapi.DriveScope) ([]*driveapi.File, error)
	GetFileFunc            func(ctx context.Context, fileID string) (*driveapi.File, error)
	DownloadFileFunc       func(ctx context.Context, fileID string) ([]byte, error)
	ExportFileFunc         func(ctx context.Context, fileID, mimeType string) ([]byte, error)
	ListSharedDrivesFunc   func(ctx context.Context, pageSize int64) ([]*driveapi.SharedDrive, error)
	StarFileFunc           func(ctx context.Context, fileID string) error
	UnstarFileFunc         func(ctx context.Context, fileID string) error
	SearchFileIDsFunc      func(ctx context.Context, query string, pageSize int64) ([]string, error)
}

// Verify MockDriveClient implements DriveClient
var _ DriveClient = (*MockDriveClient)(nil)

func (m *MockDriveClient) ListFiles(ctx context.Context, query string, pageSize int64) ([]*driveapi.File, error) {
	if m.ListFilesFunc != nil {
		return m.ListFilesFunc(ctx, query, pageSize)
	}
	return nil, nil
}

func (m *MockDriveClient) ListFilesWithScope(ctx context.Context, query string, pageSize int64, scope driveapi.DriveScope) ([]*driveapi.File, error) {
	if m.ListFilesWithScopeFunc != nil {
		return m.ListFilesWithScopeFunc(ctx, query, pageSize, scope)
	}
	// Fall back to ListFiles if no scope function defined
	if m.ListFilesFunc != nil {
		return m.ListFilesFunc(ctx, query, pageSize)
	}
	return nil, nil
}

func (m *MockDriveClient) GetFile(ctx context.Context, fileID string) (*driveapi.File, error) {
	if m.GetFileFunc != nil {
		return m.GetFileFunc(ctx, fileID)
	}
	return nil, nil
}

func (m *MockDriveClient) DownloadFile(ctx context.Context, fileID string) ([]byte, error) {
	if m.DownloadFileFunc != nil {
		return m.DownloadFileFunc(ctx, fileID)
	}
	return nil, nil
}

func (m *MockDriveClient) ExportFile(ctx context.Context, fileID, mimeType string) ([]byte, error) {
	if m.ExportFileFunc != nil {
		return m.ExportFileFunc(ctx, fileID, mimeType)
	}
	return nil, nil
}

func (m *MockDriveClient) ListSharedDrives(ctx context.Context, pageSize int64) ([]*driveapi.SharedDrive, error) {
	if m.ListSharedDrivesFunc != nil {
		return m.ListSharedDrivesFunc(ctx, pageSize)
	}
	return nil, nil
}

func (m *MockDriveClient) StarFile(ctx context.Context, fileID string) error {
	if m.StarFileFunc != nil {
		return m.StarFileFunc(ctx, fileID)
	}
	return nil
}

func (m *MockDriveClient) UnstarFile(ctx context.Context, fileID string) error {
	if m.UnstarFileFunc != nil {
		return m.UnstarFileFunc(ctx, fileID)
	}
	return nil
}

func (m *MockDriveClient) SearchFileIDs(ctx context.Context, query string, pageSize int64) ([]string, error) {
	if m.SearchFileIDsFunc != nil {
		return m.SearchFileIDsFunc(ctx, query, pageSize)
	}
	return nil, nil
}
