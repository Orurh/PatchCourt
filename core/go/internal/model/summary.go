package model

type ScanSummary struct {
	CPPHeaders int
	CPPSources int
	CPPTests   int
	GoFiles    int
	Symbols    int

	ProductionFiles int
	TestFiles       int
	GeneratedFiles  int
	ExternalFiles   int
	ConfigFiles     int
	UnknownFiles    int

	TotalEdges int
	Resolved   int
	Unresolved int
	External   int

	UsageUsed    int
	UsageUnused  int
	UsageMaybe   int
	UsageUnknown int
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
