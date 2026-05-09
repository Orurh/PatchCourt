package pathmatch

import "testing"

func TestMatch_ExactPath(t *testing.T) {
	if !Match("src/server/api_router.cc", "src/server/api_router.cc") {
		t.Fatalf("expected exact path to match")
	}
}

func TestMatch_DoublestarDirectory(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		path    string
		want    bool
	}{
		{
			name:    "direct child",
			pattern: "src/server/**",
			path:    "src/server/api_router.cc",
			want:    true,
		},
		{
			name:    "nested child",
			pattern: "src/server/**",
			path:    "src/server/http/api_router.cc",
			want:    true,
		},
		{
			name:    "different directory",
			pattern: "src/server/**",
			path:    "src/controllers/device_orchestrator.h",
			want:    false,
		},
		{
			name:    "directory itself",
			pattern: "src/server/**",
			path:    "src/server",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Match(tt.pattern, tt.path)
			if got != tt.want {
				t.Fatalf("Match(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
			}
		})
	}
}

func TestMatch_DoublestarFileSuffix(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		path    string
		want    bool
	}{
		{
			name:    "protobuf header",
			pattern: "**/*.pb.h",
			path:    "generated/camera/foo.pb.h",
			want:    true,
		},
		{
			name:    "protobuf source",
			pattern: "**/*.pb.cc",
			path:    "generated/camera/foo.pb.cc",
			want:    true,
		},
		{
			name:    "not protobuf",
			pattern: "**/*.pb.h",
			path:    "src/domain/foo.h",
			want:    false,
		},
		{
			name:    "generated marker",
			pattern: "**/*.generated.*",
			path:    "src/proto/foo.generated.cc",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Match(tt.pattern, tt.path)
			if got != tt.want {
				t.Fatalf("Match(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
			}
		})
	}
}

func TestMatch_NormalizesBackslashes(t *testing.T) {
	if !Match("src/server/**", `src\server\api_router.cc`) {
		t.Fatalf("expected path with backslashes to match")
	}
}

func TestMatch_EmptyPattern(t *testing.T) {
	if Match("", "src/server/api_router.cc") {
		t.Fatalf("empty pattern must not match")
	}
}

func TestMatchAny(t *testing.T) {
	patterns := []string{
		"build/**",
		"third_party/**",
		"**/*.pb.h",
	}

	if !MatchAny(patterns, "third_party/somelib/include/lib.h") {
		t.Fatalf("expected path to match one of patterns")
	}

	if !MatchAny(patterns, "generated/foo.pb.h") {
		t.Fatalf("expected protobuf header to match one of patterns")
	}

	if MatchAny(patterns, "src/server/api_router.cc") {
		t.Fatalf("did not expect source file to match ignore patterns")
	}
}

func TestIsTestLikeFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"src/server/api_router.cc", false},
		{"tests/api_router_test.cc", true},
		{"test/unit/camera_manager_controller_test.cc", true},
		{"src/domain/foo_spec.cpp", true},
		{"src/domain/test_camera.cc", true},
		{"src/mocks/mock_camera.h", true},
		{`src\mocks\mock_camera.h`, true},
		{"src/domain/i_camera_adapter.h", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := IsTestLikeFile(tt.path); got != tt.want {
				t.Fatalf("IsTestLikeFile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
