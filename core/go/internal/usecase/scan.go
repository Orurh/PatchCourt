package usecase

import (
	"context"

	scanusecase "github.com/orurh/patchcourt/internal/usecase/scan"
)

type ScanFormat = scanusecase.Format

const (
	ScanFormatText     = scanusecase.FormatText
	ScanFormatJSON     = scanusecase.FormatJSON
	ScanFormatMarkdown = scanusecase.FormatMarkdown
)

type ScanRequest = scanusecase.Request
type ScanResult = scanusecase.Result
type ScanService = scanusecase.Service

func NewScanService(projects ProjectBuilder) ScanService {
	return scanusecase.NewService(projects)
}

func (a *App) RunScan(ctx context.Context, req ScanRequest) (*ScanResult, error) {
	return a.scan.Run(ctx, req)
}
