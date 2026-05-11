package risk

import (
	"testing"

	contracts "github.com/orurh/patchcourt/internal/diff/contract"
	"github.com/stretchr/testify/require"
)

func TestCalculate_GroupsHardRemovedMethodsUnderRemovedParentContract(t *testing.T) {
	score := Calculate(Input{
		ContractChanges: []contracts.SymbolChange{
			{
				Kind:      contracts.ChangeKindRemoved,
				SymbolKey: "class::ICameraAdapter",
			},
			{
				Kind:      contracts.ChangeKindRemoved,
				SymbolKey: "method::ICameraAdapter::Initialize",
			},
			{
				Kind:      contracts.ChangeKindRemoved,
				SymbolKey: "method::ICameraAdapter::StartSession",
			},
			{
				Kind:      contracts.ChangeKindRemoved,
				SymbolKey: "method::IOtherContract::Run",
			},
		},
		ContractImpacts: []ContractImpact{
			{
				SymbolKey:        "class::ICameraAdapter",
				ChangeKind:       string(contracts.ChangeKindRemoved),
				TestsChanged:     true,
				DeliveryImpacted: false,
			},
			{
				SymbolKey:        "method::ICameraAdapter::Initialize",
				ChangeKind:       string(contracts.ChangeKindRemoved),
				TestsChanged:     true,
				DeliveryImpacted: false,
			},
			{
				SymbolKey:        "method::ICameraAdapter::StartSession",
				ChangeKind:       string(contracts.ChangeKindRemoved),
				TestsChanged:     true,
				DeliveryImpacted: true,
			},
			{
				SymbolKey:    "method::IOtherContract::Run",
				ChangeKind:   string(contracts.ChangeKindRemoved),
				TestsChanged: false,
				ImpactedFiles: []ContractImpactedFile{
					{File: "src/server/api_router.cc"},
				},
			},
		},
	})

	require.Equal(t, 6, score.Points)
	require.Len(t, score.Reasons, 2)

	require.Equal(t, 3, score.Reasons[0].Points)
	require.Equal(
		t,
		"contract boundary changed: class::ICameraAdapter removed with 1 delivery/API-impacted methods",
		score.Reasons[0].Message,
	)

	require.Equal(t, 3, score.Reasons[1].Points)
	require.Equal(t, "public contract symbol removed with impacted callers: method::IOtherContract::Run", score.Reasons[1].Message)
}

func TestCalculate_DoesNotScoreAmbiguousRemovedContractWithoutHardImpact(t *testing.T) {
	score := Calculate(Input{
		ContractChanges: []contracts.SymbolChange{
			{
				Kind:      contracts.ChangeKindRemoved,
				SymbolKey: "struct::CameraPreflightReport",
			},
		},
		ContractImpacts: []ContractImpact{
			{
				SymbolKey:        "struct::CameraPreflightReport",
				ChangeKind:       string(contracts.ChangeKindRemoved),
				TestsChanged:     true,
				DeliveryImpacted: false,
			},
		},
	})

	require.Equal(t, 0, score.Points)
	require.Equal(t, LevelLow, score.Level)
	require.Empty(t, score.Reasons)
}

func TestCalculate_DoesNotScoreRemovedMethodWhenParentAndMethodAreAmbiguous(t *testing.T) {
	score := Calculate(Input{
		ContractChanges: []contracts.SymbolChange{
			{
				Kind:      contracts.ChangeKindRemoved,
				SymbolKey: "class::ICameraAdapter",
			},
			{
				Kind:      contracts.ChangeKindRemoved,
				SymbolKey: "method::ICameraAdapter::Initialize",
			},
		},
		ContractImpacts: []ContractImpact{
			{
				SymbolKey:    "class::ICameraAdapter",
				ChangeKind:   string(contracts.ChangeKindRemoved),
				TestsChanged: true,
			},
			{
				SymbolKey:    "method::ICameraAdapter::Initialize",
				ChangeKind:   string(contracts.ChangeKindRemoved),
				TestsChanged: true,
			},
		},
	})

	require.Equal(t, 0, score.Points)
	require.Equal(t, LevelLow, score.Level)
	require.Empty(t, score.Reasons)
}
