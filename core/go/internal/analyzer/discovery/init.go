package discovery

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/orurh/patchcourt/internal/analyzer/lang/cpp/resolver"
	"github.com/orurh/patchcourt/internal/analyzer/project"
	"github.com/orurh/patchcourt/internal/config"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/platform/pathmatch"
)

type InitOptions struct {
	Root   string
	Strict bool
	Preset string
}

type InitResult struct {
	ConfigYAML string
}

type discoveredLayer struct {
	Name       string
	Paths      map[string]struct{}
	DependOn   map[string]struct{}
	SourceDirs map[string]struct{}
}

func GenerateInitConfig(opts InitOptions) (*InitResult, error) {
	root := opts.Root
	if root == "" {
		root = "."
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve root: %w", err)
	}

	ignorePaths := config.DefaultIgnorePaths()
	includePathStrings := discoverCPPIncludePaths(absRoot)
	if opts.Preset == "nested-cpp" {
		includePathStrings = appendExistingIncludePaths(absRoot, includePathStrings, []string{
			"src/core",
			"src/utility",
			"src/utils",
		})
	}

	includePaths := configIncludePaths(includePathStrings)

	project, err := project.Build(project.Options{
		Root:            absRoot,
		IgnorePaths:     ignorePaths,
		CPPIncludePaths: includePaths,
	})
	if err != nil {
		return nil, fmt.Errorf("scan project for init: %w", err)
	}

	if opts.Preset != "" && opts.Preset != "auto" {
		configYAML, err := renderPresetConfig(absRoot, ignorePaths, includePathStrings, project, opts)
		if err != nil {
			return nil, err
		}

		return &InitResult{
			ConfigYAML: configYAML,
		}, nil
	}

	layers := discoverLayers(project, opts.Strict)

	configYAML := renderConfig(ignorePaths, includePathStrings, layers, opts.Strict)

	return &InitResult{
		ConfigYAML: configYAML,
	}, nil
}

func discoverCPPIncludePaths(absRoot string) []string {
	candidates := []string{
		"src",
		"include",
	}

	var result []string
	for _, candidate := range candidates {
		if dirExists(filepath.Join(absRoot, filepath.FromSlash(candidate))) {
			result = append(result, candidate)
		}
	}

	return result
}

func configIncludePaths(paths []string) []resolver.IncludePath {
	result := make([]resolver.IncludePath, 0, len(paths))
	for _, path := range paths {
		result = append(result, resolver.IncludePath{
			Path:       path,
			Source:     model.ResolutionSourceConfig,
			Confidence: model.ResolutionConfidenceHigh,
		})
	}

	return result
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func discoverLayers(project *model.ProjectModel, strict bool) []discoveredLayer {
	layersByName := make(map[string]*discoveredLayer)
	fileToLayer := make(map[string]string)

	for _, file := range project.Files {
		if file.Role != model.FileRoleProduction {
			continue
		}

		layerName, pattern, ok := inferLayerFromPath(file.Path)
		if !ok {
			continue
		}

		layer := ensureLayer(layersByName, layerName)
		layer.Paths[pattern] = struct{}{}
		layer.SourceDirs[sourceDirFromPattern(pattern)] = struct{}{}
		fileToLayer[file.Path] = layerName
	}

	if !strict {
		for _, dep := range project.Dependencies {
			if dep.External || !dep.Resolved {
				continue
			}

			fromLayer := fileToLayer[dep.FromFile]
			toLayer := fileToLayer[dep.ToFile]

			if fromLayer == "" || toLayer == "" || fromLayer == toLayer {
				continue
			}

			layer := ensureLayer(layersByName, fromLayer)
			layer.DependOn[toLayer] = struct{}{}
		}
	}

	layers := make([]discoveredLayer, 0, len(layersByName))
	for _, layer := range layersByName {
		layers = append(layers, *layer)
	}

	sort.Slice(layers, func(i, j int) bool {
		return layers[i].Name < layers[j].Name
	})

	return layers
}

func ensureLayer(layers map[string]*discoveredLayer, name string) *discoveredLayer {
	if layer, ok := layers[name]; ok {
		return layer
	}

	layer := &discoveredLayer{
		Name:       name,
		Paths:      make(map[string]struct{}),
		DependOn:   make(map[string]struct{}),
		SourceDirs: make(map[string]struct{}),
	}

	layers[name] = layer
	return layer
}

func inferLayerFromPath(filePath string) (layerName string, pattern string, ok bool) {
	normalized := pathmatch.Normalize(filePath)
	parts := strings.Split(normalized, "/")

	if len(parts) < 2 {
		return "", "", false
	}

	switch parts[0] {
	case "src":
		return sanitizeLayerName(parts[1]), "src/" + parts[1] + "/**", true
	case "internal":
		return sanitizeLayerName("internal_" + parts[1]), "internal/" + parts[1] + "/**", true
	case "pkg":
		return sanitizeLayerName("pkg_" + parts[1]), "pkg/" + parts[1] + "/**", true
	case "include":
		return sanitizeLayerName(parts[1]), "include/" + parts[1] + "/**", true
	default:
		return "", "", false
	}
}

func sanitizeLayerName(value string) string {
	value = strings.ToLower(value)

	var b strings.Builder
	lastUnderscore := false

	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastUnderscore = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastUnderscore = false
		default:
			if !lastUnderscore {
				b.WriteRune('_')
				lastUnderscore = true
			}
		}
	}

	result := strings.Trim(b.String(), "_")
	if result == "" {
		return "unknown"
	}

	return result
}

