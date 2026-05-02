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

type FileModel struct {
	Path     string   `json:"path"`
	Language Language `json:"language"`
	Kind     FileKind `json:"kind"`
	Role     FileRole `json:"role"`
	Layer    string   `json:"layer,omitempty"`
	IsTest   bool     `json:"is_test"`
	Includes []string `json:"includes,omitempty"`
	Imports  []string `json:"imports,omitempty"`
}
