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
		NeedsReview:   make([]ReviewImpactItem, 0),
		UnchangedDebt: make([]ReviewImpactItem, 0),
	}

	afterPolicyIndex := buildPolicyViolationEdgeIndex(afterProject)
	beforePolicyIndex := buildPolicyViolationEdgeIndex(beforeProject)

	report.Worse = append(report.Worse, worseFindingChanges(result.FindingChanges)...)
	report.Worse = append(report.Worse, worseLayerEdgeChanges(result.LayerEdgeChanges, afterPolicyIndex)...)
	report.Worse = append(report.Worse, worseDependencyChanges(result.DependencyChanges, afterPolicyIndex)...)
	report.Worse = append(report.Worse, worseContractChanges(result.ContractChanges, result.ContractImpacts)...)

	report.NeedsReview = append(report.NeedsReview, needsReviewContractChanges(result.ContractChanges, result.ContractImpacts)...)
	report.NeedsReview = append(report.NeedsReview, needsReviewFindingChanges(result.FindingChanges)...)
	report.NeedsReview = append(report.NeedsReview, needsReviewDependencyChanges(result.DependencyChanges, afterPolicyIndex, beforePolicyIndex)...)
	report.NeedsReview = append(report.NeedsReview, needsReviewLayerEdgeChanges(result.LayerEdgeChanges, afterPolicyIndex, beforePolicyIndex)...)

	report.Better = append(report.Better, betterFindingChanges(result.FindingChanges)...)
	report.Better = append(report.Better, betterLayerEdgeChanges(result.LayerEdgeChanges, beforePolicyIndex)...)
	report.Better = append(report.Better, betterDependencyChanges(result.DependencyChanges, beforePolicyIndex)...)

	report.UnchangedDebt = unchangedDebtImpact(beforeProject, afterProject, result.FindingChanges)

	sortImpactItems(report.Worse)
	sortImpactItems(report.Better)
	sortImpactItems(report.NeedsReview)
	sortImpactItems(report.UnchangedDebt)

	return report
}
