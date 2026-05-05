package reviewcontract

import (
	"testing"

	contracts "github.com/orurh/patchcourt/internal/diff/contract"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/stretchr/testify/require"
)

func TestClassifyImpact(t *testing.T) {
	tests := []struct {
		name   string
		change contracts.SymbolChange
		want   Impact
	}{
		{
			name: "removed is breaking",
			change: contracts.SymbolChange{
				Kind: contracts.ChangeKindRemoved,
			},
			want: ImpactBreaking,
		},
		{
			name: "signature changed is breaking",
			change: contracts.SymbolChange{
				Kind: contracts.ChangeKindSignatureChanged,
			},
			want: ImpactBreaking,
		},
		{
			name: "added is additive",
			change: contracts.SymbolChange{
				Kind: contracts.ChangeKindAdded,
			},
			want: ImpactAdditive,
		},
		{
			name: "added pure virtual is breaking",
			change: contracts.SymbolChange{
				Kind:      contracts.ChangeKindModifiersChanged,
				AddedMods: []string{"pure_virtual"},
			},
			want: ImpactBreaking,
		},
		{
			name: "removed const is risky",
			change: contracts.SymbolChange{
				Kind:        contracts.ChangeKindModifiersChanged,
				RemovedMods: []string{"const"},
			},
			want: ImpactRisky,
		},
		{
			name: "unknown modifier is informational",
			change: contracts.SymbolChange{
				Kind:      contracts.ChangeKindModifiersChanged,
				AddedMods: []string{"custom"},
			},
			want: ImpactInformational,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, ClassifyImpact(tt.change))
		})
	}
}

func TestLocation(t *testing.T) {
	got := Location(contracts.SymbolChange{
		Before: &model.SymbolModel{
			File: "src/domain/interfaces/i_camera_adapter.h",
			Line: 12,
		},
		After: &model.SymbolModel{
			File: "src/domain/interfaces/i_camera_adapter.h",
			Line: 14,
		},
	})

	require.Equal(t, "src/domain/interfaces/i_camera_adapter.h:12 → 14", got)
}

func TestModifiers(t *testing.T) {
	got := Modifiers(contracts.SymbolChange{
		AddedMods:   []string{"override"},
		RemovedMods: []string{"const", "virtual"},
	})

	require.Equal(t, "added: override; removed: const, virtual", got)
}
