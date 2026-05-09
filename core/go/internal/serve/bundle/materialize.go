package bundle

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type MaterializedReviewRoots struct {
	TempDir    string
	BeforeRoot string
	AfterRoot  string
}

func materializeReviewRoots(ctx context.Context, projectRoot string, baseRef string, headRef string, worktree bool) (*MaterializedReviewRoots, error) {
	if projectRoot == "" {
		return nil, fmt.Errorf("project root is required")
	}
	if strings.TrimSpace(baseRef) == "" {
		return nil, fmt.Errorf("base ref is required")
	}
	if worktree && strings.TrimSpace(headRef) != "" {
		return nil, fmt.Errorf("head ref cannot be set when worktree=true")
	}
	if !worktree && strings.TrimSpace(headRef) == "" {
		headRef = "HEAD"
	}

	tempDir, err := os.MkdirTemp("", "patchcourt-review-roots-*")
	if err != nil {
		return nil, fmt.Errorf("create temp review roots: %w", err)
	}

	roots := &MaterializedReviewRoots{
		TempDir:    tempDir,
		BeforeRoot: filepath.Join(tempDir, "before"),
		AfterRoot:  filepath.Join(tempDir, "after"),
	}

	if err := os.MkdirAll(roots.BeforeRoot, 0o755); err != nil {
		_ = os.RemoveAll(tempDir)
		return nil, fmt.Errorf("create before root: %w", err)
	}
	if err := os.MkdirAll(roots.AfterRoot, 0o755); err != nil {
		_ = os.RemoveAll(tempDir)
		return nil, fmt.Errorf("create after root: %w", err)
	}

	if err := extractGitArchive(ctx, projectRoot, baseRef, roots.BeforeRoot); err != nil {
		_ = os.RemoveAll(tempDir)
		return nil, fmt.Errorf("materialize base %q: %w", baseRef, err)
	}

	if worktree {
		if err := copyProjectWorktree(projectRoot, roots.AfterRoot); err != nil {
			_ = os.RemoveAll(tempDir)
			return nil, fmt.Errorf("materialize working tree: %w", err)
		}
	} else {
		if err := extractGitArchive(ctx, projectRoot, headRef, roots.AfterRoot); err != nil {
			_ = os.RemoveAll(tempDir)
			return nil, fmt.Errorf("materialize head %q: %w", headRef, err)
		}
	}

	return roots, nil
}

func extractGitArchive(ctx context.Context, projectRoot string, ref string, outDir string) error {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "archive", "--format=tar", ref)
	cmd.Dir = projectRoot

	data, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git archive %s: %w: %s", ref, err, strings.TrimSpace(string(data)))
	}

	reader := tar.NewReader(bytes.NewReader(data))

	for {
		header, err := reader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read archive: %w", err)
		}

		cleanName := filepath.Clean(header.Name)
		if cleanName == "." || strings.HasPrefix(cleanName, ".."+string(filepath.Separator)) || filepath.IsAbs(cleanName) {
			return fmt.Errorf("unsafe archive path: %s", header.Name)
		}

		target := filepath.Join(outDir, cleanName)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, modeOrDefault(header.FileInfo().Mode(), 0o755)); err != nil {
				return fmt.Errorf("create archive directory %s: %w", target, err)
			}

		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return fmt.Errorf("create archive parent %s: %w", filepath.Dir(target), err)
			}

			file, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, modeOrDefault(header.FileInfo().Mode(), 0o644))
			if err != nil {
				return fmt.Errorf("create archive file %s: %w", target, err)
			}

			_, copyErr := io.Copy(file, reader)
			closeErr := file.Close()
			if copyErr != nil {
				return fmt.Errorf("write archive file %s: %w", target, copyErr)
			}
			if closeErr != nil {
				return fmt.Errorf("close archive file %s: %w", target, closeErr)
			}

		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return fmt.Errorf("create symlink parent %s: %w", filepath.Dir(target), err)
			}
			if err := os.Symlink(header.Linkname, target); err != nil {
				return fmt.Errorf("create symlink %s: %w", target, err)
			}

		default:
			// Skip unsupported archive entries for MVP.
		}
	}
}

func copyProjectWorktree(srcRoot string, dstRoot string) error {
	return filepath.WalkDir(srcRoot, func(srcPath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, err := filepath.Rel(srcRoot, srcPath)
		if err != nil {
			return err
		}

		if rel == "." {
			return nil
		}

		if shouldSkipWorktreePath(rel, entry) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		dstPath := filepath.Join(dstRoot, rel)

		info, err := entry.Info()
		if err != nil {
			return err
		}

		switch {
		case entry.IsDir():
			return os.MkdirAll(dstPath, modeOrDefault(info.Mode(), 0o755))

		case info.Mode().Type() == fs.ModeSymlink:
			linkTarget, err := os.Readlink(srcPath)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
				return err
			}
			return os.Symlink(linkTarget, dstPath)

		case info.Mode().IsRegular():
			if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
				return err
			}
			return copyRegularFile(srcPath, dstPath, modeOrDefault(info.Mode(), 0o644))

		default:
			return nil
		}
	})
}

func shouldSkipWorktreePath(rel string, entry fs.DirEntry) bool {
	normalized := filepath.ToSlash(rel)

	if normalized == ".git" || strings.HasPrefix(normalized, ".git/") {
		return true
	}

	if normalized == ".patchcourt/out" || strings.HasPrefix(normalized, ".patchcourt/out/") {
		return true
	}

	if normalized == "node_modules" || strings.Contains(normalized, "/node_modules/") {
		return true
	}

	if normalized == "build" || strings.HasPrefix(normalized, "build/") {
		return true
	}

	if normalized == "dist" || strings.HasPrefix(normalized, "dist/") {
		return true
	}

	if entry.IsDir() && strings.HasPrefix(entry.Name(), ".cache") {
		return true
	}

	return false
}

func copyRegularFile(src string, dst string, mode fs.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}

	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()

	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func modeOrDefault(mode fs.FileMode, fallback fs.FileMode) fs.FileMode {
	if mode == 0 {
		return fallback
	}
	return mode
}
