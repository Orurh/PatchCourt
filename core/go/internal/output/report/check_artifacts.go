package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"

	graphmodel "github.com/orurh/patchcourt/internal/analysis/graph"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/platform/files"
	"github.com/orurh/patchcourt/internal/reportmodel"
)

type CheckArtifact struct {
	Name string
	Path string
}

type CheckArtifactsInput struct {
	OutDir     string
	Project    *model.ProjectModel
	LayerGraph graphmodel.LayerGraph
	Report     reportmodel.CheckReport
}

func WriteCheckArtifacts(input CheckArtifactsInput) ([]CheckArtifact, error) {
	if input.OutDir == "" {
		return nil, fmt.Errorf("check output dir is required")
	}

	artifacts := make([]CheckArtifact, 0, 6)

	writeArtifact := func(name string, filename string, render func() ([]byte, error)) error {
		path := filepath.Join(input.OutDir, filename)

		data, err := render()
		if err != nil {
			return fmt.Errorf("render artifact %s: %w", path, err)
		}

		if err := files.WriteFileAtomic(path, data, 0o644); err != nil {
			return fmt.Errorf("write artifact %s: %w", path, err)
		}

		artifacts = append(artifacts, CheckArtifact{
			Name: name,
			Path: path,
		})
		return nil
	}

	if err := writeArtifact("project model", "project-model.json", func() ([]byte, error) {
		return encodeJSON(input.Project)
	}); err != nil {
		return nil, err
	}

	if err := writeArtifact("scan report", "scan.md", func() ([]byte, error) {
		var buf bytes.Buffer
		WriteScanMarkdown(&buf, input.Project)
		return buf.Bytes(), nil
	}); err != nil {
		return nil, err
	}

	if err := writeArtifact("layer graph json", "layer-graph.json", func() ([]byte, error) {
		return encodeJSON(input.LayerGraph)
	}); err != nil {
		return nil, err
	}

	if err := writeArtifact("layer graph dot", "layer-graph.dot", func() ([]byte, error) {
		var buf bytes.Buffer
		WriteLayerGraphDOT(&buf, input.LayerGraph)
		return buf.Bytes(), nil
	}); err != nil {
		return nil, err
	}

	if err := writeArtifact("layer graph mermaid", "layer-graph.mmd", func() ([]byte, error) {
		var buf bytes.Buffer
		WriteLayerGraphMermaid(&buf, input.LayerGraph)
		return buf.Bytes(), nil
	}); err != nil {
		return nil, err
	}

	if err := writeArtifact("html report", "report.html", func() ([]byte, error) {
		var buf bytes.Buffer

		if err := WriteCheckHTML(&buf, CheckHTMLInput{
			Report:     input.Report,
			Project:    input.Project,
			LayerGraph: input.LayerGraph,
		}); err != nil {
			return nil, err
		}

		return buf.Bytes(), nil
	}); err != nil {
		return nil, err
	}

	return artifacts, nil
}

func encodeJSON(value any) ([]byte, error) {
	var buf bytes.Buffer

	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(value); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
