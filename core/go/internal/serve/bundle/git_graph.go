package bundle

import (
	"net/http"
	"strings"
)

const gitGraphSchemaVersion = "patchcourt.git_graph.v1"

type GitGraphResponse struct {
	SchemaVersion string           `json:"schema_version"`
	Root          string           `json:"root"`
	Ref           string           `json:"ref,omitempty"`
	All           bool             `json:"all,omitempty"`
	Limit         int              `json:"limit"`
	Commits       []GitGraphCommit `json:"commits"`
	Layout        GitGraphLayout   `json:"layout"`
}

type GitGraphCommit struct {
	Hash          string   `json:"hash"`
	ShortHash     string   `json:"short_hash"`
	Parents       []string `json:"parents,omitempty"`
	Children      []string `json:"children,omitempty"`
	Refs          []string `json:"refs,omitempty"`
	Author        string   `json:"author"`
	Date          string   `json:"date"`
	Message       string   `json:"message"`
	Lane          int      `json:"lane"`
	ParentLanes   []int    `json:"parent_lanes,omitempty"`
	ChildLanes    []int    `json:"child_lanes,omitempty"`
	IsMerge       bool     `json:"is_merge,omitempty"`
	IsBranchPoint bool     `json:"is_branch_point,omitempty"`
}

type GitGraphLayout struct {
	RowHeight int               `json:"row_height"`
	LaneWidth int               `json:"lane_width"`
	Segments  []GitGraphSegment `json:"segments,omitempty"`
	Edges     []GitGraphEdge    `json:"edges,omitempty"`
}

type GitGraphSegment struct {
	Lane    int `json:"lane"`
	FromRow int `json:"from_row"`
	ToRow   int `json:"to_row"`
}

type GitGraphEdge struct {
	FromLane int    `json:"from_lane"`
	ToLane   int    `json:"to_lane"`
	FromRow  int    `json:"from_row"`
	ToRow    int    `json:"to_row"`
	Kind     string `json:"kind"` // branch | merge | parent
}

func registerGitGraphRoute(mux *http.ServeMux, root string) {
	mux.HandleFunc("/api/git/graph", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		limit := parsePositiveIntWithMax(r.URL.Query().Get("limit"), 100, 500)
		ref := strings.TrimSpace(r.URL.Query().Get("ref"))
		all := parseBoolQuery(r.URL.Query().Get("all"))

		commits, err := readGitCommits(r.Context(), root, GitCommitQuery{
			Ref:   ref,
			All:   all,
			Limit: limit,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		graph := buildGitGraph(root, ref, all, limit, commits)
		writeJSONValue(w, http.StatusOK, graph)
	})
}

func buildGitGraph(root string, ref string, all bool, limit int, commits []GitCommit) GitGraphResponse {
	childrenByParent := buildChildrenByParent(commits)
	laneByHash := assignGitGraphLanes(commits)

	graphCommits := make([]GitGraphCommit, 0, len(commits))

	for _, commit := range commits {
		children := childrenByParent[commit.Hash]
		lane := laneByHash[commit.Hash]

		parentLanes := make([]int, 0, len(commit.Parents))
		for _, parent := range commit.Parents {
			if parentLane, ok := laneByHash[parent]; ok {
				parentLanes = append(parentLanes, parentLane)
			}
		}

		childLanes := make([]int, 0, len(children))
		for _, child := range children {
			if childLane, ok := laneByHash[child]; ok {
				childLanes = append(childLanes, childLane)
			}
		}

		graphCommits = append(graphCommits, GitGraphCommit{
			Hash:          commit.Hash,
			ShortHash:     commit.ShortHash,
			Parents:       commit.Parents,
			Children:      children,
			Refs:          commit.Refs,
			Author:        commit.Author,
			Date:          commit.Date,
			Message:       commit.Message,
			Lane:          lane,
			ParentLanes:   uniqueInts(parentLanes),
			ChildLanes:    uniqueInts(childLanes),
			IsMerge:       len(commit.Parents) > 1,
			IsBranchPoint: len(children) > 1,
		})
	}

	return GitGraphResponse{
		SchemaVersion: gitGraphSchemaVersion,
		Root:          root,
		Ref:           ref,
		All:           all,
		Limit:         limit,
		Commits:       graphCommits,
		Layout:        buildGitGraphLayout(graphCommits),
	}
}

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
	if len(segments) <= 1 {
		return segments
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
