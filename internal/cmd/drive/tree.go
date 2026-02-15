package drive

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/google-readonly/internal/drive"
)

// TreeNode represents a node in the folder tree
type TreeNode struct {
	ID       string      `json:"id"`
	Name     string      `json:"name"`
	Type     string      `json:"type"`
	Children []*TreeNode `json:"children,omitempty"`
}

func newTreeCommand() *cobra.Command {
	var (
		depth      int
		files      bool
		jsonOutput bool
		myDrive    bool
		driveFlag  string
	)

	cmd := &cobra.Command{
		Use:   "tree [folder-id]",
		Short: "Display folder structure",
		Long: `Display the folder structure of Google Drive in a tree format.

By default, shows My Drive structure. Use --drive to show a shared drive's
folder structure.

Examples:
  gro drive tree                        # Show folder tree from My Drive root
  gro drive tree <folder-id>            # Show tree from specific folder
  gro drive tree --drive "Engineering"  # Show tree from shared drive root
  gro drive tree --depth 3              # Limit depth
  gro drive tree --files                # Include files, not just folders
  gro drive tree --json                 # Output as JSON`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			// Validate mutually exclusive flags
			if myDrive && driveFlag != "" {
				return fmt.Errorf("--my-drive and --drive are mutually exclusive")
			}

			client, err := newDriveClient()
			if err != nil {
				return fmt.Errorf("creating Drive client: %w", err)
			}

			folderID := "root"
			rootName := "My Drive"

			if len(args) > 0 {
				folderID = args[0]
				rootName = "" // Will be fetched from folder info
			} else if driveFlag != "" {
				// Resolve shared drive
				scope, err := resolveDriveScope(client, false, driveFlag)
				if err != nil {
					return fmt.Errorf("resolving drive: %w", err)
				}
				folderID = scope.DriveID
				rootName = driveFlag // Use the provided name
			}

			// Build the tree
			tree, err := buildTreeWithScope(client, folderID, rootName, depth, files)
			if err != nil {
				return fmt.Errorf("building folder tree: %w", err)
			}

			if jsonOutput {
				return printJSON(tree)
			}

			printTree(tree, "", true)
			return nil
		},
	}

	cmd.Flags().IntVarP(&depth, "depth", "d", 2, "Maximum depth to traverse")
	cmd.Flags().BoolVar(&files, "files", false, "Include files in addition to folders")
	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output results as JSON")
	cmd.Flags().BoolVar(&myDrive, "my-drive", false, "Show My Drive only (default)")
	cmd.Flags().StringVar(&driveFlag, "drive", "", "Show tree from specific shared drive (name or ID)")

	return cmd
}

// buildTree recursively builds the folder tree structure
func buildTree(client drive.DriveClientInterface, folderID string, depth int, includeFiles bool) (*TreeNode, error) {
	return buildTreeWithScope(client, folderID, "", depth, includeFiles)
}

// buildTreeWithScope builds folder tree with optional root name override
func buildTreeWithScope(client drive.DriveClientInterface, folderID, rootName string, depth int, includeFiles bool) (*TreeNode, error) {
	// Get folder info
	var folderName string
	var folderType string

	if folderID == "root" {
		folderName = "My Drive"
		folderType = "Folder"
	} else if rootName != "" && depth == 2 { // First call with override
		folderName = rootName
		folderType = "Shared Drive"
	} else {
		folder, err := client.GetFile(folderID)
		if err != nil {
			return nil, fmt.Errorf("getting folder info: %w", err)
		}
		folderName = folder.Name
		folderType = drive.GetTypeName(folder.MimeType)
	}

	node := &TreeNode{
		ID:   folderID,
		Name: folderName,
		Type: folderType,
	}

	// Stop if we've reached the depth limit
	if depth <= 0 {
		return node, nil
	}

	// Build query to list children - use scope for shared drive support
	query := fmt.Sprintf("'%s' in parents and trashed = false", folderID)
	if !includeFiles {
		query += fmt.Sprintf(" and mimeType = '%s'", drive.MimeTypeFolder)
	}

	// Use ListFilesWithScope to support shared drives
	scope := drive.DriveScope{AllDrives: true}
	children, err := client.ListFilesWithScope(query, 100, scope)
	if err != nil {
		return nil, fmt.Errorf("listing children: %w", err)
	}

	// Sort children: folders first, then by name
	sort.Slice(children, func(i, j int) bool {
		iIsFolder := children[i].MimeType == drive.MimeTypeFolder
		jIsFolder := children[j].MimeType == drive.MimeTypeFolder
		if iIsFolder != jIsFolder {
			return iIsFolder // folders first
		}
		return children[i].Name < children[j].Name
	})

	// Process children
	for _, child := range children {
		if child.MimeType == drive.MimeTypeFolder {
			// Recursively build subtree for folders (don't pass rootName on recursion)
			childNode, err := buildTreeWithScope(client, child.ID, "", depth-1, includeFiles)
			if err != nil {
				// Log error but continue with other children
				continue
			}
			node.Children = append(node.Children, childNode)
		} else {
			// Add file as leaf node
			node.Children = append(node.Children, &TreeNode{
				ID:   child.ID,
				Name: child.Name,
				Type: drive.GetTypeName(child.MimeType),
			})
		}
	}

	return node, nil
}

// printTree prints the tree structure with tree characters
func printTree(node *TreeNode, prefix string, isRoot bool) {
	if isRoot {
		fmt.Println(node.Name)
	}

	for i, child := range node.Children {
		isLast := i == len(node.Children)-1

		// Print the current line
		if isLast {
			fmt.Printf("%s└── %s\n", prefix, child.Name)
		} else {
			fmt.Printf("%s├── %s\n", prefix, child.Name)
		}

		// Print children with updated prefix
		if len(child.Children) > 0 {
			var newPrefix string
			if isLast {
				newPrefix = prefix + "    "
			} else {
				newPrefix = prefix + "│   "
			}
			printTree(child, newPrefix, false)
		}
	}
}
