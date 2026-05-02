package cpp

import (
	"bufio"
	"os"
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
			classSymbols, shouldContinue := processClassLine(line, currentClass)
			symbols = append(symbols, classSymbols...)

			if currentClass.braceDepth <= 0 {
				currentClass = nil
			}

			if shouldContinue {
				continue
			}
		}

		symbol, ok := extractDeclaredSymbolFromLine(line)
		if !ok {
			continue
		}

		symbols = append(symbols, symbol)

		if classCtx := classContextFromSymbolLine(symbol, line); classCtx != nil {
			currentClass = classCtx
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

func processClassLine(line string, currentClass *classContext) ([]DeclaredSymbol, bool) {
	if visibility, ok := parseVisibilityLabel(line); ok {
		currentClass.visibility = visibility
		currentClass.braceDepth += braceDelta(line)
		return nil, true
	}

	symbols := make([]DeclaredSymbol, 0, 2)

	if symbol, ok := extractFriendFromLine(line, currentClass.name, currentClass.visibility); ok {
		symbols = append(symbols, symbol)
	}

	if currentClass.visibility != "public" {
		return symbols, false
	}

	if symbol, ok := extractMethodFromLine(line, currentClass.name); ok {
		symbols = append(symbols, symbol)
	}

	return symbols, false
}
