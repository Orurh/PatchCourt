package bundle

func buildGitGraphLayout(commits []GitGraphCommit) GitGraphLayout {
	rowByHash := make(map[string]int, len(commits))
	commitByHash := make(map[string]GitGraphCommit, len(commits))

	for index, commit := range commits {
		rowByHash[commit.Hash] = index
		commitByHash[commit.Hash] = commit
	}

	segments := make([]GitGraphSegment, 0, len(commits))
	edges := make([]GitGraphEdge, 0)

	for row, commit := range commits {
		for parentIndex, parentHash := range commit.Parents {
			parentRow, ok := rowByHash[parentHash]
			if !ok {
				continue
			}

			parentCommit := commitByHash[parentHash]
			kind := gitGraphEdgeKind(commit, parentIndex)

			if parentCommit.Lane == commit.Lane {
				segments = append(segments, GitGraphSegment{
					Lane:    commit.Lane,
					FromRow: row,
					ToRow:   parentRow,
				})
				continue
			}

			edges = append(edges, GitGraphEdge{
				FromLane: commit.Lane,
				ToLane:   parentCommit.Lane,
				FromRow:  row,
				ToRow:    parentRow,
				Kind:     kind,
			})
		}
	}

	return GitGraphLayout{
		RowHeight: 54,
		LaneWidth: 12,
		Segments:  mergeGitGraphSegments(segments),
		Edges:     edges,
	}
}
func gitGraphEdgeKind(commit GitGraphCommit, parentIndex int) string {
	if len(commit.Parents) > 1 && parentIndex > 0 {
		return "merge"
	}

	if len(commit.Parents) > 1 {
		return "parent"
	}

	return "branch"
}
func mergeGitGraphSegments(segments []GitGraphSegment) []GitGraphSegment {
	if len(segments) == 0 {
		return nil
	}

	merged := make([]GitGraphSegment, 0, len(segments))

	for _, segment := range segments {
		if segment.FromRow == segment.ToRow {
			continue
		}

		if segment.FromRow > segment.ToRow {
			segment.FromRow, segment.ToRow = segment.ToRow, segment.FromRow
		}

		found := false
		for index := range merged {
			existing := &merged[index]
			if existing.Lane != segment.Lane {
				continue
			}

			if segment.FromRow <= existing.ToRow+1 && segment.ToRow >= existing.FromRow-1 {
				if segment.FromRow < existing.FromRow {
					existing.FromRow = segment.FromRow
				}
				if segment.ToRow > existing.ToRow {
					existing.ToRow = segment.ToRow
				}
				found = true
				break
			}
		}

		if !found {
			merged = append(merged, segment)
		}
	}

	return merged
}
