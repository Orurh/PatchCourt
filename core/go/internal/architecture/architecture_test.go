package architecture

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type forbiddenFragment struct {
	Name     string
	Pattern  string
	Prefixes []string
}

var legacyFragments = []forbiddenFragment{
	{Name: "analysis", Pattern: "internal/analysis", Prefixes: []string{"internal/analysis/"}},
	{Name: "app", Pattern: "internal/app", Prefixes: []string{"internal/app/"}},
	{Name: "cli", Pattern: "internal/cli", Prefixes: []string{"internal/cli/"}},
	{Name: "changes", Pattern: "internal/changes", Prefixes: []string{"internal/changes/"}},
	{Name: "output", Pattern: "internal/output", Prefixes: []string{"internal/output/"}},
	{Name: "render/report", Pattern: "internal/render/report", Prefixes: []string{"internal/render/report/"}},
}

var forbiddenCoreOutputPattern = regexp.MustCompile(`fmt\.Print|os\.Stdout|os\.Stderr|log\.Print|println\(`)

func TestNoLegacyInternalPathsOrImports(t *testing.T) {
	root := moduleRoot(t)

	var violations []string

	walkGoFiles(t, root, func(path string, rel string, data string) {
		if rel == "internal/architecture/architecture_test.go" {
			return
		}

		for _, fragment := range legacyFragments {
			for _, prefix := range fragment.Prefixes {
				if strings.HasPrefix(rel, prefix) {
					violations = append(violations, rel+": legacy path "+fragment.Name)
				}
			}

			if strings.Contains(data, "github.com/orurh/patchcourt/"+fragment.Pattern) {
				violations = append(violations, rel+": legacy import "+fragment.Name)
			}
		}
	})

	require.Empty(t, violations, strings.Join(violations, "\n"))
}

func TestCoreDoesNotWriteDirectlyToStdoutOrStderr(t *testing.T) {
	root := moduleRoot(t)

	var violations []string

	walkGoFiles(t, root, func(path string, rel string, data string) {
		if isAllowedOutputPackage(rel) {
			return
		}

		if strings.HasSuffix(rel, "_test.go") {
			return
		}

		if forbiddenCoreOutputPattern.MatchString(data) {
			violations = append(violations, rel)
		}
	})

	require.Empty(t, violations, strings.Join(violations, "\n"))
}

func TestUsecaseSubpackagesDoNotDependOnRootFacade(t *testing.T) {
	root := moduleRoot(t)

	var violations []string

	walkGoFiles(t, root, func(path string, rel string, data string) {
		if !strings.HasPrefix(rel, "internal/usecase/") {
			return
		}

		rest := strings.TrimPrefix(rel, "internal/usecase/")
		if !strings.Contains(rest, "/") {
			return
		}

		if strings.Contains(data, `"github.com/orurh/patchcourt/internal/usecase"`) {
			violations = append(violations, rel+": imports root usecase facade")
		}

		if regexp.MustCompile(`\btype\s+App\b|\*App\b`).MatchString(data) {
			violations = append(violations, rel+": depends on root App")
		}
	})

	require.Empty(t, violations, strings.Join(violations, "\n"))
}

func moduleRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	require.NoError(t, err)

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		require.NotEqual(t, dir, parent, "go.mod not found from %s", dir)
		dir = parent
	}
}

func walkGoFiles(t *testing.T, root string, visit func(path string, rel string, data string)) {
	t.Helper()

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		require.NoError(t, err)

		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "bin", "out", "vendor", "node_modules":
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		require.NoError(t, err)

		rel = filepath.ToSlash(rel)

		data, err := os.ReadFile(path)
		require.NoError(t, err)

		visit(path, rel, string(data))
		return nil
	})

	require.NoError(t, err)
}

func isAllowedOutputPackage(rel string) bool {
	return rel == "cmd/patchcourt/main.go" ||
		strings.HasPrefix(rel, "internal/adapter/cli/") ||
		strings.HasPrefix(rel, "internal/render/") ||
		strings.HasPrefix(rel, "internal/platform/logx/")
}
