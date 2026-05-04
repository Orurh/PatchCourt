package project

import (
	"testing"

	"github.com/orurh/patchcourt/internal/model"
)

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		name string
		path string
		want model.Language
	}{
		{
			name: "go file",
			path: "internal/usecase/scan.go",
			want: model.LanguageGo,
		},
		{
			name: "cpp header",
			path: "src/domain/interfaces/i_camera_adapter.h",
			want: model.LanguageCPP,
		},
		{
			name: "cpp hpp header",
			path: "include/camera_adapter.hpp",
			want: model.LanguageCPP,
		},
		{
			name: "cpp source cc",
			path: "src/server/api_router.cc",
			want: model.LanguageCPP,
		},
		{
			name: "cpp source cpp",
			path: "src/server/api_router.cpp",
			want: model.LanguageCPP,
		},
		{
			name: "unknown markdown",
			path: "README.md",
			want: model.LanguageUnknown,
		},
		{
			name: "case insensitive extension",
			path: "src/main.CPP",
			want: model.LanguageCPP,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectLanguage(tt.path)
			if got != tt.want {
				t.Fatalf("DetectLanguage(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestDetectFileKind(t *testing.T) {
	tests := []struct {
		name string
		path string
		lang model.Language
		want model.FileKind
	}{
		{
			name: "cpp header",
			path: "src/domain/foo.h",
			lang: model.LanguageCPP,
			want: model.FileKindHeader,
		},
		{
			name: "cpp source",
			path: "src/domain/foo.cpp",
			lang: model.LanguageCPP,
			want: model.FileKindSource,
		},
		{
			name: "cpp test by suffix",
			path: "tests/foo_test.cc",
			lang: model.LanguageCPP,
			want: model.FileKindTest,
		},
		{
			name: "cpp test by directory",
			path: "tests/foo.cc",
			lang: model.LanguageCPP,
			want: model.FileKindTest,
		},
		{
			name: "go file currently unknown kind",
			path: "internal/usecase/scan.go",
			lang: model.LanguageGo,
			want: model.FileKindUnknown,
		},
		{
			name: "unknown language",
			path: "README.md",
			lang: model.LanguageUnknown,
			want: model.FileKindUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectFileKind(tt.path, tt.lang)
			if got != tt.want {
				t.Fatalf("DetectFileKind(%q, %q) = %q, want %q", tt.path, tt.lang, got, tt.want)
			}
		})
	}
}

func TestDetectFileRole(t *testing.T) {
	tests := []struct {
		name string
		path string
		lang model.Language
		want model.FileRole
	}{
		{
			name: "production cpp",
			path: "src/server/api_router.cc",
			lang: model.LanguageCPP,
			want: model.FileRoleProduction,
		},
		{
			name: "production go",
			path: "internal/usecase/scan.go",
			lang: model.LanguageGo,
			want: model.FileRoleProduction,
		},
		{
			name: "test by suffix",
			path: "src/domain/foo_test.cc",
			lang: model.LanguageCPP,
			want: model.FileRoleTest,
		},
		{
			name: "test by directory",
			path: "tests/foo.cc",
			lang: model.LanguageCPP,
			want: model.FileRoleTest,
		},
		{
			name: "generated protobuf header",
			path: "generated/camera/foo.pb.h",
			lang: model.LanguageCPP,
			want: model.FileRoleGenerated,
		},
		{
			name: "generated marker",
			path: "src/proto/foo.generated.cc",
			lang: model.LanguageCPP,
			want: model.FileRoleGenerated,
		},
		{
			name: "external third party",
			path: "third_party/somelib/include/lib.h",
			lang: model.LanguageCPP,
			want: model.FileRoleExternal,
		},
		{
			name: "config cmake",
			path: "CMakeLists.txt",
			lang: model.LanguageUnknown,
			want: model.FileRoleConfig,
		},
		{
			name: "config yaml",
			path: ".patchcourt.yaml",
			lang: model.LanguageUnknown,
			want: model.FileRoleConfig,
		},
		{
			name: "unknown role",
			path: "README.md",
			lang: model.LanguageUnknown,
			want: model.FileRoleUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectFileRole(tt.path, tt.lang)
			if got != tt.want {
				t.Fatalf("DetectFileRole(%q, %q) = %q, want %q", tt.path, tt.lang, got, tt.want)
			}
		})
	}
}

func TestIsTestFile(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "go test suffix",
			path: "internal/usecase/scan_test.go",
			want: true,
		},
		{
			name: "cpp test suffix",
			path: "src/domain/foo_test.cc",
			want: true,
		},
		{
			name: "cpp spec suffix",
			path: "src/domain/foo_spec.cpp",
			want: true,
		},
		{
			name: "tests directory",
			path: "tests/device_orchestrator.cc",
			want: true,
		},
		{
			name: "integration tests directory",
			path: "integration_tests/api_router.cc",
			want: true,
		},
		{
			name: "production file",
			path: "src/server/api_router.cc",
			want: false,
		},
		{
			name: "word contest should not match",
			path: "src/domain/contest.cc",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsTestFile(tt.path)
			if got != tt.want {
				t.Fatalf("IsTestFile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsGeneratedFile(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "generated directory",
			path: "generated/camera/foo.h",
			want: true,
		},
		{
			name: "gen directory",
			path: "src/gen/foo.cc",
			want: true,
		},
		{
			name: "protobuf go",
			path: "src/proto/foo.pb.go",
			want: true,
		},
		{
			name: "protobuf cpp",
			path: "src/proto/foo.pb.cc",
			want: true,
		},
		{
			name: "grpc protobuf cpp header",
			path: "src/proto/foo.grpc.pb.h",
			want: true,
		},
		{
			name: "generated marker",
			path: "src/proto/foo.generated.cc",
			want: true,
		},
		{
			name: "production file",
			path: "src/domain/foo.h",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsGeneratedFile(tt.path)
			if got != tt.want {
				t.Fatalf("IsGeneratedFile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsIgnoredAnalysisRole(t *testing.T) {
	tests := []struct {
		role model.FileRole
		want bool
	}{
		{model.FileRoleProduction, false},
		{model.FileRoleTest, true},
		{model.FileRoleGenerated, true},
		{model.FileRoleExternal, true},
		{model.FileRoleConfig, false},
		{model.FileRoleUnknown, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			got := IsIgnoredAnalysisRole(tt.role)
			if got != tt.want {
				t.Fatalf("IsIgnoredAnalysisRole(%q) = %v, want %v", tt.role, got, tt.want)
			}
		})
	}
}

func TestIgnoredAnalysisFileSet(t *testing.T) {
	got := IgnoredAnalysisFileSet([]model.FileModel{
		{Path: "src/server/api_router.cc", Role: model.FileRoleProduction},
		{Path: "tests/api_router_test.cc", Role: model.FileRoleTest},
		{Path: "generated/foo.pb.cc", Role: model.FileRoleGenerated},
		{Path: "third_party/lib.cc", Role: model.FileRoleExternal},
	})

	if got["src/server/api_router.cc"] {
		t.Fatalf("production file must not be ignored")
	}

	for _, path := range []string{
		"tests/api_router_test.cc",
		"generated/foo.pb.cc",
		"third_party/lib.cc",
	} {
		if !got[path] {
			t.Fatalf("expected %s to be ignored in %#v", path, got)
		}
	}
}

func TestIsIgnoredAnalysisPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"src/server/api_router.cc", false},
		{"tests/api_router_test.cc", true},
		{"generated/foo.pb.cc", true},
		{"third_party/lib.cc", true},
		{"vendor/lib/lib.cc", true},
		{"src/proto/foo.grpc.pb.h", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := IsIgnoredAnalysisPath(tt.path)
			if got != tt.want {
				t.Fatalf("IsIgnoredAnalysisPath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
