package drive

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-cli-collective/google-readonly/internal/drive"
)

func TestTreeCommand(t *testing.T) {
	cmd := newTreeCommand()

	t.Run("has correct use", func(t *testing.T) {
		assert.Equal(t, "tree [folder-id]", cmd.Use)
	})

	t.Run("accepts zero or one argument", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		assert.NoError(t, err)

		err = cmd.Args(cmd, []string{"folder-id"})
		assert.NoError(t, err)

		err = cmd.Args(cmd, []string{"folder-id", "extra"})
		assert.Error(t, err)
	})

	t.Run("has depth flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("depth")
		assert.NotNil(t, flag)
		assert.Equal(t, "d", flag.Shorthand)
		assert.Equal(t, "2", flag.DefValue)
	})

	t.Run("has files flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("files")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("has json flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
		assert.Equal(t, "j", flag.Shorthand)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("has short description", func(t *testing.T) {
		assert.Contains(t, cmd.Short, "folder structure")
	})
}

func TestPrintTree(t *testing.T) {
	// Capture stdout for testing
	captureOutput := func(fn func()) string {
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		fn()

		w.Close()
		os.Stdout = old

		var buf bytes.Buffer
		io.Copy(&buf, r)
		return buf.String()
	}

	t.Run("prints single node", func(t *testing.T) {
		node := &TreeNode{
			ID:   "root",
			Name: "My Drive",
			Type: "Folder",
		}

		output := captureOutput(func() {
			printTree(node, "", true)
		})

		assert.Equal(t, "My Drive\n", output)
	})

	t.Run("prints tree with children", func(t *testing.T) {
		node := &TreeNode{
			ID:   "root",
			Name: "My Drive",
			Type: "Folder",
			Children: []*TreeNode{
				{ID: "1", Name: "Documents", Type: "Folder"},
				{ID: "2", Name: "Photos", Type: "Folder"},
			},
		}

		output := captureOutput(func() {
			printTree(node, "", true)
		})

		assert.Contains(t, output, "My Drive")
		assert.Contains(t, output, "├── Documents")
		assert.Contains(t, output, "└── Photos")
	})

	t.Run("prints nested tree", func(t *testing.T) {
		node := &TreeNode{
			ID:   "root",
			Name: "My Drive",
			Type: "Folder",
			Children: []*TreeNode{
				{
					ID:   "1",
					Name: "Projects",
					Type: "Folder",
					Children: []*TreeNode{
						{ID: "1a", Name: "Project A", Type: "Folder"},
						{ID: "1b", Name: "Project B", Type: "Folder"},
					},
				},
				{ID: "2", Name: "Documents", Type: "Folder"},
			},
		}

		output := captureOutput(func() {
			printTree(node, "", true)
		})

		assert.Contains(t, output, "My Drive")
		assert.Contains(t, output, "├── Projects")
		assert.Contains(t, output, "│   ├── Project A")
		assert.Contains(t, output, "│   └── Project B")
		assert.Contains(t, output, "└── Documents")
	})

	t.Run("prints deeply nested tree", func(t *testing.T) {
		node := &TreeNode{
			ID:   "root",
			Name: "Root",
			Type: "Folder",
			Children: []*TreeNode{
				{
					ID:   "1",
					Name: "Level1",
					Type: "Folder",
					Children: []*TreeNode{
						{
							ID:   "2",
							Name: "Level2",
							Type: "Folder",
							Children: []*TreeNode{
								{ID: "3", Name: "Level3", Type: "Folder"},
							},
						},
					},
				},
			},
		}

		output := captureOutput(func() {
			printTree(node, "", true)
		})

		assert.Contains(t, output, "Root")
		assert.Contains(t, output, "└── Level1")
		assert.Contains(t, output, "    └── Level2")
		assert.Contains(t, output, "        └── Level3")
	})

	t.Run("handles empty children", func(t *testing.T) {
		node := &TreeNode{
			ID:       "root",
			Name:     "Empty Folder",
			Type:     "Folder",
			Children: []*TreeNode{},
		}

		output := captureOutput(func() {
			printTree(node, "", true)
		})

		assert.Equal(t, "Empty Folder\n", output)
	})
}

func TestTreeNode(t *testing.T) {
	t.Run("serializes to JSON correctly", func(t *testing.T) {
		node := &TreeNode{
			ID:   "abc123",
			Name: "Test",
			Type: "Folder",
			Children: []*TreeNode{
				{ID: "child1", Name: "Child", Type: "Document"},
			},
		}

		assert.Equal(t, "abc123", node.ID)
		assert.Equal(t, "Test", node.Name)
		assert.Equal(t, "Folder", node.Type)
		assert.Len(t, node.Children, 1)
	})

	t.Run("handles nil children", func(t *testing.T) {
		node := &TreeNode{
			ID:       "abc123",
			Name:     "Test",
			Type:     "Folder",
			Children: nil,
		}

		assert.Nil(t, node.Children)
	})
}

// mockDriveClient implements drive.DriveClientInterface for testing
type mockDriveClient struct {
	files    map[string]*drive.File   // fileID -> File
	children map[string][]*drive.File // folderID -> children
}

func newMockDriveClient() *mockDriveClient {
	return &mockDriveClient{
		files:    make(map[string]*drive.File),
		children: make(map[string][]*drive.File),
	}
}

func (m *mockDriveClient) GetFile(fileID string) (*drive.File, error) {
	if f, ok := m.files[fileID]; ok {
		return f, nil
	}
	return nil, fmt.Errorf("file not found: %s", fileID)
}

