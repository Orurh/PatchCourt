package model

type SymbolKind string

const (
	SymbolKindUnknown SymbolKind = "unknown"
	SymbolKindClass   SymbolKind = "class"
	SymbolKindStruct  SymbolKind = "struct"
	SymbolKindEnum    SymbolKind = "enum"
	SymbolKindUsing   SymbolKind = "using"
	SymbolKindTypedef SymbolKind = "typedef"
	SymbolKindMethod  SymbolKind = "method"
)

type SymbolModel struct {
	Name       string     `json:"name"`
	Kind       SymbolKind `json:"kind"`
	File       string     `json:"file,omitempty"`
	Parent     string     `json:"parent,omitempty"`
	Signature  string     `json:"signature,omitempty"`
	Exported   bool       `json:"exported"`
	Visibility string     `json:"visibility,omitempty"`
	Confidence Confidence `json:"confidence"`
}
