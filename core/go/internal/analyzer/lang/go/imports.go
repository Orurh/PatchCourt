package goanalysis

import (
	"go/parser"
	"go/token"
	"strconv"
)

func ParseImports(path string) ([]string, error) {
	fileSet := token.NewFileSet()

	file, err := parser.ParseFile(fileSet, path, nil, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}

	imports := make([]string, 0, len(file.Imports))
	for _, spec := range file.Imports {
		value, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}

		if value == "" {
			continue
		}

		imports = append(imports, value)
	}

	return imports, nil
}
