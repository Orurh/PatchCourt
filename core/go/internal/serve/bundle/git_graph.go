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
