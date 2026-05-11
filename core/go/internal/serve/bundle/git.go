package bundle

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const gitRefFieldSeparator = "|||PATCHCOURT|||"

type GitStatus struct {
	Root               string `json:"root"`
	Branch             string `json:"branch"`
	Head               string `json:"head"`
	ShortHead          string `json:"short_head"`
	HasWorktreeChanges bool   `json:"has_worktree_changes"`
	Ahead              int    `json:"ahead,omitempty"`
	Behind             int    `json:"behind,omitempty"`
	Upstream           string `json:"upstream,omitempty"`
}

type GitBranchesResponse struct {
	Root     string      `json:"root"`
	Current  string      `json:"current"`
	Branches []GitBranch `json:"branches"`
}

type GitBranch struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Head      string `json:"head"`
	ShortHead string `json:"short_head"`
	Current   bool   `json:"current,omitempty"`
	Upstream  string `json:"upstream,omitempty"`
	Ahead     int    `json:"ahead,omitempty"`
	Behind    int    `json:"behind,omitempty"`
}

type GitRefsResponse struct {
	Root string   `json:"root"`
	Refs []GitRef `json:"refs"`
}

type GitRef struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Target    string `json:"target"`
	ShortHash string `json:"short_hash"`
}

type GitCommitsResponse struct {
	Root    string      `json:"root"`
	Ref     string      `json:"ref,omitempty"`
	All     bool        `json:"all,omitempty"`
	Limit   int         `json:"limit"`
	Commits []GitCommit `json:"commits"`
}

type GitCommit struct {
	Hash      string   `json:"hash"`
	ShortHash string   `json:"short_hash"`
	Parents   []string `json:"parents,omitempty"`
	Refs      []string `json:"refs,omitempty"`
	Author    string   `json:"author"`
	Date      string   `json:"date"`
	Message   string   `json:"message"`
}

func registerGitRoutes(mux *http.ServeMux, root string) {
	mux.HandleFunc("/api/git/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		status, err := readGitStatus(r.Context(), root)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSONValue(w, http.StatusOK, status)
	})

	mux.HandleFunc("/api/git/branches", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		branches, err := readGitBranches(r.Context(), root)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSONValue(w, http.StatusOK, branches)
	})

	mux.HandleFunc("/api/git/refs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		refs, err := readGitRefs(r.Context(), root)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSONValue(w, http.StatusOK, refs)
	})

	registerGitGraphRoute(mux, root)

	mux.HandleFunc("/api/git/commits", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		limit := parsePositiveIntWithMax(r.URL.Query().Get("limit"), 50, 500)
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

		writeJSONValue(w, http.StatusOK, GitCommitsResponse{
			Root:    root,
			Ref:     ref,
			All:     all,
			Limit:   limit,
			Commits: commits,
		})
	})
}

func writeJSONValue(w http.ResponseWriter, status int, value any) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		http.Error(w, fmt.Sprintf("encode json: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSONBytes(w, status, append(data, '\n'))
}
