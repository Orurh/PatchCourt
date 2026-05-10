package usecase

import (
	"testing"

	contracts "github.com/orurh/patchcourt/internal/diff/contract"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/reportmodel"
	reviewusecase "github.com/orurh/patchcourt/internal/usecase/review"
	"github.com/stretchr/testify/require"
)

func TestBuildImpactReport_PutsAmbiguousRemovedContractIntoNeedsReview(t *testing.T) {
	result := &reviewusecase.ReviewResult{
		ContractChanges: []contracts.SymbolChange{
			{
				Kind:      contracts.ChangeKindRemoved,
				SymbolKey: "class::ICameraAdapter",
				Before: &model.SymbolModel{
					File:      "src/domain/interfaces/i_camera_adapter.h",
					Line:      20,
					Signature: "class ICameraAdapter {",
				},
			},
		},
	}

	impact := reviewusecase.BuildImpactReport(result, nil, nil)

	require.Empty(t, impact.Worse)
	require.Len(t, impact.NeedsReview, 1)
	require.Equal(t, "contract_boundary_changed", impact.NeedsReview[0].Kind)
	require.Equal(t, "class::ICameraAdapter", impact.NeedsReview[0].Detail)
	require.Contains(t, impact.NeedsReview[0].Suggestion, "abstraction cleanup")
	require.Len(t, impact.NeedsReview[0].Evidence, 1)
}

func TestBuildImpactReport_PutsContractWithImpactedCallersIntoNeedsReview(t *testing.T) {
	result := &reviewusecase.ReviewResult{
		ContractChanges: []contracts.SymbolChange{
			{
				Kind:      contracts.ChangeKindSignatureChanged,
				SymbolKey: "method::ICameraManagerController::StartSession",
				Before: &model.SymbolModel{
					File:      "src/domain/interfaces/i_camera_manager_controller.h",
					Line:      30,
					Signature: "virtual bool StartSession(SessionSettings settings) = 0;",
				},
				After: &model.SymbolModel{
					File:      "src/domain/interfaces/i_camera_manager_controller.h",
					Line:      30,
					Signature: "virtual bool StartSession(const SessionSettings& settings) = 0;",
				},
			},
		},
		ContractImpacts: []reportmodel.ContractImpact{
			{
				SymbolKey:    "method::ICameraManagerController::StartSession",
				ChangeKind:   string(contracts.ChangeKindSignatureChanged),
				TestsChanged: false,
				ImpactedFiles: []reportmodel.ContractImpactedFile{
					{File: "src/server/api_router.cc", Reason: "mentions changed symbol"},
				},
			},
		},
	}

	impact := reviewusecase.BuildImpactReport(result, nil, nil)

	require.Empty(t, impact.Worse)
	require.Len(t, impact.NeedsReview, 1)
	require.Equal(t, "contract_delivery_impact", impact.NeedsReview[0].Kind)
	require.Contains(t, impact.NeedsReview[0].Detail, "method::ICameraManagerController::StartSession")
}
