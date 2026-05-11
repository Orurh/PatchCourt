package bundle

import (
	"context"
	"strings"
)

func readGitBranches(ctx context.Context, root string) (GitBranchesResponse, error) {
	current, err := runGitTrimmed(ctx, root, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return GitBranchesResponse{}, err
	}

	format := strings.Join([]string{
		"%(refname:short)",
		"%(objectname)",
		"%(objectname:short)",
		"%(upstream:short)",
	}, gitRefFieldSeparator)

	out, err := runGitTrimmed(ctx, root, "branch", "--all", "--format="+format)
	if err != nil {
		return GitBranchesResponse{}, err
	}

	branches := make([]GitBranch, 0)
	seen := make(map[string]struct{})

	for _, line := range splitNonEmptyLines(out) {
		parts := strings.Split(line, gitRefFieldSeparator)
		if len(parts) != 4 {
			continue
		}

		name := strings.TrimSpace(parts[0])
		if name == "" || strings.HasSuffix(name, "/HEAD") || strings.Contains(name, "HEAD detached") {
			continue
		}

		kind := "local"
		if strings.HasPrefix(name, "remotes/") {
			name = strings.TrimPrefix(name, "remotes/")
			kind = "remote"
		}
		if strings.HasPrefix(name, "origin/") {
			kind = "remote"
		}

		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}

		branch := GitBranch{
			Name:      name,
			Kind:      kind,
			Head:      parts[1],
			ShortHead: parts[2],
			Current:   name == current,
			Upstream:  parts[3],
		}

		if branch.Upstream != "" && branch.Kind == "local" {
			counts, err := runGitTrimmed(ctx, root, "rev-list", "--left-right", "--count", branch.Upstream+"..."+branch.Name)
			if err == nil {
				branch.Behind, branch.Ahead = parseAheadBehind(counts)
			}
		}

		branches = append(branches, branch)
	}

	return GitBranchesResponse{
		Root:     root,
		Current:  current,
		Branches: branches,
	}, nil
}
