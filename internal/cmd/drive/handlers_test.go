package drive

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	driveapi "github.com/open-cli-collective/google-readonly/internal/drive"
	"github.com/open-cli-collective/google-readonly/internal/testutil"
)

// captureOutput captures stdout during test execution
func captureOutput(t *testing.T, f func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

// withMockClient sets up a mock client factory for tests
func withMockClient(mock driveapi.DriveClientInterface, f func()) {
	originalFactory := ClientFactory
	ClientFactory = func() (driveapi.DriveClientInterface, error) {
		return mock, nil
	}
	defer func() { ClientFactory = originalFactory }()
	f()
}

// withFailingClientFactory sets up a factory that returns an error
func withFailingClientFactory(f func()) {
	originalFactory := ClientFactory
	ClientFactory = func() (driveapi.DriveClientInterface, error) {
		return nil, errors.New("connection failed")
	}
	defer func() { ClientFactory = originalFactory }()
	f()
}

func TestListCommand_Success(t *testing.T) {
	mock := &testutil.MockDriveClient{
		ListFilesFunc: func(query string, _ int64) ([]*driveapi.File, error) {
			assert.Contains(t, query, "'root' in parents")
			return testutil.SampleDriveFiles(2), nil
		},
	}

	cmd := newListCommand()

	withMockClient(mock, func() {
		output := captureOutput(t, func() {
			err := cmd.Execute()
			assert.NoError(t, err)
		})

		assert.Contains(t, output, "file_a")
		assert.Contains(t, output, "test-document.pdf")
	})
}

func TestListCommand_JSONOutput(t *testing.T) {
	mock := &testutil.MockDriveClient{
		ListFilesFunc: func(_ string, _ int64) ([]*driveapi.File, error) {
			return testutil.SampleDriveFiles(1), nil
		},
	}

	cmd := newListCommand()
	cmd.SetArgs([]string{"--json"})

	withMockClient(mock, func() {
		output := captureOutput(t, func() {
			err := cmd.Execute()
			assert.NoError(t, err)
		})

		var files []*driveapi.File
		err := json.Unmarshal([]byte(output), &files)
		assert.NoError(t, err)
		assert.Len(t, files, 1)
	})
}

func TestListCommand_Empty(t *testing.T) {
	mock := &testutil.MockDriveClient{
		ListFilesFunc: func(_ string, _ int64) ([]*driveapi.File, error) {
			return []*driveapi.File{}, nil
		},
	}

	cmd := newListCommand()

	withMockClient(mock, func() {
		output := captureOutput(t, func() {
			err := cmd.Execute()
			assert.NoError(t, err)
		})

		assert.Contains(t, output, "No files found")
	})
}

func TestListCommand_WithFolder(t *testing.T) {
	mock := &testutil.MockDriveClient{
		ListFilesFunc: func(query string, _ int64) ([]*driveapi.File, error) {
			assert.Contains(t, query, "'folder123' in parents")
			return testutil.SampleDriveFiles(1), nil
		},
	}

	cmd := newListCommand()
	cmd.SetArgs([]string{"folder123"})

	withMockClient(mock, func() {
		output := captureOutput(t, func() {
			err := cmd.Execute()
			assert.NoError(t, err)
		})

		assert.Contains(t, output, "file_a")
	})
}

func TestListCommand_WithTypeFilter(t *testing.T) {
	mock := &testutil.MockDriveClient{
		ListFilesFunc: func(query string, _ int64) ([]*driveapi.File, error) {
			assert.Contains(t, query, "mimeType")
			return testutil.SampleDriveFiles(1), nil
		},
	}

	cmd := newListCommand()
	cmd.SetArgs([]string{"--type", "document"})

	withMockClient(mock, func() {
		output := captureOutput(t, func() {
			err := cmd.Execute()
			assert.NoError(t, err)
		})

		assert.Contains(t, output, "file_a")
	})
}

func TestListCommand_InvalidType(t *testing.T) {
	cmd := newListCommand()
	cmd.SetArgs([]string{"--type", "invalid"})

	withMockClient(&testutil.MockDriveClient{}, func() {
		err := cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown file type")
	})
}

func TestListCommand_APIError(t *testing.T) {
	mock := &testutil.MockDriveClient{
		ListFilesFunc: func(_ string, _ int64) ([]*driveapi.File, error) {
			return nil, errors.New("API error")
		},
	}

	cmd := newListCommand()

	withMockClient(mock, func() {
		err := cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to list files")
	})
}

func TestListCommand_ClientCreationError(t *testing.T) {
	cmd := newListCommand()

	withFailingClientFactory(func() {
		err := cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create Drive client")
	})
}

func TestSearchCommand_Success(t *testing.T) {
	mock := &testutil.MockDriveClient{
		ListFilesFunc: func(query string, _ int64) ([]*driveapi.File, error) {
			assert.Contains(t, query, "fullText contains 'report'")
			return testutil.SampleDriveFiles(2), nil
		},
	}

	cmd := newSearchCommand()
	cmd.SetArgs([]string{"report"})

	withMockClient(mock, func() {
		output := captureOutput(t, func() {
			err := cmd.Execute()
			assert.NoError(t, err)
		})

		assert.Contains(t, output, "file_a")
		assert.Contains(t, output, "2 file(s)")
	})
}

func TestSearchCommand_NameOnly(t *testing.T) {
	mock := &testutil.MockDriveClient{
		ListFilesFunc: func(query string, _ int64) ([]*driveapi.File, error) {
			assert.Contains(t, query, "name contains 'budget'")
			return testutil.SampleDriveFiles(1), nil
		},
	}

	cmd := newSearchCommand()
	cmd.SetArgs([]string{"budget", "--name"})

	withMockClient(mock, func() {
		output := captureOutput(t, func() {
			err := cmd.Execute()
			assert.NoError(t, err)
		})

		assert.Contains(t, output, "file_a")
	})
}

func TestSearchCommand_JSONOutput(t *testing.T) {
	mock := &testutil.MockDriveClient{
		ListFilesFunc: func(_ string, _ int64) ([]*driveapi.File, error) {
			return testutil.SampleDriveFiles(1), nil
		},
	}

	cmd := newSearchCommand()
	cmd.SetArgs([]string{"test", "--json"})

	withMockClient(mock, func() {
		output := captureOutput(t, func() {
			err := cmd.Execute()
			assert.NoError(t, err)
		})

		var files []*driveapi.File
		err := json.Unmarshal([]byte(output), &files)
		assert.NoError(t, err)
		assert.Len(t, files, 1)
	})
}

func TestSearchCommand_NoResults(t *testing.T) {
	mock := &testutil.MockDriveClient{
		ListFilesFunc: func(_ string, _ int64) ([]*driveapi.File, error) {
			return []*driveapi.File{}, nil
		},
	}

	cmd := newSearchCommand()
	cmd.SetArgs([]string{"nonexistent"})

	withMockClient(mock, func() {
		output := captureOutput(t, func() {
			err := cmd.Execute()
			assert.NoError(t, err)
		})

		assert.Contains(t, output, "No files found")
	})
}

func TestSearchCommand_APIError(t *testing.T) {
	mock := &testutil.MockDriveClient{
		ListFilesFunc: func(_ string, _ int64) ([]*driveapi.File, error) {
			return nil, errors.New("API error")
		},
	}

	cmd := newSearchCommand()
	cmd.SetArgs([]string{"test"})

	withMockClient(mock, func() {
		err := cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to search files")
	})
}

func TestGetCommand_Success(t *testing.T) {
	mock := &testutil.MockDriveClient{
		GetFileFunc: func(fileID string) (*driveapi.File, error) {
			assert.Equal(t, "file123", fileID)
			return testutil.SampleDriveFile("file123"), nil
		},
	}

	cmd := newGetCommand()
	cmd.SetArgs([]string{"file123"})

	withMockClient(mock, func() {
		output := captureOutput(t, func() {
			err := cmd.Execute()
			assert.NoError(t, err)
		})

		assert.Contains(t, output, "file123")
		assert.Contains(t, output, "test-document.pdf")
		assert.Contains(t, output, "owner@example.com")
	})
}

func TestGetCommand_JSONOutput(t *testing.T) {
	mock := &testutil.MockDriveClient{
		GetFileFunc: func(_ string) (*driveapi.File, error) {
			return testutil.SampleDriveFile("file123"), nil
		},
	}

	cmd := newGetCommand()
	cmd.SetArgs([]string{"file123", "--json"})

	withMockClient(mock, func() {
		output := captureOutput(t, func() {
			err := cmd.Execute()
			assert.NoError(t, err)
		})

		var file driveapi.File
		err := json.Unmarshal([]byte(output), &file)
		assert.NoError(t, err)
		assert.Equal(t, "file123", file.ID)
	})
}

func TestGetCommand_NotFound(t *testing.T) {
	mock := &testutil.MockDriveClient{
		GetFileFunc: func(_ string) (*driveapi.File, error) {
			return nil, errors.New("file not found")
		},
	}

	cmd := newGetCommand()
	cmd.SetArgs([]string{"nonexistent"})

	withMockClient(mock, func() {
		err := cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get file")
	})
}

func TestDownloadCommand_RegularFile(t *testing.T) {
	// Create a temp directory for download
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	mock := &testutil.MockDriveClient{
		GetFileFunc: func(_ string) (*driveapi.File, error) {
			return testutil.SampleDriveFile("file123"), nil
		},
		DownloadFileFunc: func(fileID string) ([]byte, error) {
			assert.Equal(t, "file123", fileID)
			return []byte("test content"), nil
		},
	}

	cmd := newDownloadCommand()
	cmd.SetArgs([]string{"file123"})

	withMockClient(mock, func() {
		output := captureOutput(t, func() {
			err := cmd.Execute()
			assert.NoError(t, err)
		})

		assert.Contains(t, output, "Downloading")
		assert.Contains(t, output, "Saved to")
	})
}

func TestDownloadCommand_ToStdout(t *testing.T) {
	mock := &testutil.MockDriveClient{
		GetFileFunc: func(_ string) (*driveapi.File, error) {
			return testutil.SampleDriveFile("file123"), nil
		},
		DownloadFileFunc: func(_ string) ([]byte, error) {
			return []byte("test content"), nil
		},
	}

	cmd := newDownloadCommand()
	cmd.SetArgs([]string{"file123", "--stdout"})

	withMockClient(mock, func() {
		output := captureOutput(t, func() {
			err := cmd.Execute()
			assert.NoError(t, err)
		})

		assert.Equal(t, "test content", output)
	})
}

func TestDownloadCommand_GoogleDocRequiresFormat(t *testing.T) {
	mock := &testutil.MockDriveClient{
		GetFileFunc: func(_ string) (*driveapi.File, error) {
			return testutil.SampleGoogleDoc("doc123"), nil
		},
	}

	cmd := newDownloadCommand()
	cmd.SetArgs([]string{"doc123"})

	withMockClient(mock, func() {
		err := cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "requires --format flag")
	})
}

