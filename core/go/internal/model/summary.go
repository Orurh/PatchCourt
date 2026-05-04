package model

type ScanSummary struct {
	CPPHeaders int `json:"cpp_headers"`
	CPPSources int `json:"cpp_sources"`
	CPPTests   int `json:"cpp_tests"`
	GoFiles    int `json:"go_files"`
	Symbols    int `json:"symbols"`

	ProductionFiles int `json:"production_files"`
	TestFiles       int `json:"test_files"`
	GeneratedFiles  int `json:"generated_files"`
	ExternalFiles   int `json:"external_files"`
	ConfigFiles     int `json:"config_files"`
	UnknownFiles    int `json:"unknown_files"`

	TotalEdges int `json:"total_edges"`
	Resolved   int `json:"resolved"`
	Unresolved int `json:"unresolved"`
	External   int `json:"external"`

	UsageUsed    int `json:"usage_used"`
	UsageUnused  int `json:"usage_unused"`
	UsageMaybe   int `json:"usage_maybe"`
	UsageUnknown int `json:"usage_unknown"`
}

func BuildScanSummary(project *ProjectModel) ScanSummary {
	var summary ScanSummary
	summary.Symbols = len(project.Symbols)

	for _, file := range project.Files {
		switch file.Role {
		case FileRoleProduction:
			summary.ProductionFiles++
		case FileRoleTest:
			summary.TestFiles++
		case FileRoleGenerated:
			summary.GeneratedFiles++
		case FileRoleExternal:
			summary.ExternalFiles++
		case FileRoleConfig:
			summary.ConfigFiles++
		default:
			summary.UnknownFiles++
		}

		switch file.Language {
		case LanguageCPP:
			if file.Role == FileRoleTest {
				summary.CPPTests++
				continue
			}

			switch file.Kind {
			case FileKindHeader:
				summary.CPPHeaders++
			case FileKindSource:
				summary.CPPSources++
			}
		case LanguageGo:
			summary.GoFiles++
		}
	}

	for _, dep := range project.Dependencies {
		summary.TotalEdges++

		switch dep.Usage {
		case DependencyUsageUsed:
			summary.UsageUsed++
		case DependencyUsageUnused:
			summary.UsageUnused++
		case DependencyUsageMaybe:
			summary.UsageMaybe++
		default:
			summary.UsageUnknown++
		}

		if dep.External {
			summary.External++
			continue
		}

		if dep.Resolved {
			summary.Resolved++
		} else {
			summary.Unresolved++
		}
	}

	return summary
}