func sourceDirFromPattern(pattern string) string {
	return strings.TrimSuffix(pattern, "/**")
}

func renderConfig(ignorePaths []string, includePaths []string, layers []discoveredLayer, strict bool) string {
	var b bytes.Buffer

	b.WriteString("# Generated by PatchCourt.\n")
	b.WriteString("# Review this file before using it in CI.\n")
	b.WriteString("# The initial architecture is inferred from the current project structure and dependency graph.\n")
	if strict {
		b.WriteString("# Strict mode: may_depend_on is intentionally empty for discovered layers.\n")
	} else {
		b.WriteString("# Baseline mode: may_depend_on is inferred from current dependencies.\n")
	}
	b.WriteString("\n")

	b.WriteString("ignore:\n")
	b.WriteString("  paths:\n")
	for _, path := range ignorePaths {
		fmt.Fprintf(&b, "    - %q\n", path)
	}

	b.WriteString("\n")
	b.WriteString("cpp:\n")
	b.WriteString("  compile_commands:\n")
	b.WriteString("    auto_discover: true\n")

	if len(includePaths) > 0 {
		b.WriteString("  include_paths:\n")
		for _, path := range includePaths {
			fmt.Fprintf(&b, "    - %q\n", path)
		}
	}

	b.WriteString("\n")
	b.WriteString("layers:\n")

	if len(layers) == 0 {
		b.WriteString("  # No layers were discovered. Add paths manually.\n")
		return b.String()
	}

	for _, layer := range layers {
		fmt.Fprintf(&b, "  %s:\n", layer.Name)

		b.WriteString("    paths:\n")
		for _, path := range sortedKeys(layer.Paths) {
			fmt.Fprintf(&b, "      - %q\n", path)
		}

		deps := sortedKeys(layer.DependOn)
		if len(deps) == 0 {
			b.WriteString("    may_depend_on: []\n")
		} else {
			b.WriteString("    may_depend_on:\n")
			for _, dep := range deps {
				fmt.Fprintf(&b, "      - %s\n", dep)
			}
		}

		b.WriteString("\n")
	}

	return b.String()
}

func sortedKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}

	sort.Strings(keys)
	return keys
}

type presetLayer struct {
	Name        string
	Paths       []string
	MayDependOn []string
}

func renderPresetConfig(absRoot string, ignorePaths []string, includePaths []string, projectModel *model.ProjectModel, opts InitOptions) (string, error) {
	switch opts.Preset {
	case "go-clean":
		return renderGoCleanPresetConfig(absRoot, ignorePaths, includePaths), nil
	case "nested-cpp":
		return renderNestedCPPPresetConfig(absRoot, ignorePaths, includePaths, projectModel, opts.Strict), nil
	default:
		return "", fmt.Errorf("unknown init preset %q", opts.Preset)
	}
}

func renderNestedCPPPresetConfig(absRoot string, ignorePaths []string, includePaths []string, projectModel *model.ProjectModel, strict bool) string {
	layers := discoverNestedCPPLayers(absRoot)

	if !strict && projectModel != nil {
		fillNestedCPPBaselineDependencies(layers, projectModel)
	}

	return renderPresetLayersConfig(ignorePaths, includePaths, layers, "nested-cpp")
}

