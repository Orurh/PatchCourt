package bundle

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func resolveWorkspaceOutsideRoot(root string, workspace string) (string, error) {
	if workspace == "" {
		cacheDir, err := os.UserCacheDir()
		if err != nil || cacheDir == "" {
			cacheDir = os.TempDir()
		}

		workspace = filepath.Join(cacheDir, "patchcourt", "server")
	}

	absWorkspace, err := filepath.Abs(workspace)
	if err != nil {
		return "", fmt.Errorf("resolve workspace path: %w", err)
	}

	if root == "" {
		return absWorkspace, nil
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve project root: %w", err)
	}

	rel, err := filepath.Rel(absRoot, absWorkspace)
	if err != nil {
		return "", fmt.Errorf("compare workspace and project root: %w", err)
	}

	if rel == "." || (!strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != "..") {
		return "", fmt.Errorf("workspace must be outside checked project root: workspace=%s root=%s", absWorkspace, absRoot)
	}

	return absWorkspace, nil
}
