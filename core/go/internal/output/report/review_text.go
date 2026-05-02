package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/orurh/patchcourt/internal/analysis/contracts"
)

func WriteReviewText(w io.Writer, changes []contracts.SymbolChange) {
	fmt.Fprintln(w, "PatchCourt review")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Contract changes:")
	fmt.Fprintf(w, "  total: %d\n", len(changes))

	for _, change := range changes {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "  [%s] %s\n", change.Kind, change.SymbolKey)

		if change.Before != nil && change.Before.Signature != "" {
			fmt.Fprintf(w, "    before: %s\n", change.Before.Signature)
		}

		if change.After != nil && change.After.Signature != "" {
			fmt.Fprintf(w, "    after:  %s\n", change.After.Signature)
		}

		if len(change.AddedMods) > 0 {
			fmt.Fprintf(w, "    added modifiers: %s\n", strings.Join(change.AddedMods, ", "))
		}

		if len(change.RemovedMods) > 0 {
			fmt.Fprintf(w, "    removed modifiers: %s\n", strings.Join(change.RemovedMods, ", "))
		}
	}
}
