package cpp

import "github.com/orurh/patchcourt/internal/model"

type DeclaredSymbol struct {
	Name       string
	Kind       model.SymbolKind
	Parent     string
	Signature  string
	Modifiers  []string
	Exported   bool
	Visibility string
	Confidence model.Confidence
}

func (s DeclaredSymbol) ToModel(filePath string) model.SymbolModel {
	return model.SymbolModel{
		Name:       s.Name,
		Kind:       s.Kind,
		File:       filePath,
		Parent:     s.Parent,
		Signature:  s.Signature,
		Modifiers:  s.Modifiers,
		Exported:   s.Exported,
		Visibility: s.Visibility,
		Confidence: s.Confidence,
	}
}
