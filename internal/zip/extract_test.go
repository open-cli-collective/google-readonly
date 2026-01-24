package zip

import (
	"archive/zip"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestZip(t *testing.T, files map[string][]byte) string {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "test-*.zip")
	require.NoError(t, err)
	defer tmpFile.Close()

	w := zip.NewWriter(tmpFile)
	for name, content := range files {
		f, err := w.Create(name)
		require.NoError(t, err)
		_, err = f.Write(content)
		require.NoError(t, err)
	}
	require.NoError(t, w.Close())

	return tmpFile.Name()
}

func TestExtract(t *testing.T) {
	t.Run("extracts simple zip file", func(t *testing.T) {
		zipPath := createTestZip(t, map[string][]byte{
			"file1.txt": []byte("content 1"),
			"file2.txt": []byte("content 2"),
		})
		defer os.Remove(zipPath)

		destDir := t.TempDir()
		err := Extract(zipPath, destDir, DefaultOptions())
		require.NoError(t, err)

		content1, err := os.ReadFile(filepath.Join(destDir, "file1.txt"))
		require.NoError(t, err)
		assert.Equal(t, "content 1", string(content1))

		content2, err := os.ReadFile(filepath.Join(destDir, "file2.txt"))
		require.NoError(t, err)
		assert.Equal(t, "content 2", string(content2))
	})

	t.Run("extracts nested directories", func(t *testing.T) {
		zipPath := createTestZip(t, map[string][]byte{
			"dir1/file1.txt":      []byte("nested 1"),
			"dir1/dir2/file2.txt": []byte("nested 2"),
		})
		defer os.Remove(zipPath)

		destDir := t.TempDir()
		err := Extract(zipPath, destDir, DefaultOptions())
		require.NoError(t, err)

		content, err := os.ReadFile(filepath.Join(destDir, "dir1", "dir2", "file2.txt"))
		require.NoError(t, err)
		assert.Equal(t, "nested 2", string(content))
	})

	t.Run("creates destination directory if not exists", func(t *testing.T) {
		zipPath := createTestZip(t, map[string][]byte{
			"test.txt": []byte("test"),
		})
		defer os.Remove(zipPath)

		destDir := filepath.Join(t.TempDir(), "new", "nested", "dir")
		err := Extract(zipPath, destDir, DefaultOptions())
		require.NoError(t, err)

		_, err = os.Stat(filepath.Join(destDir, "test.txt"))
		assert.NoError(t, err)
	})

	t.Run("rejects invalid zip file", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "invalid-*.zip")
		require.NoError(t, err)
		tmpFile.WriteString("not a zip file")
		tmpFile.Close()
		defer os.Remove(tmpFile.Name())

		destDir := t.TempDir()
		err = Extract(tmpFile.Name(), destDir, DefaultOptions())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to open zip")
	})
}

func TestExtractSecurityPathTraversal(t *testing.T) {
	t.Run("rejects path with leading ..", func(t *testing.T) {
		// Create a malicious zip with path traversal
		tmpFile, err := os.CreateTemp("", "malicious-*.zip")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		w := zip.NewWriter(tmpFile)
		// Manually create a file with path traversal
		header := &zip.FileHeader{
			Name:   "../../../etc/passwd",
			Method: zip.Store,
		}
		f, err := w.CreateHeader(header)
		require.NoError(t, err)
		f.Write([]byte("malicious"))
		w.Close()
		tmpFile.Close()

		destDir := t.TempDir()
		err = Extract(tmpFile.Name(), destDir, DefaultOptions())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid file path")
	})

	t.Run("rejects absolute paths", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "malicious-*.zip")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		w := zip.NewWriter(tmpFile)
		header := &zip.FileHeader{
			Name:   "/etc/passwd",
			Method: zip.Store,
		}
		f, err := w.CreateHeader(header)
		require.NoError(t, err)
		f.Write([]byte("malicious"))
		w.Close()
		tmpFile.Close()

		destDir := t.TempDir()
		err = Extract(tmpFile.Name(), destDir, DefaultOptions())
		assert.Error(t, err)
	})
}

