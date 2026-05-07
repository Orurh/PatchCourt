package check

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/reportmodel"
)

type CheckHTMLInput struct {
	Report     reportmodel.CheckReport
	Project    *model.ProjectModel
	LayerGraph any
}

type checkHTMLPayload struct {
	Report       reportmodel.CheckReport `json:"report"`
	Root         string                  `json:"root"`
	ConfigPath   string                  `json:"config_path,omitempty"`
	OutDir       string                  `json:"out_dir"`
	Summary      model.ScanSummary       `json:"summary"`
	LayerGraph   any                     `json:"layer_graph"`
	Files        []model.FileModel       `json:"files"`
	Findings     []model.Finding         `json:"findings"`
	Dependencies []model.DependencyEdge  `json:"dependencies"`
}

func WriteCheckHTML(w io.Writer, input CheckHTMLInput) error {
	payload := checkHTMLPayload{
		Report:     input.Report,
		Root:       input.Report.Root,
		ConfigPath: input.Report.ConfigPath,
		OutDir:     input.Report.OutDir,
		Summary:    input.Report.Summary,
		LayerGraph: input.LayerGraph,
	}

	if input.Project != nil {
		payload.Files = input.Project.Files
		payload.Findings = input.Project.Findings
		payload.Dependencies = input.Project.Dependencies
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal check html payload: %w", err)
	}

	jsonPayload := strings.ReplaceAll(string(data), "</script", "<\\/script")
	page := renderCheckHTMLTemplate(jsonPayload)

	if _, err := io.WriteString(w, page); err != nil {
		return fmt.Errorf("write check html: %w", err)
	}

	return nil
}
