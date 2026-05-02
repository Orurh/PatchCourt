package compilecmds

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/orurh/patchcourt/internal/platform/pathmatch"
)

type Entry struct {
	Directory string   `json:"directory"`
	File      string   `json:"file"`
	Command   string   `json:"command"`
	Arguments []string `json:"arguments"`
}

type Database struct {
	Entries []Entry
}

func Load(path string) (*Database, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read compile commands %s: %w", path, err)
	}

	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("parse compile commands %s: %w", path, err)
	}

	return &Database{
		Entries: entries,
	}, nil
}

func IncludePaths(db *Database, root string) []string {
	if db == nil {
		return nil
	}

	seen := make(map[string]struct{})
	result := make([]string, 0)

	for _, entry := range db.Entries {
		args := entry.Arguments
		if len(args) == 0 && entry.Command != "" {
			args = splitCommand(entry.Command)
		}

		entryDir := entry.Directory
		if entryDir == "" {
			entryDir = root
		}

		for _, includePath := range extractIncludePaths(args) {
			normalized := normalizeIncludePath(root, entryDir, includePath)
			if normalized == "" {
				continue
			}

			if _, ok := seen[normalized]; ok {
				continue
			}

			seen[normalized] = struct{}{}
			result = append(result, normalized)
		}
	}

	return result
}

func extractIncludePaths(args []string) []string {
	result := make([]string, 0)

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch {
		case arg == "-I" || arg == "-isystem" || arg == "-iquote" || arg == "--include-directory":
			if i+1 < len(args) {
				result = append(result, args[i+1])
				i++
			}

		case strings.HasPrefix(arg, "-I") && arg != "-I":
			result = append(result, strings.TrimPrefix(arg, "-I"))

		case strings.HasPrefix(arg, "-isystem="):
			result = append(result, strings.TrimPrefix(arg, "-isystem="))

		case strings.HasPrefix(arg, "-iquote="):
			result = append(result, strings.TrimPrefix(arg, "-iquote="))

		case strings.HasPrefix(arg, "--include-directory="):
			result = append(result, strings.TrimPrefix(arg, "--include-directory="))
		}
	}

	return result
}

func normalizeIncludePath(root string, entryDir string, includePath string) string {
	includePath = strings.TrimSpace(includePath)
	if includePath == "" {
		return ""
	}

	if !filepath.IsAbs(includePath) {
		includePath = filepath.Join(entryDir, includePath)
	}

	rel, err := filepath.Rel(root, includePath)
	if err != nil {
		return ""
	}

	if strings.HasPrefix(rel, "..") {
		return ""
	}

	return pathmatch.Normalize(rel)
}

func splitCommand(command string) []string {
	fields := make([]string, 0)
	var current strings.Builder
	inSingleQuote := false
	inDoubleQuote := false

	for _, r := range command {
		switch {
		case r == '\'' && !inDoubleQuote:
			inSingleQuote = !inSingleQuote
		case r == '"' && !inSingleQuote:
			inDoubleQuote = !inDoubleQuote
		case r == ' ' || r == '\t' || r == '\n':
			if inSingleQuote || inDoubleQuote {
				current.WriteRune(r)
				continue
			}

			if current.Len() > 0 {
				fields = append(fields, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		fields = append(fields, current.String())
	}

	return fields
}
