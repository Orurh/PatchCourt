package compilecmds

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/shlex"
	"github.com/orurh/patchcourt/internal/platform/pathmatch"
)

type Entry struct {
	Directory string `json:"directory"`
	File      string `json:"file"`
	// Command — строковая shell-команда из compile_commands.json.
	// Arguments предпочтительнее, если присутствует: там аргументы уже
	// разложены по массиву согласно формату JSON Compilation Database.
	Command   string   `json:"command"`
	Arguments []string `json:"arguments"`
}

type Database struct {
	Entries []Entry
}

type IncludePathKind string

const (
	IncludePathKindNormal IncludePathKind = "normal"
	IncludePathKindSystem IncludePathKind = "system"
	IncludePathKindQuote  IncludePathKind = "quote"
)

type IncludePathEntry struct {
	Path string          `json:"path"`
	Kind IncludePathKind `json:"kind"`
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

// IncludePaths returns normalized include path strings for backward compatibility.
// Prefer IncludePathEntries when caller needs to distinguish -I from -isystem/-iquote.
func IncludePaths(db *Database, root string) []string {
	entries := IncludePathEntries(db, root)

	result := make([]string, 0, len(entries))
	for _, entry := range entries {
		result = append(result, entry.Path)
	}

	return result
}

func IncludePathEntries(db *Database, root string) []IncludePathEntry {
	if db == nil {
		return nil
	}

	seen := make(map[string]struct{})
	result := make([]IncludePathEntry, 0)

	for _, entry := range db.Entries {
		args := entryArgs(entry)

		entryDir := entry.Directory
		if entryDir == "" {
			entryDir = root
		}

		for _, includePath := range extractIncludePathEntries(args) {
			normalized := normalizeIncludePath(root, entryDir, includePath.Path)
			if normalized == "" {
				continue
			}

			key := string(includePath.Kind) + "|" + normalized
			if _, ok := seen[key]; ok {
				continue
			}

			seen[key] = struct{}{}
			result = append(result, IncludePathEntry{
				Path: normalized,
				Kind: includePath.Kind,
			})
		}
	}

	return result
}

func entryArgs(entry Entry) []string {
	if len(entry.Arguments) > 0 {
		return entry.Arguments
	}

	if entry.Command == "" {
		return nil
	}

	args, err := shlex.Split(entry.Command)
	if err == nil {
		return args
	}

	return splitCommand(entry.Command)
}

func extractIncludePathEntries(args []string) []IncludePathEntry {
	result := make([]IncludePathEntry, 0)

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch {
		case arg == "-I" || arg == "--include-directory":
			if i+1 < len(args) {
				result = append(result, IncludePathEntry{
					Path: args[i+1],
					Kind: IncludePathKindNormal,
				})
				i++
			}

		case arg == "-isystem":
			if i+1 < len(args) {
				result = append(result, IncludePathEntry{
					Path: args[i+1],
					Kind: IncludePathKindSystem,
				})
				i++
			}

		case arg == "-iquote":
			if i+1 < len(args) {
				result = append(result, IncludePathEntry{
					Path: args[i+1],
					Kind: IncludePathKindQuote,
				})
				i++
			}

		case strings.HasPrefix(arg, "-I") && arg != "-I":
			result = append(result, IncludePathEntry{
				Path: strings.TrimPrefix(arg, "-I"),
				Kind: IncludePathKindNormal,
			})

		case strings.HasPrefix(arg, "-isystem="):
			result = append(result, IncludePathEntry{
				Path: strings.TrimPrefix(arg, "-isystem="),
				Kind: IncludePathKindSystem,
			})

		case strings.HasPrefix(arg, "-iquote="):
			result = append(result, IncludePathEntry{
				Path: strings.TrimPrefix(arg, "-iquote="),
				Kind: IncludePathKindQuote,
			})

		case strings.HasPrefix(arg, "--include-directory="):
			result = append(result, IncludePathEntry{
				Path: strings.TrimPrefix(arg, "--include-directory="),
				Kind: IncludePathKindNormal,
			})
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
