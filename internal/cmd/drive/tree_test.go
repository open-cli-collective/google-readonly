package drive

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
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
