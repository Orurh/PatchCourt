package changes

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/orurh/patchcourt/internal/model"
)

const (
	DefaultStateDirName = ".patchcourt/state"
	LatestStateName     = "latest"

	projectModelFileName = "project-model.json"
	metadataFileName     = "metadata.json"
)

type StateMetadata struct {
	SchemaVersion int       `json:"schema_version"`
	CreatedAt     time.Time `json:"created_at"`
	Root          string    `json:"root"`
	ConfigPath    string    `json:"config_path,omitempty"`
	Files         int       `json:"files"`
	Dependencies  int       `json:"dependencies"`
	Findings      int       `json:"findings"`
}

type SaveStateOptions struct {
	Root       string
	ConfigPath string
	Name       string
	Project    *model.ProjectModel
}

type LoadStateOptions struct {
	Root string
	Name string
}

type LoadedState struct {
	Project  *model.ProjectModel `json:"project"`
	Metadata StateMetadata       `json:"metadata"`
	Path     string              `json:"path"`
}

func SaveState(opts SaveStateOptions) (StateMetadata, error) {
	if opts.Project == nil {
		return StateMetadata{}, fmt.Errorf("project model is required")
	}

	root := opts.Root
	if root == "" {
		root = opts.Project.Root
	}
	if root == "" {
		root = "."
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return StateMetadata{}, fmt.Errorf("resolve state root: %w", err)
	}

	name := opts.Name
	if name == "" {
		name = LatestStateName
	}

	dir := StateDir(absRoot, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return StateMetadata{}, fmt.Errorf("create state dir %s: %w", dir, err)
	}

	projectPath := filepath.Join(dir, projectModelFileName)
	if err := writeJSONFile(projectPath, opts.Project); err != nil {
		return StateMetadata{}, err
	}

	metadata := StateMetadata{
		SchemaVersion: 1,
		CreatedAt:     time.Now().UTC(),
		Root:          absRoot,
		ConfigPath:    opts.ConfigPath,
		Files:         len(opts.Project.Files),
		Dependencies:  len(opts.Project.Dependencies),
		Findings:      len(opts.Project.Findings),
	}

	metadataPath := filepath.Join(dir, metadataFileName)
	if err := writeJSONFile(metadataPath, metadata); err != nil {
		return StateMetadata{}, err
	}

	return metadata, nil
}

func LoadState(opts LoadStateOptions) (*LoadedState, error) {
	root := opts.Root
	if root == "" {
		root = "."
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve state root: %w", err)
	}

	name := opts.Name
	if name == "" {
		name = LatestStateName
	}

	dir := StateDir(absRoot, name)

	projectPath := filepath.Join(dir, projectModelFileName)
	project, err := ReadProjectModel(projectPath)
	if err != nil {
		return nil, fmt.Errorf("read state project model: %w", err)
	}

	var metadata StateMetadata
	metadataPath := filepath.Join(dir, metadataFileName)
	if err := readJSONFile(metadataPath, &metadata); err != nil {
		return nil, fmt.Errorf("read state metadata: %w", err)
	}

	return &LoadedState{
		Project:  project,
		Metadata: metadata,
		Path:     projectPath,
	}, nil
}

func StateDir(root string, name string) string {
	if name == "" {
		name = LatestStateName
	}

	return filepath.Join(root, DefaultStateDirName, name)
}

func ReadProjectModel(path string) (*model.ProjectModel, error) {
	var project model.ProjectModel
	if err := readJSONFile(path, &project); err != nil {
		return nil, err
	}

	return &project, nil
}

func writeJSONFile(path string, value any) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	writeErr := encoder.Encode(value)
	closeErr := file.Close()

	if writeErr != nil {
		return fmt.Errorf("write %s: %w", path, writeErr)
	}

	if closeErr != nil {
		return fmt.Errorf("close %s: %w", path, closeErr)
	}

	return nil
}

func readJSONFile(path string, value any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	if err := json.Unmarshal(data, value); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}

	return nil
}
