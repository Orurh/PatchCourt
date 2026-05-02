package resolver

import (
	"path/filepath"

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