func discoverNestedCPPLayers(absRoot string) []presetLayer {
	layers := make([]presetLayer, 0)

	coreRoot := filepath.Join(absRoot, "src", "core")
	if entries, err := os.ReadDir(coreRoot); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			name := nestedCPPLayerName(entry.Name())
			if name == "" {
				continue
			}

			relPath := "src/core/" + entry.Name() + "/**"
			if !hasReviewableSourceUnder(absRoot, strings.TrimSuffix(relPath, "/**")) {
				continue
			}

			layers = append(layers, presetLayer{
				Name:  name,
				Paths: []string{relPath},
			})
		}
	}

	for _, candidate := range []struct {
		name string
		path string
	}{
		{name: "utility", path: "src/utility/**"},
		{name: "utils", path: "src/utils/**"},
		{name: "include", path: "include/**"},
	} {
		if !presetLayerExists(absRoot, candidate.path) {
			continue
		}

		if !hasReviewableSourceUnder(absRoot, strings.TrimSuffix(candidate.path, "/**")) {
			continue
		}

		if containsPresetLayer(layers, candidate.name) {
			continue
		}

		layers = append(layers, presetLayer{
			Name:  candidate.name,
			Paths: []string{candidate.path},
		})
	}

	sort.SliceStable(layers, func(i int, j int) bool {
		return layers[i].Name < layers[j].Name
	})

	return layers
}

func fillNestedCPPBaselineDependencies(layers []presetLayer, projectModel *model.ProjectModel) {
	if len(layers) == 0 || projectModel == nil {
		return
	}

	existing := make(map[string]struct{}, len(layers))
	for _, layer := range layers {
		existing[layer.Name] = struct{}{}
	}

	depsByLayer := make(map[string]map[string]struct{})

	for _, dep := range projectModel.Dependencies {
		if dep.External || !dep.Resolved {
			continue
		}

		fromLayer := nestedCPPLayerForFile(layers, dep.FromFile)
		toLayer := nestedCPPLayerForFile(layers, dep.ToFile)

		if fromLayer == "" || toLayer == "" || fromLayer == toLayer {
			continue
		}

		if _, ok := existing[fromLayer]; !ok {
			continue
		}
		if _, ok := existing[toLayer]; !ok {
			continue
		}

		if depsByLayer[fromLayer] == nil {
			depsByLayer[fromLayer] = make(map[string]struct{})
		}
		depsByLayer[fromLayer][toLayer] = struct{}{}
	}

	for i := range layers {
		deps := sortedKeys(depsByLayer[layers[i].Name])
		layers[i].MayDependOn = deps
	}
}

func nestedCPPLayerForFile(layers []presetLayer, file string) string {
	for _, layer := range layers {
		for _, pattern := range layer.Paths {
			if pathmatch.Match(pattern, file) {
				return layer.Name
			}
		}
	}

	return ""
}

func nestedCPPLayerName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return ""
	}

	var b strings.Builder
	lastUnderscore := false

	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastUnderscore = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastUnderscore = false
		case r == '_' || r == '-' || r == '.' || r == ' ':
			if !lastUnderscore {
				b.WriteRune('_')
				lastUnderscore = true
			}
		}
	}

	result := strings.Trim(b.String(), "_")
	if result == "" {
		return ""
	}

	return result
}

func containsPresetLayer(layers []presetLayer, name string) bool {
	for _, layer := range layers {
		if layer.Name == name {
			return true
		}
	}

	return false
}

func hasReviewableSourceUnder(absRoot string, relDir string) bool {
	dir := filepath.Join(absRoot, filepath.FromSlash(relDir))

	found := false
	_ = filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if found {
			return filepath.SkipAll
		}

		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "build", "cmake-build-debug", "cmake-build-release", "node_modules", "vendor", "third_party", "external", "generated":
				return filepath.SkipDir
			}
			return nil
		}

		name := strings.ToLower(entry.Name())
		if strings.HasSuffix(name, ".cc") ||
			strings.HasSuffix(name, ".cpp") ||
			strings.HasSuffix(name, ".cxx") ||
			strings.HasSuffix(name, ".c") ||
			strings.HasSuffix(name, ".h") ||
			strings.HasSuffix(name, ".hpp") ||
			strings.HasSuffix(name, ".hh") ||
			strings.HasSuffix(name, ".hxx") {
			found = true
		}

		return nil
	})

	return found
}

func appendExistingIncludePaths(absRoot string, paths []string, candidates []string) []string {
	result := append([]string(nil), paths...)

	for _, candidate := range candidates {
		if !dirExists(filepath.Join(absRoot, filepath.FromSlash(candidate))) {
			continue
		}

		if stringSliceContains(result, candidate) {
			continue
		}

		result = append(result, candidate)
	}

	return result
}