func TestDownloadCommand_ExportGoogleDoc(t *testing.T) {
	// Create a temp directory for download
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	mock := &testutil.MockDriveClient{
		GetFileFunc: func(_ string) (*driveapi.File, error) {
			return testutil.SampleGoogleDoc("doc123"), nil
		},
		ExportFileFunc: func(fileID, mimeType string) ([]byte, error) {
			assert.Equal(t, "doc123", fileID)
			assert.Contains(t, mimeType, "pdf")
			return []byte("pdf content"), nil
		},
	}

	cmd := newDownloadCommand()
	cmd.SetArgs([]string{"doc123", "--format", "pdf"})

	withMockClient(mock, func() {
		output := captureOutput(t, func() {
			err := cmd.Execute()
			assert.NoError(t, err)
		})

		assert.Contains(t, output, "Exporting")
		assert.Contains(t, output, "Saved to")
	})
}

func TestDownloadCommand_RegularFileCannotUseFormat(t *testing.T) {
	mock := &testutil.MockDriveClient{
		GetFileFunc: func(_ string) (*driveapi.File, error) {
			return testutil.SampleDriveFile("file123"), nil
		},
	}

	cmd := newDownloadCommand()
	cmd.SetArgs([]string{"file123", "--format", "pdf"})

	withMockClient(mock, func() {
		err := cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "--format flag is only for Google Workspace files")
	})
}

func TestDownloadCommand_APIError(t *testing.T) {
	mock := &testutil.MockDriveClient{
		GetFileFunc: func(_ string) (*driveapi.File, error) {
			return testutil.SampleDriveFile("file123"), nil
		},
		DownloadFileFunc: func(_ string) ([]byte, error) {
			return nil, errors.New("download failed")
		},
	}

	cmd := newDownloadCommand()
	cmd.SetArgs([]string{"file123", "--stdout"})

	withMockClient(mock, func() {
		err := cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to download file")
	})
}

func TestDownloadCommand_ClientCreationError(t *testing.T) {
	cmd := newDownloadCommand()
	cmd.SetArgs([]string{"file123"})

	withFailingClientFactory(func() {
		err := cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create Drive client")
	})
}
