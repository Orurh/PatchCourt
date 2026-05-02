package cpp

import (
	"bufio"
	"os"
	"strings"

	"github.com/orurh/patchcourt/internal/model"
)

func ExtractDeclaredSymbols(path string) ([]DeclaredSymbol, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var symbols []DeclaredSymbol
	var currentClass *classContext

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := normalizeDeclarationLine(scanner.Text())
		if line == "" {
			continue
		}

		if currentClass != nil {
			if visibility, ok := parseVisibilityLabel(line); ok {
				currentClass.visibility = visibility
				currentClass.braceDepth += braceDelta(line)
				if currentClass.braceDepth <= 0 {
					currentClass = nil
				}
				continue
			}

			if symbol, ok := extractFriendFromLine(line, currentClass.name, currentClass.visibility); ok {
				symbols = append(symbols, symbol)
			}

			if currentClass.visibility == "public" {
				if symbol, ok := extractMethodFromLine(line, currentClass.name); ok {
					symbols = append(symbols, symbol)
				}
			}
		}

		if symbol, ok := extractDeclaredSymbolFromLine(line); ok {
			symbols = append(symbols, symbol)

			if symbol.Kind == model.SymbolKindClass || symbol.Kind == model.SymbolKindStruct {
				if strings.Contains(line, "{") {
					currentClass = &classContext{
						name:       symbol.Name,
						kind:       symbol.Kind,
						visibility: defaultVisibility(symbol.Kind),
						braceDepth: braceDelta(line),
					}

					if currentClass.braceDepth <= 0 {
						currentClass = nil
					}
				}
			}
		}

		if currentClass != nil {
			currentClass.braceDepth += braceDeltaWithoutOpeningClassLine(line)
			if currentClass.braceDepth <= 0 {
				currentClass = nil
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return symbols, nil
}
