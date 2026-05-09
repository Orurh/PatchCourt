package review

import (
	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/reportmodel"
)

type ReviewImpactReport = reportmodel.ReviewImpactReport
type ReviewImpactItem = reportmodel.ReviewImpactItem

func BuildImpactReport(result *ReviewResult, beforeProject *model.ProjectModel, afterProject *model.ProjectModel) ReviewImpactReport {
	if result == nil {
		return ReviewImpactReport{}
	}

	report := ReviewImpactReport{
		Worse:         make([]ReviewImpactItem, 0),
		Better:        make([]ReviewImpactItem, 0),
		UnchangedDebt: make([]ReviewImpactItem, 0),
	}

	policyIndex := buildPolicyViolationEdgeIndex(afterProject)

	report.Worse = append(report.Worse, worseFindingChanges(result.FindingChanges)...)
	report.Worse = append(report.Worse, worseLayerEdgeChanges(result.LayerEdgeChanges, policyIndex)...)
	report.Worse = append(report.Worse, worseDependencyChanges(result.DependencyChanges, policyIndex)...)
	report.Worse = append(report.Worse, worseContractChanges(result.ContractChanges, result.ContractImpacts)...)

	report.Better = append(report.Better, betterFindingChanges(result.FindingChanges)...)
	report.Better = append(report.Better, betterLayerEdgeChanges(result.LayerEdgeChanges)...)
	report.Better = append(report.Better, betterDependencyChanges(result.DependencyChanges)...)

	report.UnchangedDebt = unchangedDebtImpact(beforeProject, afterProject, result.FindingChanges)

	sortImpactItems(report.Worse)
	sortImpactItems(report.Better)
	sortImpactItems(report.UnchangedDebt)

	return report
}
