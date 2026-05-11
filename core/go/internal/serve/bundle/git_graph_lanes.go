package bundle

import "strings"

func assignGitGraphLanes(commits []GitCommit) map[string]int {
	preferredLaneByHash := preferredGitGraphLanes(commits)

	laneByHash := make(map[string]int, len(commits))
	nextLane := nextLaneAfter(preferredLaneByHash)

	for _, commit := range commits {
		if preferredLane, ok := preferredLaneByHash[commit.Hash]; ok {
			laneByHash[commit.Hash] = preferredLane
		}

		if _, ok := laneByHash[commit.Hash]; !ok {
			laneByHash[commit.Hash] = nextLane
			nextLane++
		}

		currentLane := laneByHash[commit.Hash]

		for index, parent := range commit.Parents {
			if parentLane, ok := preferredLaneByHash[parent]; ok {
				laneByHash[parent] = parentLane
				continue
			}

			if _, exists := laneByHash[parent]; exists {
				continue
			}

			if index == 0 {
				laneByHash[parent] = currentLane
				continue
			}

			laneByHash[parent] = nextLane
			nextLane++
		}
	}

	return laneByHash
}
func preferredGitGraphLanes(commits []GitCommit) map[string]int {
	laneByRefGroup := make(map[string]int)
	laneByHash := make(map[string]int)
	nextLane := 0

	for _, commit := range commits {
		refGroup := preferredRefGroup(commit.Refs)
		if refGroup == "" {
			continue
		}

		lane, ok := laneByRefGroup[refGroup]
		if !ok {
			lane = nextLane
			laneByRefGroup[refGroup] = lane
			nextLane++
		}

		laneByHash[commit.Hash] = lane
	}

	return laneByHash
}
func preferredRefGroup(refs []string) string {
	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		switch {
		case ref == "HEAD":
			continue
		case ref == "main" || ref == "origin/main":
			return "main"
		case ref == "develop" || ref == "origin/develop":
			return "develop"
		case strings.HasPrefix(ref, "origin/"):
			return strings.TrimPrefix(ref, "origin/")
		case strings.HasPrefix(ref, "tag:"):
			continue
		case ref != "":
			return ref
		}
	}

	return ""
}
func nextLaneAfter(lanes map[string]int) int {
	next := 0
	for _, lane := range lanes {
		if lane >= next {
			next = lane + 1
		}
	}
	return next
}
