package usecase

import (
	"context"

	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/reportmodel"
	reviewusecase "github.com/orurh/patchcourt/internal/usecase/review"
)

type ReviewSummary = reportmodel.ReviewSummary
type ReviewResult = reportmodel.ReviewResult
type ReviewImpactReport = reportmodel.ReviewImpactReport
type ReviewImpactItem = reportmodel.ReviewImpactItem

type ReviewFormat = reviewusecase.Format

const (
	ReviewFormatText     = reviewusecase.FormatText
	ReviewFormatJSON     = reviewusecase.FormatJSON
	ReviewFormatMarkdown = reviewusecase.FormatMarkdown
)

type ReviewRequest = reviewusecase.Request
type ReviewService = reviewusecase.Service
type ReviewProjectLoader = reviewusecase.ProjectLoader

func NewReviewProjectLoader(analysis AnalysisService) ReviewProjectLoader {
	return reviewusecase.NewProjectLoader(analysis)
}

func NewReviewService(projects ReviewProjectLoader) ReviewService {
	return reviewusecase.NewService(projects)
}

func (a *App) RunReview(ctx context.Context, req ReviewRequest) (*ReviewResult, error) {
	return a.review.Run(ctx, req)
}

func BuildReviewImpactReport(result *ReviewResult, beforeProject *model.ProjectModel, afterProject *model.ProjectModel) ReviewImpactReport {
	return reviewusecase.BuildImpactReport(result, beforeProject, afterProject)
}
