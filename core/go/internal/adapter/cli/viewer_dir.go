package cli

import (
	"fmt"
	"os"
	"path/filepath"
)

func resolveViewerDir(explicit string) (string, error) {
	if explicit != "" {
		return requireViewerDist(explicit)
	}

	for _, candidate := range viewerDirCandidates() {
		resolved, err := requireViewerDist(candidate)
		if err == nil {
			return resolved, nil
		}
	}

	return "", fmt.Errorf(`viewer dist not found

Build it with:
  make viewer-build

Or pass it explicitly:
  patchcourt open . --viewer-dir /path/to/viewer/dist

Release archives should contain:
  patchcourt
  viewer-dist/index.html`)
}

func resolveOptionalViewerDir(explicit string) (string, error) {
	if explicit == "" {
		return "", nil
	}

	return requireViewerDist(explicit)
}

func viewerDirCandidates() []string {
	candidates := []string{
		"viewer-dist",
		filepath.Join("web", "viewer", "dist"),
		filepath.Join("..", "web", "viewer", "dist"),
		filepath.Join("..", "..", "web", "viewer", "dist"),
	}

	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "viewer-dist"),
			filepath.Join(exeDir, "..", "viewer-dist"),
			filepath.Join(exeDir, "web", "viewer", "dist"),
		)
	}

	return candidates
}

func requireViewerDist(dir string) (string, error) {
	if dir == "" {
		return "", fmt.Errorf("viewer dist path is empty")
	}

	clean := filepath.Clean(dir)
	indexPath := filepath.Join(clean, "index.html")

	info, err := os.Stat(indexPath)
	if err != nil {
		return "", fmt.Errorf("viewer dist %q is not valid: missing index.html", clean)
	}
	if info.IsDir() {
		return "", fmt.Errorf("viewer dist %q is not valid: index.html is a directory", clean)
	}

	abs, err := filepath.Abs(clean)
	if err != nil {
		return clean, nil
	}

	return abs, nil
}
