package drive

import (
	driveapi "github.com/open-cli-collective/google-readonly/internal/drive"
)

// MockDriveClient is a configurable mock for DriveClient.
type MockDriveClient struct {
	ListFilesFunc          func(query string, pageSize int64) ([]*driveapi.File, error)
	ListFilesWithScopeFunc func(query string, pageSize int64, scope driveapi.DriveScope) ([]*driveapi.File, error)
	GetFileFunc            func(fileID string) (*driveapi.File, error)
	DownloadFileFunc       func(fileID string) ([]byte, error)
	ExportFileFunc         func(fileID, mimeType string) ([]byte, error)
	ListSharedDrivesFunc   func(pageSize int64) ([]*driveapi.SharedDrive, error)
}

// Verify MockDriveClient implements DriveClient
var _ DriveClient = (*MockDriveClient)(nil)

func (m *MockDriveClient) ListFiles(query string, pageSize int64) ([]*driveapi.File, error) {
	if m.ListFilesFunc != nil {
		return m.ListFilesFunc(query, pageSize)
	}
	return nil, nil
}

func (m *MockDriveClient) ListFilesWithScope(query string, pageSize int64, scope driveapi.DriveScope) ([]*driveapi.File, error) {
	if m.ListFilesWithScopeFunc != nil {
		return m.ListFilesWithScopeFunc(query, pageSize, scope)
	}
	// Fall back to ListFiles if no scope function defined
	if m.ListFilesFunc != nil {
		return m.ListFilesFunc(query, pageSize)
	}
	return nil, nil
}

func (m *MockDriveClient) GetFile(fileID string) (*driveapi.File, error) {
	if m.GetFileFunc != nil {
		return m.GetFileFunc(fileID)
	}
	return nil, nil
}

func (m *MockDriveClient) DownloadFile(fileID string) ([]byte, error) {
	if m.DownloadFileFunc != nil {
		return m.DownloadFileFunc(fileID)
	}
	return nil, nil
}

func (m *MockDriveClient) ExportFile(fileID, mimeType string) ([]byte, error) {
	if m.ExportFileFunc != nil {
		return m.ExportFileFunc(fileID, mimeType)
	}
	return nil, nil
}

func (m *MockDriveClient) ListSharedDrives(pageSize int64) ([]*driveapi.SharedDrive, error) {
	if m.ListSharedDrivesFunc != nil {
		return m.ListSharedDrivesFunc(pageSize)
	}
	return nil, nil
}
