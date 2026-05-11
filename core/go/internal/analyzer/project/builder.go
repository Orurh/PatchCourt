package project

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/orurh/patchcourt/internal/analyzer/lang/cpp"
	"github.com/orurh/patchcourt/internal/analyzer/lang/cpp/resolver"
	goanalysis "github.com/orurh/patchcourt/internal/analyzer/lang/go"
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

	if err := collectCPPDependencies(absRoot, project, cppIncludeResolver); err != nil {
		return nil, err
	}

	if err := collectGoDependencies(absRoot, project, fileIndex); err != nil {
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

func collectCPPDependencies(absRoot string, project *model.ProjectModel, cppIncludeResolver resolver.CPPIncludeResolver) error {
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

type goModuleRoot struct {
	AbsPath    string
	RelPath    string
	ModulePath string
}

func collectGoDependencies(absRoot string, project *model.ProjectModel, fileIndex resolver.FileIndex) error {
	modules, err := discoverGoModuleRoots(absRoot)
	if err != nil {
		return err
	}
	if len(modules) == 0 {
		return nil
	}

	for i := range project.Files {
		file := &project.Files[i]

		if file.Language != model.LanguageGo {
			continue
		}

		module, ok := goModuleForFile(modules, file.Path)
		if !ok {
			continue
		}

		absPath := filepath.Join(absRoot, filepath.FromSlash(file.Path))

		imports, err := goanalysis.ParseImports(absPath)
		if err != nil {
			return fmt.Errorf("parse go imports %s: %w", file.Path, err)
		}

		for _, importPath := range imports {
			edge := model.DependencyEdge{
				FromFile: file.Path,
				Target:   importPath,
				Kind:     model.DependencyKindImport,
				Usage:    model.DependencyUsageUnknown,
			}

			relDir, local := localGoImportDir(module.ModulePath, importPath)
			if !local {
				edge.External = true
				edge.Resolved = false
				edge.ResolutionSource = model.ResolutionSourceNone
				edge.ResolutionConfidence = model.ResolutionConfidenceLow
				project.Dependencies = append(project.Dependencies, edge)
				continue
			}

			if resolved := resolveGoPackageFileInModule(fileIndex, module.RelPath, relDir); resolved != "" {
				edge.ToFile = resolved
				edge.Resolved = true
				edge.ResolutionSource = model.ResolutionSourceHeuristic
				edge.ResolutionConfidence = model.ResolutionConfidenceHigh
			} else {
				edge.Resolved = false
				edge.ResolutionSource = model.ResolutionSourceNone
				edge.ResolutionConfidence = model.ResolutionConfidenceLow
			}

			project.Dependencies = append(project.Dependencies, edge)
		}
	}

	return nil
}

func discoverGoModuleRoots(absRoot string) ([]goModuleRoot, error) {
	modules := make([]goModuleRoot, 0)

	err := filepath.WalkDir(absRoot, func(currentPath string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if entry.IsDir() {
			if currentPath != absRoot && shouldSkipDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}

		if entry.Name() != "go.mod" {
			return nil
		}

		moduleAbs := filepath.Dir(currentPath)
		modulePath := goanalysis.ModulePath(moduleAbs)
		if modulePath == "" {
			return nil
		}

		relPath, err := filepath.Rel(absRoot, moduleAbs)
		if err != nil {
			return fmt.Errorf("relative go module root: %w", err)
		}

		relPath = pathmatch.Normalize(relPath)
		if relPath == "." {
			relPath = ""
		}

		modules = append(modules, goModuleRoot{
			AbsPath:    moduleAbs,
			RelPath:    relPath,
			ModulePath: modulePath,
		})

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("discover go modules: %w", err)
	}

	sort.Slice(modules, func(i, j int) bool {
		if len(modules[i].RelPath) != len(modules[j].RelPath) {
			return len(modules[i].RelPath) > len(modules[j].RelPath)
		}
		return modules[i].RelPath < modules[j].RelPath
	})

	return modules, nil
}

func goModuleForFile(modules []goModuleRoot, filePath string) (goModuleRoot, bool) {
	filePath = pathmatch.Normalize(filePath)

	for _, module := range modules {
		if module.RelPath == "" {
			return module, true
		}

		if filePath == module.RelPath || strings.HasPrefix(filePath, module.RelPath+"/") {
			return module, true
		}
	}

	return goModuleRoot{}, false
}

func localGoImportDir(modulePath string, importPath string) (string, bool) {
	if importPath == modulePath {
		return "", true
	}

	prefix := modulePath + "/"
	if !strings.HasPrefix(importPath, prefix) {
		return "", false
	}

	return pathmatch.Normalize(strings.TrimPrefix(importPath, prefix)), true
}

func resolveGoPackageFileInModule(index resolver.FileIndex, moduleRelPath string, relDir string) string {
	packageDir := joinSlashPath(moduleRelPath, relDir)

	candidates := []string{
		joinSlashPath(packageDir, "doc.go"),
		joinSlashPath(packageDir, "main.go"),
	}

	for _, candidate := range candidates {
		if resolved, ok := index.ResolvePath(candidate); ok {
			return resolved
		}
	}

	prefix := packageDir
	if prefix != "" {
		prefix += "/"
	}

	for _, file := range index.Files() {
		if prefix != "" && !strings.HasPrefix(file, prefix) {
			continue
		}

		if !strings.HasSuffix(file, ".go") {
			continue
		}

		if strings.HasSuffix(file, "_test.go") {
			continue
		}

		return file
	}

	return ""
}

func joinSlashPath(left string, right string) string {
	left = strings.Trim(pathmatch.Normalize(left), "/")
	right = strings.Trim(pathmatch.Normalize(right), "/")

	switch {
	case left == "":
		return right
	case right == "":
		return left
	default:
		return path.Join(left, right)
	}
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