func TestExtractSecurityLimits(t *testing.T) {
	t.Run("rejects zip with too many files", func(t *testing.T) {
		files := make(map[string][]byte)
		for i := 0; i < 10; i++ {
			files[filepath.Join("file", string(rune('a'+i))+".txt")] = []byte("x")
		}
		zipPath := createTestZip(t, files)
		defer os.Remove(zipPath)

		destDir := t.TempDir()
		opts := Options{
			MaxFileSize:  MaxFileSize,
			MaxTotalSize: MaxTotalSize,
			MaxFiles:     5, // Less than files in zip
			MaxDepth:     MaxDepth,
		}
		err := Extract(zipPath, destDir, opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "too many files")
	})

	t.Run("rejects file exceeding max size", func(t *testing.T) {
		zipPath := createTestZip(t, map[string][]byte{
			"large.txt": make([]byte, 1000),
		})
		defer os.Remove(zipPath)

		destDir := t.TempDir()
		opts := Options{
			MaxFileSize:  100, // Less than file size
			MaxTotalSize: MaxTotalSize,
			MaxFiles:     MaxFiles,
			MaxDepth:     MaxDepth,
		}
		err := Extract(zipPath, destDir, opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds max size")
	})

	t.Run("rejects total size exceeding limit", func(t *testing.T) {
		zipPath := createTestZip(t, map[string][]byte{
			"file1.txt": make([]byte, 600),
			"file2.txt": make([]byte, 600),
		})
		defer os.Remove(zipPath)

		destDir := t.TempDir()
		opts := Options{
			MaxFileSize:  MaxFileSize,
			MaxTotalSize: 1000, // Less than total
			MaxFiles:     MaxFiles,
			MaxDepth:     MaxDepth,
		}
		err := Extract(zipPath, destDir, opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds limit")
	})

	t.Run("rejects path too deep", func(t *testing.T) {
		zipPath := createTestZip(t, map[string][]byte{
			"a/b/c/d/e/f/deep.txt": []byte("deep"),
		})
		defer os.Remove(zipPath)

		destDir := t.TempDir()
		opts := Options{
			MaxFileSize:  MaxFileSize,
			MaxTotalSize: MaxTotalSize,
			MaxFiles:     MaxFiles,
			MaxDepth:     3, // Less than actual depth
		}
		err := Extract(zipPath, destDir, opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "too deep")
	})
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()
	assert.Equal(t, int64(MaxFileSize), opts.MaxFileSize)
	assert.Equal(t, int64(MaxTotalSize), opts.MaxTotalSize)
	assert.Equal(t, MaxFiles, opts.MaxFiles)
	assert.Equal(t, MaxDepth, opts.MaxDepth)
}

func TestValidateZip(t *testing.T) {
	t.Run("accepts valid zip within limits", func(t *testing.T) {
		zipPath := createTestZip(t, map[string][]byte{
			"file1.txt": []byte("content"),
		})
		defer os.Remove(zipPath)

		r, err := zip.OpenReader(zipPath)
		require.NoError(t, err)
		defer r.Close()

		err = validateZip(&r.Reader, DefaultOptions())
		assert.NoError(t, err)
	})
}

// mockFS implements FileSystem for testing error paths
type mockFS struct {
	mkdirAllErr error
	openFileErr error
	mkdirCalls  int
	failAfterN  int // fail MkdirAll after N calls (0 = fail immediately)
}

func (m *mockFS) MkdirAll(path string, perm os.FileMode) error {
	m.mkdirCalls++
	if m.failAfterN > 0 && m.mkdirCalls <= m.failAfterN {
		return nil
	}
	return m.mkdirAllErr
}

func (m *mockFS) OpenFile(name string, flag int, perm os.FileMode) (io.WriteCloser, error) {
	if m.openFileErr != nil {
		return nil, m.openFileErr
	}
	return os.OpenFile(name, flag, perm)
}

func (m *mockFS) Remove(name string) error {
	return os.Remove(name)
}

// errorWriter is a writer that fails after writing some bytes
type errorWriter struct {
	written int
	failAt  int
	err     error
}

func (w *errorWriter) Write(p []byte) (int, error) {
	if w.written >= w.failAt {
		return 0, w.err
	}
	w.written += len(p)
	return len(p), nil
}

func (w *errorWriter) Close() error {
	return nil
}

// mockFSWithErrorWriter returns an error writer instead of a real file
type mockFSWithErrorWriter struct {
	writer *errorWriter
}

func (m *mockFSWithErrorWriter) MkdirAll(path string, perm os.FileMode) error {
	return nil
}

func (m *mockFSWithErrorWriter) OpenFile(name string, flag int, perm os.FileMode) (io.WriteCloser, error) {
	return m.writer, nil
}

func (m *mockFSWithErrorWriter) Remove(name string) error {
	return nil
}

func TestExtractFileSystemErrors(t *testing.T) {
	// Save original fs and restore after tests
	originalFS := fs
	defer func() { fs = originalFS }()

	t.Run("returns error when MkdirAll fails for destination", func(t *testing.T) {
		fs = &mockFS{
			mkdirAllErr: errors.New("permission denied"),
		}

		zipPath := createTestZip(t, map[string][]byte{
			"test.txt": []byte("content"),
		})
		defer os.Remove(zipPath)

		err := Extract(zipPath, "/tmp/test-dest", DefaultOptions())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create destination")
	})

	t.Run("returns error when MkdirAll fails for parent directory", func(t *testing.T) {
		fs = &mockFS{
			mkdirAllErr: errors.New("disk full"),
			failAfterN:  1, // succeed for dest dir, fail for parent dir
		}

		zipPath := createTestZip(t, map[string][]byte{
			"subdir/test.txt": []byte("content"),
		})
		defer os.Remove(zipPath)

		destDir := t.TempDir()
		err := Extract(zipPath, destDir, DefaultOptions())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "disk full")
	})

	t.Run("returns error when OpenFile fails", func(t *testing.T) {
		fs = &mockFS{
			openFileErr: errors.New("too many open files"),
		}

		zipPath := createTestZip(t, map[string][]byte{
			"test.txt": []byte("content"),
		})
		defer os.Remove(zipPath)

		destDir := t.TempDir()
		err := Extract(zipPath, destDir, DefaultOptions())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "too many open files")
	})

	t.Run("returns error when io.Copy fails", func(t *testing.T) {
		fs = &mockFSWithErrorWriter{
			writer: &errorWriter{
				failAt: 0,
				err:    errors.New("write error"),
			},
		}

		zipPath := createTestZip(t, map[string][]byte{
			"test.txt": []byte("content"),
		})
		defer os.Remove(zipPath)

		destDir := t.TempDir()
		err := Extract(zipPath, destDir, DefaultOptions())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "write error")
	})
}

