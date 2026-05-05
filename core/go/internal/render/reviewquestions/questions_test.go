package reviewquestions

import (
	"testing"

	contracts "github.com/orurh/patchcourt/internal/diff/contract"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/reportmodel"
	"github.com/stretchr/testify/require"
)

func TestBuild_ReturnsDefaultQuestionWhenNoSignals(t *testing.T) {
	got := Build(reportmodel.ReviewResult{}, 10)

	require.Len(t, got, 1)
	require.Contains(t, got[0].Text, "No specific high-signal questions")
}

func TestBuild_AsksAboutWorseImpact(t *testing.T) {
	got := Build(reportmodel.ReviewResult{
		Impact: reportmodel.ReviewImpactReport{
			Worse: []reportmodel.ReviewImpactItem{
				{
					Title:  "Added forbidden dependency",
					ID:     "architecture.api.cameras",
					Detail: "api -> cameras",
				},
			},
		},
	}, 10)

	require.Len(t, got, 1)
	require.Contains(t, got[0].Text, "Check whether this regression is intentional")
	require.Contains(t, got[0].Text, "architecture.api.cameras")
	require.Contains(t, got[0].Text, "api -> cameras")
}

func TestBuild_AsksAboutContractChangeWithoutTests(t *testing.T) {
	got := Build(reportmodel.ReviewResult{
		ContractChanges: []contracts.SymbolChange{
			{
				Kind:      contracts.ChangeKindSignatureChanged,
				SymbolKey: "method::ICameraAdapter::RunPreflight",
				After: &model.SymbolModel{
					File: "src/domain/interfaces/i_camera_adapter.h",
				},
			},
		},
		ChangedFiles: []string{"src/domain/interfaces/i_camera_adapter.h"},
	}, 10)

	require.Len(t, got, 1)
	require.Contains(t, got[0].Text, "but no test-like files changed")
}

func TestBuild_AsksToVerifyChangedTests(t *testing.T) {
	got := Build(reportmodel.ReviewResult{
		ContractChanges: []contracts.SymbolChange{
			{
				Kind:      contracts.ChangeKindSignatureChanged,
				SymbolKey: "method::ICameraAdapter::RunPreflight",
				After: &model.SymbolModel{
					File: "src/domain/interfaces/i_camera_adapter.h",
				},
			},
		},
		ChangedFiles: []string{
			"src/domain/interfaces/i_camera_adapter.h",
			"tests/i_camera_adapter_test.cc",
		},
	}, 10)

	require.Len(t, got, 1)
	require.Contains(t, got[0].Text, "test-like files changed")
}

func TestBuild_DeduplicatesContractQuestionsBySymbolKey(t *testing.T) {
	got := Build(reportmodel.ReviewResult{
		ContractChanges: []contracts.SymbolChange{
			{
				Kind:      contracts.ChangeKindSignatureChanged,
				SymbolKey: "method::ICameraAdapter::StartSession",
			},
			{
				Kind:      contracts.ChangeKindModifiersChanged,
				SymbolKey: "method::ICameraAdapter::StartSession",
			},
		},
	}, 10)

	require.Len(t, got, 1)
	require.Contains(t, got[0].Text, "method::ICameraAdapter::StartSession")
}
