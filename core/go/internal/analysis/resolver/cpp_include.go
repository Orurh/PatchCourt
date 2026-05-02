package resolver

import (
	"path"

	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/platform/pathmatch"
)

type IncludeResolution struct {
	ToFile     string
	Resolved   bool
	Source     model.ResolutionSource
	Confidence model.ResolutionConfidence
	Ambiguous  bool
	Candidates []string
}

type CPPIncludeResolver struct {
	fileIndex    FileIndex
	includePaths []string
}

func NewCPPIncludeResolver(fileIndex FileIndex, includePaths []string) CPPIncludeResolver {
	normalizedIncludePaths := make([]string, 0, len(includePaths))
	for _, includePath := range includePaths {
		normalized := pathmatch.Normalize(includePath)
		if normalized == "" {
			continue
		}
		normalizedIncludePaths = append(normalizedIncludePaths, normalized)
	}

	return CPPIncludeResolver{
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
		candidate := path.Clean(path.Join(includePath, normalizedTarget))
		if resolved, ok := r.fileIndex.ResolvePath(candidate); ok {
			return IncludeResolution{
				ToFile:     resolved,
				Resolved:   true,
				Source:     model.ResolutionSourceConfig,
				Confidence: model.ResolutionConfidenceHigh,
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
