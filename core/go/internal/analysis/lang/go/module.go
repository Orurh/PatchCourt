package goanalysis

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

func ModulePath(root string) string {
	file, err := os.Open(filepath.Join(root, "go.mod"))
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}

	return ""
}
