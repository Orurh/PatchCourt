package bundle

func buildChildrenByParent(commits []GitCommit) map[string][]string {
	visible := make(map[string]struct{}, len(commits))
	for _, commit := range commits {
		visible[commit.Hash] = struct{}{}
	}

	childrenByParent := make(map[string][]string)
	for _, commit := range commits {
		for _, parent := range commit.Parents {
			if _, ok := visible[parent]; !ok {
				continue
			}
			childrenByParent[parent] = append(childrenByParent[parent], commit.Hash)
		}
	}

	return childrenByParent
}
func uniqueInts(values []int) []int {
	if len(values) <= 1 {
		return values
	}

	seen := make(map[int]struct{}, len(values))
	result := make([]int, 0, len(values))

	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}

	return result
}