func (m *mockDriveClient) ListFiles(query string, _ int64) ([]*drive.File, error) {
	// Extract folderID from query like "'folder123' in parents and trashed = false"
	// The query format is: "'<folderID>' in parents and trashed = false"
	for folderID, files := range m.children {
		searchPattern := fmt.Sprintf("'%s' in parents", folderID)
		if strings.Contains(query, searchPattern) {
			return files, nil
		}
	}
	return []*drive.File{}, nil
}

func (m *mockDriveClient) ListFilesWithScope(query string, pageSize int64, _ drive.DriveScope) ([]*drive.File, error) {
	// Delegate to ListFiles for testing purposes
	return m.ListFiles(query, pageSize)
}

func (m *mockDriveClient) DownloadFile(_ string) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockDriveClient) ExportFile(_ string, _ string) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockDriveClient) ListSharedDrives(_ int64) ([]*drive.SharedDrive, error) {
	return nil, fmt.Errorf("not implemented")
}

func TestBuildTree(t *testing.T) {
	t.Run("builds tree for root folder", func(t *testing.T) {
		mock := newMockDriveClient()
		// Set up child folders in both children and files maps
		mock.children["root"] = []*drive.File{
			{ID: "folder1", Name: "Documents", MimeType: drive.MimeTypeFolder},
			{ID: "folder2", Name: "Photos", MimeType: drive.MimeTypeFolder},
		}
		// GetFile is called for each child folder during recursion
		mock.files["folder1"] = &drive.File{ID: "folder1", Name: "Documents", MimeType: drive.MimeTypeFolder}
		mock.files["folder2"] = &drive.File{ID: "folder2", Name: "Photos", MimeType: drive.MimeTypeFolder}

		tree, err := buildTree(mock, "root", 1, false)

		assert.NoError(t, err)
		assert.Equal(t, "root", tree.ID)
		assert.Equal(t, "My Drive", tree.Name)
		assert.Equal(t, "Folder", tree.Type)
		assert.Len(t, tree.Children, 2)
	})

	t.Run("builds tree for specific folder", func(t *testing.T) {
		mock := newMockDriveClient()
		mock.files["folder123"] = &drive.File{
			ID:       "folder123",
			Name:     "My Folder",
			MimeType: drive.MimeTypeFolder,
		}
		mock.children["folder123"] = []*drive.File{
			{ID: "doc1", Name: "Notes.txt", MimeType: "text/plain"},
		}

		tree, err := buildTree(mock, "folder123", 1, true)

		assert.NoError(t, err)
		assert.Equal(t, "folder123", tree.ID)
		assert.Equal(t, "My Folder", tree.Name)
		assert.Len(t, tree.Children, 1)
		assert.Equal(t, "Notes.txt", tree.Children[0].Name)
	})

	t.Run("respects depth limit", func(t *testing.T) {
		mock := newMockDriveClient()
		mock.children["root"] = []*drive.File{
			{ID: "folder1", Name: "Level1", MimeType: drive.MimeTypeFolder},
		}
		mock.files["folder1"] = &drive.File{ID: "folder1", Name: "Level1", MimeType: drive.MimeTypeFolder}
		mock.children["folder1"] = []*drive.File{
			{ID: "folder2", Name: "Level2", MimeType: drive.MimeTypeFolder},
		}
		mock.files["folder2"] = &drive.File{ID: "folder2", Name: "Level2", MimeType: drive.MimeTypeFolder}

		// With depth 1, should not recurse into Level1
		tree, err := buildTree(mock, "root", 1, false)

		assert.NoError(t, err)
		assert.Len(t, tree.Children, 1)
		assert.Equal(t, "Level1", tree.Children[0].Name)
		// Children of Level1 should be empty due to depth limit
		assert.Empty(t, tree.Children[0].Children)
	})

	t.Run("returns node with no children at depth 0", func(t *testing.T) {
		mock := newMockDriveClient()
		mock.children["root"] = []*drive.File{
			{ID: "folder1", Name: "Folder", MimeType: drive.MimeTypeFolder},
		}

		tree, err := buildTree(mock, "root", 0, false)

		assert.NoError(t, err)
		assert.Equal(t, "My Drive", tree.Name)
		assert.Nil(t, tree.Children)
	})

	t.Run("includes files when includeFiles is true", func(t *testing.T) {
		mock := newMockDriveClient()
		mock.children["root"] = []*drive.File{
			{ID: "folder1", Name: "Docs", MimeType: drive.MimeTypeFolder},
			{ID: "file1", Name: "readme.txt", MimeType: "text/plain"},
		}
		mock.files["folder1"] = &drive.File{ID: "folder1", Name: "Docs", MimeType: drive.MimeTypeFolder}

		tree, err := buildTree(mock, "root", 1, true)

		assert.NoError(t, err)
		assert.Len(t, tree.Children, 2)
	})

	t.Run("sorts folders before files", func(t *testing.T) {
		mock := newMockDriveClient()
		mock.children["root"] = []*drive.File{
			{ID: "file1", Name: "aaa.txt", MimeType: "text/plain"},
			{ID: "folder1", Name: "zzz-folder", MimeType: drive.MimeTypeFolder},
		}
		mock.files["folder1"] = &drive.File{ID: "folder1", Name: "zzz-folder", MimeType: drive.MimeTypeFolder}

		tree, err := buildTree(mock, "root", 1, true)

		assert.NoError(t, err)
		assert.Len(t, tree.Children, 2)
		// Folder should come first despite alphabetical order
		assert.Equal(t, "zzz-folder", tree.Children[0].Name)
		assert.Equal(t, "aaa.txt", tree.Children[1].Name)
	})
}
