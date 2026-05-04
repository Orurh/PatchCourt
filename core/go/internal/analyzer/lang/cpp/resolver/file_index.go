package resolver

import (
	"path/filepath"
	"sort"

	"github.com/orurh/patchcourt/internal/model"
)

type FileIndex struct {
	byPath map[string]string
	byBase map[string][]string
}

func NewFileIndex(files []model.FileModel) FileIndex {
	index := FileIndex{
		byPath: make(map[string]string, len(files)),
		byBase: make(map[string][]string, len(files)),
	}

	for _, file := range files {
		index.byPath[file.Path] = file.Path

		base := filepath.Base(file.Path)
		index.byBase[base] = append(index.byBase[base], file.Path)
	}

	return index
}

func (i FileIndex) ResolvePath(path string) (string, bool) {
	resolved, ok := i.byPath[path]
	return resolved, ok
}

func (i FileIndex) ResolveBase(base string) []string {
	return i.byBase[base]
}

func (i FileIndex) Files() []string {
	files := make([]string, 0, len(i.byPath))
	for file := range i.byPath {
		files = append(files, file)
	}

	sort.Strings(files)
	return files
}
