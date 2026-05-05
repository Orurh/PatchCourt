package review

import (
	"testing"

	contracts "github.com/orurh/patchcourt/internal/diff/contract"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/reportmodel"
	"github.com/stretchr/testify/require"
)

func TestBuildReviewView_ClassifiesContractChangeImpact(t *testing.T) {
	result := reportmodel.ReviewResult{
		ContractChanges: []contracts.SymbolChange{
			{
				Kind:      contracts.ChangeKindSignatureChanged,
				SymbolKey: "method::ICameraAdapter::RunPreflight",
				Before: &model.SymbolModel{
					Signature: "bool RunPreflight() const;",
				},
				After: &model.SymbolModel{
					Signature: "bool RunPreflight(int cameraIndex) const;",
				},
			},
			{
				Kind:      contracts.ChangeKindAdded,
				SymbolKey: "method::ICameraAdapter::Stop",
				After: &model.SymbolModel{
					Signature: "void Stop();",
				},
			},
			{
				Kind:        contracts.ChangeKindModifiersChanged,
				SymbolKey:   "method::ICameraAdapter::Start",
				RemovedMods: []string{"const"},
			},
			{
				Kind:      contracts.ChangeKindModifiersChanged,
				SymbolKey: "method::ICameraAdapter::Capture",
				AddedMods: []string{"pure_virtual"},
			},
		},
	}

	view := BuildReviewView(result)

	require.Len(t, view.ContractRows, 4)
	require.Equal(t, "breaking", view.ContractRows[0].Impact)
	require.Equal(t, "additive", view.ContractRows[1].Impact)
	require.Equal(t, "risky", view.ContractRows[2].Impact)
	require.Equal(t, "breaking", view.ContractRows[3].Impact)
}
