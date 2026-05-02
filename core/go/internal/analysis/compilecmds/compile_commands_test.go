package compilecmds

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIncludePaths_ExtractsArgumentsIncludePaths(t *testing.T) {
	root := t.TempDir()
	buildDir := filepath.Join(root, "build")

	db := &Database{
		Entries: []Entry{
			{
				Directory: buildDir,
				File:      filepath.Join(root, "src/main.cc"),
				Arguments: []string{
					"clang++",
					"-I",
					"../include",
					"-Isrc",
					"-isystem",
					"../third_party/lib/include",
					"-iquote=../quoted",
					"--include-directory=../generated",
					"-c",
					"../src/main.cc",
				},
			},
		},
	}

	got := IncludePaths(db, root)

	assertStringSliceContains(t, got, "include")
	assertStringSliceContains(t, got, "build/src")
	assertStringSliceContains(t, got, "third_party/lib/include")
	assertStringSliceContains(t, got, "quoted")
	assertStringSliceContains(t, got, "generated")
}

func TestIncludePaths_ExtractsCommandIncludePaths(t *testing.T) {
	root := t.TempDir()

	db := &Database{
		Entries: []Entry{
			{
				Directory: root,
				File:      filepath.Join(root, "src/main.cc"),
				Command:   `clang++ -I include -isystem=third_party/include -iquote "quoted includes" --include-directory generated -c src/main.cc`,
			},
		},
	}

	got := IncludePaths(db, root)

	assertStringSliceContains(t, got, "include")
	assertStringSliceContains(t, got, "third_party/include")
	assertStringSliceContains(t, got, "quoted includes")
	assertStringSliceContains(t, got, "generated")
}

func TestIncludePaths_ParsesCommandWithShellEscaping(t *testing.T) {
	root := t.TempDir()

	db := &Database{
		Entries: []Entry{
			{
				Directory: root,
				File:      filepath.Join(root, "src/main.cc"),
				Command:   `clang++ -I "include path" -isystem 'third party/include' -c src/main.cc`,
			},
		},
	}

	got := IncludePaths(db, root)

	assertStringSliceContains(t, got, "include path")
	assertStringSliceContains(t, got, "third party/include")
}

func TestLoad_ParseCompileCommands(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "compile_commands.json")

	content := `[
  {
    "directory": "/repo/build",
    "command": "clang++ -I../include -c ../src/main.cc",
    "file": "/repo/src/main.cc"
  }
]`

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write compile_commands.json: %v", err)
	}

	db, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(db.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(db.Entries))
	}

	if db.Entries[0].Command == "" {
		t.Fatalf("expected command to be parsed")
	}
}

func assertStringSliceContains(t *testing.T, values []string, expected string) {
	t.Helper()

	for _, value := range values {
		if value == expected {
			return
		}
	}

	t.Fatalf("expected %q in %#v", expected, values)
}
