package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/orurh/patchcourt/internal/analysis/lang/cpp"
	"github.com/orurh/patchcourt/internal/analysis/resolver"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/platform/pathmatch"
)

type Options struct {
	Root            string
	IgnorePaths     []string
	CPPIncludePaths []resolver.IncludePath
}

func Build(opts Options) (*model.ProjectModel, error) {
	root := opts.Root
	if root == "" {
		root = "."
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve root: %w", err)
	}

	project := &model.ProjectModel{
		Root: absRoot,
	}

	if err := collectFiles(absRoot, opts.IgnorePaths, project); err != nil {
		return nil, err
	}

	if err := collectSymbols(absRoot, project); err != nil {
		return nil, err
	}

	fileIndex := resolver.NewFileIndex(project.Files)
	cppIncludeResolver := resolver.NewCPPIncludeResolver(absRoot, fileIndex, opts.CPPIncludePaths)

	if err := collectDependencies(absRoot, project, cppIncludeResolver); err != nil {
		return nil, err
	}

	return project, nil
}

func collectFiles(absRoot string, ignorePaths []string, project *model.ProjectModel) error {
	return filepath.WalkDir(absRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relPath, err := filepath.Rel(absRoot, path)
		if err != nil {
			return fmt.Errorf("relative path: %w", err)
		}

		normalizedRelPath := filepath.ToSlash(relPath)

		if entry.IsDir() {
			if shouldSkipDir(entry.Name()) || shouldIgnorePath(normalizedRelPath, ignorePaths) {
				return filepath.SkipDir
			}
			return nil
		}

		if shouldIgnorePath(normalizedRelPath, ignorePaths) {
			return nil
		}

		lang := DetectLanguage(path)
		if lang == model.LanguageUnknown {
			return nil
		}

		isTest := IsTestFile(normalizedRelPath)

		file := model.FileModel{
			Path:     normalizedRelPath,
			Language: lang,
			Kind:     DetectFileKind(normalizedRelPath, lang),
			Role:     DetectFileRole(normalizedRelPath, lang),
			IsTest:   isTest,
		}

		project.Files = append(project.Files, file)
		return nil
	})
}

func collectDependencies(absRoot string, project *model.ProjectModel, cppIncludeResolver resolver.CPPIncludeResolver) error {
	for i := range project.Files {
		file := &project.Files[i]

		if file.Language != model.LanguageCPP {
			continue
		}

		absPath := filepath.Join(absRoot, filepath.FromSlash(file.Path))

		includes, err := cpp.ParseIncludes(absPath)
		if err != nil {
			return fmt.Errorf("parse includes %s: %w", file.Path, err)
		}

		for _, include := range includes {
			file.Includes = append(file.Includes, include.Target)

			edge := model.DependencyEdge{
				FromFile: file.Path,
				Target:   include.Target,
				Kind:     model.DependencyKindInclude,
				Usage:    model.DependencyUsageUnknown,
				External: include.Kind == cpp.IncludeKindSystem,
			}

			if edge.External {
				edge.ResolutionSource = model.ResolutionSourceNone
				edge.ResolutionConfidence = model.ResolutionConfidenceLow
			} else {
				resolution := cppIncludeResolver.Resolve(file.Path, include.Target)
				edge.ToFile = resolution.ToFile
				edge.Resolved = resolution.Resolved
				edge.External = resolution.External
				edge.ResolutionSource = resolution.Source
				edge.ResolutionConfidence = resolution.Confidence
				edge.Ambiguous = resolution.Ambiguous
				edge.Candidates = resolution.Candidates
			}

			project.Dependencies = append(project.Dependencies, edge)
		}
	}

	return nil
}

func shouldSkipDir(name string) bool {
	switch strings.ToLower(name) {
	case ".git", "build", "cmake-build-debug", "cmake-build-release", "node_modules", "vendor", ".idea", ".vscode":
		return true
	default:
		return false
	}
}

func shouldIgnorePath(path string, patterns []string) bool {
	if path == "." {
		return false
	}

	return pathmatch.MatchAny(patterns, path)
}

func collectSymbols(absRoot string, project *model.ProjectModel) error {
	for i := range project.Files {
		file := &project.Files[i]

		if file.Language != model.LanguageCPP {
			continue
		}

		if file.Role != model.FileRoleProduction {
			continue
		}

		absPath := filepath.Join(absRoot, filepath.FromSlash(file.Path))

		declaredSymbols, err := cpp.ExtractDeclaredSymbols(absPath)
		if err != nil {
			return fmt.Errorf("extract symbols %s: %w", file.Path, err)
		}

		for _, declaredSymbol := range declaredSymbols {
			symbol := declaredSymbol.ToModel(file.Path)

			file.Symbols = append(file.Symbols, symbol)
			project.Symbols = append(project.Symbols, symbol)
		}
	}

	return nil
}
