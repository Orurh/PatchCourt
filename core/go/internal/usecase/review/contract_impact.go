package review

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	contracts "github.com/orurh/patchcourt/internal/diff/contract"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/platform/pathmatch"
	"github.com/orurh/patchcourt/internal/reportmodel"
)

const (
	contractImpactBreaking      = "breaking"
	contractImpactRisky         = "risky"
	contractImpactAdditive      = "additive"
	contractImpactInformational = "informational"

	contractImpactConfidenceLow    = "low"
	contractImpactConfidenceMedium = "medium"
)

func BuildContractImpacts(
	changes []contracts.SymbolChange,
	afterProject *model.ProjectModel,
	changedFiles []string,
) []reportmodel.ContractImpact {
	if len(changes) == 0 {
		return nil
	}

	impacts := make([]reportmodel.ContractImpact, 0, len(changes))

	for _, change := range changes {
		parentName, methodName := contractSymbolParts(change.SymbolKey)
		impact := reportmodel.ContractImpact{
			SymbolKey:     change.SymbolKey,
			ChangeKind:    string(change.Kind),
			Impact:        classifyContractImpact(change),
			Location:      contractChangeLocation(change),
			ParentName:    parentName,
			MethodName:    methodName,
			TestsChanged:  anyTestLikeFileChanged(changedFiles),
			Confidence:    contractImpactConfidenceLow,
			ImpactedFiles: nil,
		}

		if afterProject != nil && methodName != "" {
			impact.ImpactedFiles = scanContractImpactedFiles(afterProject, change, parentName, methodName)
			if len(impact.ImpactedFiles) > 0 {
				impact.Confidence = contractImpactConfidenceMedium
			}

			for _, item := range impact.ImpactedFiles {
				if isDeliveryLayerOrPath(item.Layer, item.File) {
					impact.DeliveryImpacted = true
					break
				}
			}
		}

		impacts = append(impacts, impact)
	}

	return impacts
}

func contractSymbolParts(symbolKey string) (parentName string, methodName string) {
	parts := strings.Split(symbolKey, "::")
	if len(parts) < 3 {
		return "", ""
	}

	return parts[len(parts)-2], parts[len(parts)-1]
}

func scanContractImpactedFiles(
	project *model.ProjectModel,
	change contracts.SymbolChange,
	parentName string,
	methodName string,
) []reportmodel.ContractImpactedFile {
	root := project.Root
	if root == "" {
		return nil
	}

	contractFile := contractChangeFile(change)
	fileLayers := projectFileLayers(project)
	files := projectFilePaths(project)

	seen := make(map[string]struct{})
	items := make([]reportmodel.ContractImpactedFile, 0)

	add := func(file string, reason string, line int) {
		if file == "" {
			return
		}

		key := file + "|" + reason
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}

		items = append(items, reportmodel.ContractImpactedFile{
			File:   file,
			Layer:  fileLayers[file],
			Reason: reason,
			Line:   line,
		})
	}

	for _, file := range files {
		if file == "" {
			continue
		}

		content, lines, ok := readProjectFile(root, file)
		if !ok {
			continue
		}

		if file != contractFile && parentName != "" && likelyImplementsParent(content, parentName) {
			add(file, "likely_implementation", firstLineContaining(lines, parentName))
		}

		if file != contractFile && likelyReferencesMethod(content, methodName) {
			add(file, "likely_method_reference", firstLineContaining(lines, methodName+"("))
		}
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Reason != items[j].Reason {
			return items[i].Reason < items[j].Reason
		}
		return items[i].File < items[j].File
	})

	return items
}

func readProjectFile(root string, file string) (content string, lines []string, ok bool) {
	path := filepath.Join(root, filepath.FromSlash(file))

	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil, false
	}

	content = string(data)
	lines = strings.Split(content, "\n")
	return content, lines, true
}

func projectFileLayers(project *model.ProjectModel) map[string]string {
	layers := make(map[string]string, len(project.Files))

	for _, file := range project.Files {
		if file.Path == "" {
			continue
		}

		layers[file.Path] = file.Layer
	}

	return layers
}

