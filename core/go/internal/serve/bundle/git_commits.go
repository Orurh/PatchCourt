package bundle

import (
	"context"
	"fmt"
	"strings"
)

type GitCommitQuery struct {
	Ref   string
	All   bool
	Limit int
}

func readGitCommits(ctx context.Context, root string, query GitCommitQuery) ([]GitCommit, error) {
	if query.Limit <= 0 {
		query.Limit = 50
	}

	format := "%H%x1f%h%x1f%P%x1f%D%x1f%an%x1f%aI%x1f%s"
	args := []string{
		"log",
		fmt.Sprintf("-n%d", query.Limit),
		"--date=iso-strict",
		"--format=" + format,
	}

	if query.All {
		args = append(args, "--all")
	} else if strings.TrimSpace(query.Ref) != "" {
		args = append(args, query.Ref)
	}

	out, err := runGitTrimmed(ctx, root, args...)
	if err != nil {
		return nil, err
	}

	if out == "" {
		return nil, nil
	}

	lines := strings.Split(out, "\n")
	commits := make([]GitCommit, 0, len(lines))

	for _, line := range lines {
		parts := strings.Split(line, "\x1f")
		if len(parts) != 7 {
			continue
		}

		commits = append(commits, GitCommit{
			Hash:      parts[0],
			ShortHash: parts[1],
			Parents:   strings.Fields(parts[2]),
			Refs:      parseDecorations(parts[3]),
			Author:    parts[4],
			Date:      parts[5],
			Message:   parts[6],
		})
	}

	return commits, nil
}

func parseDecorations(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	raw := strings.Split(value, ",")
	refs := make([]string, 0, len(raw))

	for _, item := range raw {
		item = strings.TrimSpace(item)
		item = strings.TrimPrefix(item, "HEAD -> ")
		if item == "" {
			continue
		}
		refs = append(refs, item)
	}

	return refs
}
