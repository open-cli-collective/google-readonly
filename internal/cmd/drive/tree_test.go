package drive

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/open-cli-collective/google-readonly/internal/drive"
	"github.com/open-cli-collective/google-readonly/internal/testutil"
)

func TestTreeCommand(t *testing.T) {
	cmd := newTreeCommand()

	t.Run("has correct use", func(t *testing.T) {
		testutil.Equal(t, cmd.Use, "tree [folder-id]")
	})

	t.Run("accepts zero or one argument", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		testutil.NoError(t, err)

		err = cmd.Args(cmd, []string{"folder-id"})
		testutil.NoError(t, err)

		err = cmd.Args(cmd, []string{"folder-id", "extra"})
		testutil.Error(t, err)
	})

	t.Run("has depth flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("depth")
		testutil.NotNil(t, flag)
		testutil.Equal(t, flag.Shorthand, "d")
		testutil.Equal(t, flag.DefValue, "2")
	})

	t.Run("has files flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("files")
		testutil.NotNil(t, flag)
		testutil.Equal(t, flag.DefValue, "false")
	})

	t.Run("has json flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		testutil.NotNil(t, flag)
		testutil.Equal(t, flag.Shorthand, "j")
		testutil.Equal(t, flag.DefValue, "false")
	})

	t.Run("has short description", func(t *testing.T) {
		testutil.Contains(t, cmd.Short, "folder structure")
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

		testutil.Equal(t, output, "My Drive\n")
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

		testutil.Contains(t, output, "My Drive")
		testutil.Contains(t, output, "├── Documents")
		testutil.Contains(t, output, "└── Photos")
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

		testutil.Contains(t, output, "My Drive")
		testutil.Contains(t, output, "├── Projects")
		testutil.Contains(t, output, "│   ├── Project A")
		testutil.Contains(t, output, "│   └── Project B")
		testutil.Contains(t, output, "└── Documents")
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

		testutil.Contains(t, output, "Root")
		testutil.Contains(t, output, "└── Level1")
		testutil.Contains(t, output, "    └── Level2")
		testutil.Contains(t, output, "        └── Level3")
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

		testutil.Equal(t, output, "Empty Folder\n")
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

		testutil.Equal(t, node.ID, "abc123")
		testutil.Equal(t, node.Name, "Test")
		testutil.Equal(t, node.Type, "Folder")
		testutil.Len(t, node.Children, 1)
	})

	t.Run("handles nil children", func(t *testing.T) {
		node := &TreeNode{
			ID:       "abc123",
			Name:     "Test",
			Type:     "Folder",
			Children: nil,
		}

		testutil.Nil(t, node.Children)
	})
}

// mockDriveClient implements DriveClient for testing
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

		testutil.NoError(t, err)
		testutil.Equal(t, tree.ID, "root")
		testutil.Equal(t, tree.Name, "My Drive")
		testutil.Equal(t, tree.Type, "Folder")
		testutil.Len(t, tree.Children, 2)
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

		testutil.NoError(t, err)
		testutil.Equal(t, tree.ID, "folder123")
		testutil.Equal(t, tree.Name, "My Folder")
		testutil.Len(t, tree.Children, 1)
		testutil.Equal(t, tree.Children[0].Name, "Notes.txt")
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

		testutil.NoError(t, err)
		testutil.Len(t, tree.Children, 1)
		testutil.Equal(t, tree.Children[0].Name, "Level1")
		// Children of Level1 should be empty due to depth limit
		testutil.Len(t, tree.Children[0].Children, 0)
	})

	t.Run("returns node with no children at depth 0", func(t *testing.T) {
		mock := newMockDriveClient()
		mock.children["root"] = []*drive.File{
			{ID: "folder1", Name: "Folder", MimeType: drive.MimeTypeFolder},
		}

		tree, err := buildTree(mock, "root", 0, false)

		testutil.NoError(t, err)
		testutil.Equal(t, tree.Name, "My Drive")
		testutil.Nil(t, tree.Children)
	})

	t.Run("includes files when includeFiles is true", func(t *testing.T) {
		mock := newMockDriveClient()
		mock.children["root"] = []*drive.File{
			{ID: "folder1", Name: "Docs", MimeType: drive.MimeTypeFolder},
			{ID: "file1", Name: "readme.txt", MimeType: "text/plain"},
		}
		mock.files["folder1"] = &drive.File{ID: "folder1", Name: "Docs", MimeType: drive.MimeTypeFolder}

		tree, err := buildTree(mock, "root", 1, true)

		testutil.NoError(t, err)
		testutil.Len(t, tree.Children, 2)
	})

	t.Run("sorts folders before files", func(t *testing.T) {
		mock := newMockDriveClient()
		mock.children["root"] = []*drive.File{
			{ID: "file1", Name: "aaa.txt", MimeType: "text/plain"},
			{ID: "folder1", Name: "zzz-folder", MimeType: drive.MimeTypeFolder},
		}
		mock.files["folder1"] = &drive.File{ID: "folder1", Name: "zzz-folder", MimeType: drive.MimeTypeFolder}

		tree, err := buildTree(mock, "root", 1, true)

		testutil.NoError(t, err)
		testutil.Len(t, tree.Children, 2)
		// Folder should come first despite alphabetical order
		testutil.Equal(t, tree.Children[0].Name, "zzz-folder")
		testutil.Equal(t, tree.Children[1].Name, "aaa.txt")
	})
}
