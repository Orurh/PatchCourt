package model

type Language string

const (
	LanguageUnknown Language = "unknown"
	LanguageCPP     Language = "cpp"
	LanguageGo      Language = "go"
)

type FileKind string

const (
	FileKindUnknown FileKind = "unknown"
	FileKindHeader  FileKind = "header"
	FileKindSource  FileKind = "source"
	FileKindTest    FileKind = "test"
)

type FileRole string

const (
	FileRoleUnknown    FileRole = "unknown"
	FileRoleProduction FileRole = "production"
	FileRoleTest       FileRole = "test"
	FileRoleGenerated  FileRole = "generated"
	FileRoleExternal   FileRole = "external"
	FileRoleConfig     FileRole = "config"
)

type LayerAssignmentSource string

const (
	LayerAssignmentSourceUnknown    LayerAssignmentSource = ""
	LayerAssignmentSourceConfig     LayerAssignmentSource = "config"
	LayerAssignmentSourceDiscovered LayerAssignmentSource = "discovered"
)

type FileModel struct {
	Path        string                `json:"path"`
	Language    Language              `json:"language"`
	Kind        FileKind              `json:"kind"`
	Role        FileRole              `json:"role"`
	Layer       string                `json:"layer,omitempty"`
	LayerSource LayerAssignmentSource `json:"layer_source,omitempty"`
	IsTest      bool                  `json:"is_test"`
	Includes    []string              `json:"includes,omitempty"`
	Imports     []string              `json:"imports,omitempty"`
	Symbols     []SymbolModel         `json:"symbols,omitempty"`
}
