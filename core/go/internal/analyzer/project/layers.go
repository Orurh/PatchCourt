package project

import (
	"path"
	"strings"

	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/platform/pathmatch"
)

// AssignDiscoveredLayers assigns a deterministic, structure-based layer to each
// production file. This is intentionally not a policy decision.
//
// It answers:
//
//	"Where does this file appear to belong based on repository layout?"
//
// It does not answer:
//
//	"Is this dependency allowed?"
func AssignDiscoveredLayers(project *model.ProjectModel) {
	if project == nil {
		return
	}

	for i := range project.Files {
		file := &project.Files[i]

		if file.Role != model.FileRoleProduction {
			continue
		}

		layer := DiscoverLayer(file.Path)
		if layer == "" {
			continue
		}

		file.Layer = layer
		file.LayerSource = model.LayerAssignmentSourceDiscovered
	}

	enrichDependencyLayers(project)
}

// DiscoverLayer returns a stable best-effort layer name derived from path.
//
// Rules are deliberately simple and deterministic:
//   - src/main.cc, main.cc             -> entrypoint
//   - src/<dir>/**                    -> <dir>
//   - include/<dir>/**                -> <dir>
//   - internal/<dir>/**               -> internal_<dir>
//   - pkg/<dir>/**                    -> pkg_<dir>
//   - generic shared-looking dirs     -> shared
func DiscoverLayer(filePath string) string {
	normalized := pathmatch.Normalize(filePath)
	if normalized == "" {
		return ""
	}

	parts := strings.Split(normalized, "/")
	base := strings.ToLower(path.Base(normalized))

	if isEntrypointFile(parts, base) {
		return "entrypoint"
	}

	if len(parts) < 2 {
		return ""
	}

	switch parts[0] {
	case "src":
		if len(parts) == 2 {
			return "entrypoint"
		}

		return normalizeDiscoveredLayer(parts[1])

	case "include":
		if len(parts) >= 3 {
			return normalizeDiscoveredLayer(parts[1])
		}

	case "internal":
		if len(parts) >= 3 {
			return normalizeDiscoveredLayer("internal_" + parts[1])
		}

	case "pkg":
		if len(parts) >= 3 {
			return normalizeDiscoveredLayer("pkg_" + parts[1])
		}
	}

	return ""
}

func enrichDependencyLayers(project *model.ProjectModel) {
	fileLayers := make(map[string]string, len(project.Files))
	for _, file := range project.Files {
		fileLayers[file.Path] = file.Layer
	}

	for i := range project.Dependencies {
		dep := &project.Dependencies[i]
		dep.FromLayer = fileLayers[dep.FromFile]

		if dep.Resolved {
			dep.ToLayer = fileLayers[dep.ToFile]
		}
	}
}

func isEntrypointFile(parts []string, base string) bool {
	switch base {
	case "main.c", "main.cc", "main.cpp", "main.cxx", "main.go":
		return true
	}

	return len(parts) == 2 && parts[0] == "cmd" && strings.HasSuffix(base, ".go")
}

func normalizeDiscoveredLayer(value string) string {
	value = strings.ToLower(value)

	switch value {
	case "utils", "util", "common", "shared", "configs", "config", "constants":
		return "shared"
	}

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