func stringSliceContains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}

	return false
}

func renderGoCleanPresetConfig(absRoot string, ignorePaths []string, includePaths []string) string {
	layers := []presetLayer{
		{
			Name:        "cmd",
			Paths:       []string{"cmd/**"},
			MayDependOn: []string{"cli"},
		},
		{
			Name:        "cli",
			Paths:       []string{"internal/adapter/cli/**"},
			MayDependOn: []string{"usecase", "render"},
		},
		{
			Name:        "usecase",
			Paths:       []string{"internal/usecase/**"},
			MayDependOn: []string{"analyzer", "config", "diff", "model", "platform", "reportmodel", "source", "state"},
		},
		{
			Name:        "diff",
			Paths:       []string{"internal/diff/**"},
			MayDependOn: []string{"model", "platform"},
		},
		{
			Name:        "source",
			Paths:       []string{"internal/source/**"},
			MayDependOn: []string{"analyzer", "model", "platform", "state"},
		},
		{
			Name:        "state",
			Paths:       []string{"internal/state/**"},
			MayDependOn: []string{"model", "platform"},
		},
		{
			Name:        "analyzer",
			Paths:       []string{"internal/analyzer/**"},
			MayDependOn: []string{"config", "model", "platform"},
		},
		{
			Name:        "config",
			Paths:       []string{"internal/config/**"},
			MayDependOn: []string{"model"},
		},
		{
			Name:        "model",
			Paths:       []string{"internal/model/**"},
			MayDependOn: []string{},
		},
		{
			Name:        "reportmodel",
			Paths:       []string{"internal/reportmodel/**"},
			MayDependOn: []string{"analyzer", "config", "model"},
		},
		{
			Name:        "render",
			Paths:       []string{"internal/render/**"},
			MayDependOn: []string{"analyzer", "model", "platform", "reportmodel"},
		},
		{
			Name:        "platform",
			Paths:       []string{"internal/platform/**"},
			MayDependOn: []string{},
		},
	}

	existing := make(map[string]struct{})
	filtered := make([]presetLayer, 0, len(layers))

	for _, layer := range layers {
		if len(layer.Paths) == 0 {
			continue
		}

		if !presetLayerExists(absRoot, layer.Paths[0]) {
			continue
		}

		existing[layer.Name] = struct{}{}
		filtered = append(filtered, layer)
	}

	for i := range filtered {
		deps := filtered[i].MayDependOn[:0]
		for _, dep := range filtered[i].MayDependOn {
			if _, ok := existing[dep]; ok {
				deps = append(deps, dep)
			}
		}
		filtered[i].MayDependOn = deps
	}

	return renderPresetLayersConfig(ignorePaths, includePaths, filtered, "go-clean")
}

func presetLayerExists(absRoot string, pattern string) bool {
	dir := strings.TrimSuffix(pattern, "/**")
	return dirExists(filepath.Join(absRoot, filepath.FromSlash(dir)))
}

func renderPresetLayersConfig(ignorePaths []string, includePaths []string, layers []presetLayer, preset string) string {
	var b bytes.Buffer

	b.WriteString("# Generated by PatchCourt.\n")
	b.WriteString("# Review this file before using it in CI.\n")
	fmt.Fprintf(&b, "# Preset: %s\n", preset)
	b.WriteString("\n")

	b.WriteString("ignore:\n")
	b.WriteString("  paths:\n")
	for _, path := range ignorePaths {
		fmt.Fprintf(&b, "    - %q\n", path)
	}

	b.WriteString("\n")
	b.WriteString("cpp:\n")
	b.WriteString("  compile_commands:\n")
	b.WriteString("    auto_discover: true\n")

	if len(includePaths) > 0 {
		b.WriteString("  include_paths:\n")
		for _, path := range includePaths {
			fmt.Fprintf(&b, "    - %q\n", path)
		}
	}

	b.WriteString("\n")
	b.WriteString("layers:\n")

	for _, layer := range layers {
		fmt.Fprintf(&b, "  %s:\n", layer.Name)

		b.WriteString("    paths:\n")
		for _, path := range layer.Paths {
			fmt.Fprintf(&b, "      - %q\n", path)
		}

		if len(layer.MayDependOn) == 0 {
			b.WriteString("    may_depend_on: []\n")
		} else {
			b.WriteString("    may_depend_on:\n")
			for _, dep := range layer.MayDependOn {
				fmt.Fprintf(&b, "      - %s\n", dep)
			}
		}

		b.WriteString("\n")
	}

	return b.String()
}
