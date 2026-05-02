package cpp

import (
	"bufio"
	"os"
	"regexp"
)

type IncludeKind string

const (
	IncludeKindLocal  IncludeKind = "local"
	IncludeKindSystem IncludeKind = "system"
)

type Include struct {
	Target string
	Kind   IncludeKind
}

var includeRE = regexp.MustCompile(`^\s*#\s*include\s*([<"])([^>"]+)[>"]`)

func ParseIncludes(path string) ([]Include, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var includes []Include

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		match := includeRE.FindStringSubmatch(scanner.Text())
		if len(match) != 3 {
			continue
		}

		kind := IncludeKindLocal
		if match[1] == "<" {
			kind = IncludeKindSystem
		}

		includes = append(includes, Include{
			Target: match[2],
			Kind:   kind,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return includes, nil
}
