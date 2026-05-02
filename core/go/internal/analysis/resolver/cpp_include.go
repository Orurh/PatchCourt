package resolver

import (
	"os"
	"path"
	"path/filepath"

	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/platform/pathmatch"
)

type IncludePath struct {
	Path       string
	Source     model.ResolutionSource
	Confidence model.ResolutionConfidence
	System     bool
}

type IncludeResolution struct {
	ToFile     string
	Resolved   bool
	External   bool
	Source     model.ResolutionSource
	Confidence model.ResolutionConfidence
	Ambiguous  bool
	Candidates []string
}

type CPPIncludeResolver struct {
	root         string
	fileIndex    FileIndex
	includePaths []IncludePath
}

func NewCPPIncludeResolver(root string, fileIndex FileIndex, includePaths []IncludePath) CPPIncludeResolver {
	normalizedIncludePaths := make([]IncludePath, 0, len(includePaths))
	for _, includePath := range includePaths {
		normalized := pathmatch.Normalize(includePath.Path)
		if normalized == "" {
			continue
		}
		source := includePath.Source
		if source == "" {
			source = model.ResolutionSourceConfig
		}

		confidence := includePath.Confidence
		if confidence == "" {
			confidence = model.ResolutionConfidenceHigh
		}

		includePath.Path = normalized
		includePath.Source = source
		includePath.Confidence = confidence
		normalizedIncludePaths = append(normalizedIncludePaths, includePath)
	}

	return CPPIncludeResolver{
		root:         root,
		fileIndex:    fileIndex,
		includePaths: normalizedIncludePaths,
	}
}

func (r CPPIncludeResolver) Resolve(fromFile string, target string) IncludeResolution {
	normalizedTarget := pathmatch.Normalize(target)

	if resolved, ok := r.fileIndex.ResolvePath(normalizedTarget); ok {
		return IncludeResolution{
			ToFile:     resolved,
			Resolved:   true,
			Source:     model.ResolutionSourceHeuristic,
			Confidence: model.ResolutionConfidenceMedium,
		}
	}

	for _, includePath := range r.includePaths {
		candidate := path.Clean(path.Join(includePath.Path, normalizedTarget))

		if includePath.System && r.fileExists(candidate) {
			return IncludeResolution{
				Resolved:   false,
				External:   true,
				Source:     includePath.Source,
				Confidence: includePath.Confidence,
			}
		}

		if resolved, ok := r.fileIndex.ResolvePath(candidate); ok {
			return IncludeResolution{
				ToFile:     resolved,
				Resolved:   true,
				Source:     includePath.Source,
				Confidence: includePath.Confidence,
			}
		}

		if r.fileExists(candidate) {
			return IncludeResolution{
				Resolved:   false,
				External:   true,
				Source:     includePath.Source,
				Confidence: includePath.Confidence,
			}
		}
	}

	fromDir := path.Dir(pathmatch.Normalize(fromFile))
	relativeToCurrentFile := path.Clean(path.Join(fromDir, normalizedTarget))

	if resolved, ok := r.fileIndex.ResolvePath(relativeToCurrentFile); ok {
		return IncludeResolution{
			ToFile:     resolved,
			Resolved:   true,
			Source:     model.ResolutionSourceHeuristic,
			Confidence: model.ResolutionConfidenceMedium,
		}
	}

	candidates := r.fileIndex.ResolveBase(path.Base(normalizedTarget))
	if len(candidates) == 1 {
		return IncludeResolution{
			ToFile:     candidates[0],
			Resolved:   true,
			Source:     model.ResolutionSourceHeuristic,
			Confidence: model.ResolutionConfidenceLow,
			Candidates: candidates,
		}
	}

	if len(candidates) > 1 {
		return IncludeResolution{
			Resolved:   false,
			Source:     model.ResolutionSourceHeuristic,
			Confidence: model.ResolutionConfidenceLow,
			Ambiguous:  true,
			Candidates: candidates,
		}
	}

	return IncludeResolution{
		Resolved:   false,
		Source:     model.ResolutionSourceNone,
		Confidence: model.ResolutionConfidenceLow,
	}
}

func (r CPPIncludeResolver) fileExists(projectRelativePath string) bool {
	if r.root == "" {
		return false
	}

	absPath := filepath.Join(r.root, filepath.FromSlash(projectRelativePath))
	info, err := os.Stat(absPath)
	return err == nil && !info.IsDir()
}
