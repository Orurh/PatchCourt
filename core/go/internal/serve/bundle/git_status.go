package bundle

import (
	"context"
	"strings"
)

func readGitStatus(ctx context.Context, root string) (GitStatus, error) {
	branch, err := runGitTrimmed(ctx, root, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return GitStatus{}, err
	}

	head, err := runGitTrimmed(ctx, root, "rev-parse", "HEAD")
	if err != nil {
		return GitStatus{}, err
	}

	shortHead, err := runGitTrimmed(ctx, root, "rev-parse", "--short", "HEAD")
	if err != nil {
		return GitStatus{}, err
	}

	statusOut, err := runGitTrimmed(ctx, root, "status", "--porcelain")
	if err != nil {
		return GitStatus{}, err
	}

	status := GitStatus{
		Root:               root,
		Branch:             branch,
		Head:               head,
		ShortHead:          shortHead,
		HasWorktreeChanges: strings.TrimSpace(statusOut) != "",
	}

	upstream, err := runGitTrimmed(ctx, root, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	if err == nil && upstream != "" {
		status.Upstream = upstream

		counts, err := runGitTrimmed(ctx, root, "rev-list", "--left-right", "--count", upstream+"...HEAD")
		if err == nil {
			behind, ahead := parseAheadBehind(counts)
			status.Behind = behind
			status.Ahead = ahead
		}
	}

	return status, nil
}