func projectFilePaths(project *model.ProjectModel) []string {
	seen := make(map[string]struct{})

	add := func(file string) {
		if file == "" {
			return
		}
		seen[file] = struct{}{}
	}

	for _, file := range project.Files {
		add(file.Path)
	}

	for _, symbol := range project.Symbols {
		add(symbol.File)
	}

	for _, dep := range project.Dependencies {
		add(dep.FromFile)
		add(dep.ToFile)
	}

	files := make([]string, 0, len(seen))
	for file := range seen {
		files = append(files, file)
	}

	sort.Strings(files)
	return files
}

func likelyImplementsParent(content string, parentName string) bool {
	if parentName == "" {
		return false
	}

	return strings.Contains(content, ": public "+parentName) ||
		strings.Contains(content, ": public virtual "+parentName) ||
		strings.Contains(content, ": "+parentName)
}

func likelyReferencesMethod(content string, methodName string) bool {
	if methodName == "" {
		return false
	}

	return strings.Contains(content, methodName+"(")
}

func firstLineContaining(lines []string, needle string) int {
	if needle == "" {
		return 0
	}

	for i, line := range lines {
		if strings.Contains(line, needle) {
			return i + 1
		}
	}

	return 0
}

func classifyContractImpact(change contracts.SymbolChange) string {
	switch change.Kind {
	case contracts.ChangeKindRemoved:
		return contractImpactBreaking

	case contracts.ChangeKindSignatureChanged:
		return contractImpactBreaking

	case contracts.ChangeKindAdded:
		if change.After != nil && symbolHasPureVirtual(change.After) {
			return contractImpactBreaking
		}
		return contractImpactAdditive

	case contracts.ChangeKindModifiersChanged:
		if containsString(change.AddedMods, "pure_virtual") {
			return contractImpactBreaking
		}

		if containsAnyString(change.RemovedMods, []string{
			"virtual",
			"const",
			"noexcept",
			"override",
			"final",
			"pure_virtual",
		}) {
			return contractImpactRisky
		}

		if containsAnyString(change.AddedMods, []string{
			"final",
			"override",
			"noexcept",
		}) {
			return contractImpactRisky
		}

		return contractImpactInformational

	default:
		return contractImpactInformational
	}
}

func symbolHasPureVirtual(symbol *model.SymbolModel) bool {
	if symbol == nil {
		return false
	}

	if containsString(symbol.Modifiers, "pure_virtual") {
		return true
	}

	return strings.Contains(symbol.Signature, "= 0")
}

func contractChangeLocation(change contracts.SymbolChange) string {
	file := contractChangeFile(change)
	if file == "" {
		return ""
	}

	beforeLine := 0
	if change.Before != nil {
		beforeLine = change.Before.Line
	}

	afterLine := 0
	if change.After != nil {
		afterLine = change.After.Line
	}

	switch {
	case beforeLine > 0 && afterLine > 0 && beforeLine != afterLine:
		return file + ":" + intString(beforeLine) + " → " + intString(afterLine)
	case afterLine > 0:
		return file + ":" + intString(afterLine)
	case beforeLine > 0:
		return file + ":" + intString(beforeLine)
	default:
		return file
	}
}

func contractChangeFile(change contracts.SymbolChange) string {
	if change.After != nil && change.After.File != "" {
		return change.After.File
	}

	if change.Before != nil && change.Before.File != "" {
		return change.Before.File
	}

	return ""
}

func intString(value int) string {
	return strconv.FormatInt(int64(value), 10)
}

func anyTestLikeFileChanged(files []string) bool {
	for _, file := range files {
		if pathmatch.IsTestLikeFile(file) {
			return true
		}
	}

	return false
}

func isDeliveryLayerOrPath(layer string, file string) bool {
	layer = strings.ToLower(layer)
	file = strings.ToLower(strings.ReplaceAll(file, "\\", "/"))

	switch layer {
	case "api", "server", "web", "http", "grpc", "controllers", "controller", "delivery":
		return true
	}

	return strings.Contains(file, "/api/") ||
		strings.Contains(file, "/server/") ||
		strings.Contains(file, "/controllers/") ||
		strings.Contains(file, "/controller/")
}

func containsAnyString(values []string, targets []string) bool {
	for _, target := range targets {
		if containsString(values, target) {
			return true
		}
	}

	return false
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}

	return false
}
