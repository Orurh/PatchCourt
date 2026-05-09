package bundle

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"
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

func parseAheadBehind(value string) (behind int, ahead int) {
	parts := strings.Fields(value)
	if len(parts) != 2 {
		return 0, 0
	}

	behind, _ = strconv.Atoi(parts[0])
	ahead, _ = strconv.Atoi(parts[1])

	return behind, ahead
}

func parsePositiveIntWithMax(raw string, fallback int, max int) int {
	if raw == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return fallback
	}

	if parsed > max {
		return max
	}

	return parsed
}

func parseBoolQuery(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func splitNonEmptyLines(value string) []string {
	lines := strings.Split(value, "\n")
	result := make([]string, 0, len(lines))

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		result = append(result, line)
	}

	return result
}

func runGitTrimmed(ctx context.Context, root string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = root

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}

	return strings.TrimSpace(string(out)), nil
}

func writeJSONValue(w http.ResponseWriter, status int, value any) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		http.Error(w, fmt.Sprintf("encode json: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSONBytes(w, status, append(data, '\n'))
}