func TestExtractDirectoryEntry(t *testing.T) {
	// Save original fs and restore after test
	originalFS := fs
	defer func() { fs = originalFS }()

	t.Run("extracts directory entries", func(t *testing.T) {
		// Create zip with explicit directory entry
		tmpFile, err := os.CreateTemp("", "dir-test-*.zip")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		w := zip.NewWriter(tmpFile)
		// Create a directory entry with proper permissions
		header := &zip.FileHeader{
			Name:   "mydir/",
			Method: zip.Store,
		}
		header.SetMode(os.ModeDir | 0755)
		_, err = w.CreateHeader(header)
		require.NoError(t, err)
		w.Close()
		tmpFile.Close()

		destDir := t.TempDir()
		err = Extract(tmpFile.Name(), destDir, DefaultOptions())
		require.NoError(t, err)

		// Verify directory was created
		info, err := os.Stat(filepath.Join(destDir, "mydir"))
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("returns error when MkdirAll fails for directory entry", func(t *testing.T) {
		fs = &mockFS{
			mkdirAllErr: errors.New("cannot create directory"),
			failAfterN:  1, // succeed for dest dir, fail for directory entry
		}

		// Create zip with explicit directory entry
		tmpFile, err := os.CreateTemp("", "dir-test-*.zip")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		w := zip.NewWriter(tmpFile)
		_, err = w.Create("mydir/")
		require.NoError(t, err)
		w.Close()
		tmpFile.Close()

		destDir := t.TempDir()
		err = Extract(tmpFile.Name(), destDir, DefaultOptions())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot create directory")
	})
}
