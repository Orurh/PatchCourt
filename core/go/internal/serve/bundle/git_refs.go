package bundle

import (
	"context"
	"strings"
)

func readGitRefs(ctx context.Context, root string) (GitRefsResponse, error) {
	format := strings.Join([]string{
		"%(refname)",
		"%(refname:short)",
		"%(objectname)",
		"%(objectname:short)",
	}, gitRefFieldSeparator)

	out, err := runGitTrimmed(ctx, root, "for-each-ref",
		"--format="+format,
		"refs/heads",
		"refs/remotes",
		"refs/tags",
	)
	if err != nil {
		return GitRefsResponse{}, err
	}

	refs := make([]GitRef, 0)
	for _, line := range splitNonEmptyLines(out) {
		parts := strings.Split(line, gitRefFieldSeparator)
		if len(parts) != 4 {
			continue
		}

		fullName := parts[0]
		shortName := parts[1]

		if strings.HasSuffix(fullName, "/HEAD") {
			continue
		}

		kind := "other"
		switch {
		case strings.HasPrefix(fullName, "refs/heads/"):
			kind = "branch"
		case strings.HasPrefix(fullName, "refs/remotes/"):
			kind = "remote"
		case strings.HasPrefix(fullName, "refs/tags/"):
			kind = "tag"
		}

		refs = append(refs, GitRef{
			Name:      shortName,
			Kind:      kind,
			Target:    parts[2],
			ShortHash: parts[3],
		})
	}

	return GitRefsResponse{
		Root: root,
		Refs: refs,
	}, nil
}
