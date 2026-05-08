package reviewbundle

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/orurh/patchcourt/internal/platform/files"
	"github.com/orurh/patchcourt/internal/render/llmpack"
	renderreview "github.com/orurh/patchcourt/internal/render/review"
	rendersarif "github.com/orurh/patchcourt/internal/render/sarif"
	"github.com/orurh/patchcourt/internal/reportmodel"
)

const SchemaVersion = "patchcourt.review_bundle.v1"

type Options struct {
	Dir      string
	MaxItems int
}

type Manifest struct {
	SchemaVersion string            `json:"schema_version"`
	Artifacts     map[string]string `json:"artifacts"`
}

func Write(dir string, result reportmodel.ReviewResult) error {
	return WriteWithOptions(Options{
		Dir:      dir,
		MaxItems: 10,
	}, result)
}

func WriteWithOptions(opts Options, result reportmodel.ReviewResult) error {
	if opts.Dir == "" {
		return fmt.Errorf("review bundle output directory is required")
	}

	if opts.MaxItems <= 0 {
		opts.MaxItems = 10
	}

	if err := os.MkdirAll(opts.Dir, 0o755); err != nil {
		return fmt.Errorf("create review bundle directory %s: %w", opts.Dir, err)
	}

	manifest := Manifest{
		SchemaVersion: SchemaVersion,
		Artifacts: map[string]string{
			"review":      "review.json",
			"html":        "review.html",
			"llm_context": "review-context.md",
			"sarif":       "patchcourt.sarif",
		},
	}

	if err := writeJSON(filepath.Join(opts.Dir, "manifest.json"), manifest); err != nil {
		return err
	}

	if err := writeJSON(filepath.Join(opts.Dir, "review.json"), result); err != nil {
		return err
	}

	if err := writeHTML(filepath.Join(opts.Dir, "review.html"), result); err != nil {
		return err
	}

	if err := llmpack.WriteReviewContextFile(filepath.Join(opts.Dir, "review-context.md"), llmpack.ReviewContextInput{
		Result:   result,
		MaxItems: opts.MaxItems,
	}); err != nil {
		return err
	}

	if err := rendersarif.WriteReviewSARIFFile(filepath.Join(opts.Dir, "patchcourt.sarif"), result); err != nil {
		return err
	}

	return nil
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", filepath.Base(path), err)
	}

	data = append(data, '\n')

	if err := files.WriteFileAtomic(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	return nil
}

func writeHTML(path string, result reportmodel.ReviewResult) error {
	var buf bytes.Buffer

	if err := renderreview.WriteReviewHTML(&buf, result); err != nil {
		return err
	}

	if err := files.WriteFileAtomic(path, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write review HTML %s: %w", path, err)
	}

	return nil
}
