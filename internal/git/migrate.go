package gitstore

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type ghRepo struct {
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	Private       bool   `json:"private"`
	DefaultBranch string `json:"default_branch"`
	Language      string `json:"language"`
	Fork          bool   `json:"fork"`
	Archived      bool   `json:"archived"`
	CloneURL      string `json:"clone_url"`
}

// ListGitHubOwnerRepos lists non-fork repos for owner via GitHub API (uses secrets token).
func ListGitHubOwnerRepos(owner string) ([]ghRepo, error) {
	owner = strings.TrimSpace(owner)
	if owner == "" {
		return nil, fmt.Errorf("owner required")
	}
	tok := githubTokenLocal()
	client := &http.Client{Timeout: 45 * time.Second}
	var all []ghRepo
	for page := 1; page <= 10; page++ {
		var url string
		if tok != "" {
			url = fmt.Sprintf("https://api.github.com/user/repos?per_page=100&page=%d&affiliation=owner", page)
		} else {
			url = fmt.Sprintf("https://api.github.com/users/%s/repos?per_page=100&page=%d&type=owner", owner, page)
		}
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/vnd.github+json")
		if tok != "" {
			req.Header.Set("Authorization", "Bearer "+tok)
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode >= 300 {
			return nil, fmt.Errorf("github HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
		}
		var batch []ghRepo
		if err := json.Unmarshal(b, &batch); err != nil {
			return nil, err
		}
		if len(batch) == 0 {
			break
		}
		for _, r := range batch {
			if r.Fork || r.Archived {
				continue
			}
			if r.FullName != "" {
				parts := strings.SplitN(r.FullName, "/", 2)
				if len(parts) == 2 && !strings.EqualFold(parts[0], owner) {
					continue
				}
			}
			all = append(all, r)
		}
		if len(batch) < 100 {
			break
		}
	}
	return all, nil
}

// MigrateOpts controls bulk register from GitHub.
type MigrateOpts struct {
	Owner    string
	DryRun   bool
	Sync     bool // mirror-fetch after add
	SkipName map[string]bool
}

// MigrateFromGitHub registers each owner repo as a project (no build_on_fetch).
func MigrateFromGitHub(o MigrateOpts) (added, skipped []string, err error) {
	if o.Owner == "" {
		o.Owner = "PavelNeyman"
	}
	if o.SkipName == nil {
		o.SkipName = map[string]bool{"netductor": true}
	}
	repos, err := ListGitHubOwnerRepos(o.Owner)
	if err != nil {
		return nil, nil, err
	}
	existing, _ := LoadProjects()
	have := map[string]bool{}
	for _, p := range existing {
		have[p.Name] = true
	}
	for _, r := range repos {
		name := sanitize(r.Name)
		if name == "" || o.SkipName[name] || have[name] {
			skipped = append(skipped, r.Name)
			continue
		}
		host := "vps"
		lang := strings.ToLower(r.Language)
		if lang == "swift" || strings.Contains(strings.ToLower(r.Name), "ios") || strings.Contains(strings.ToLower(r.Name), "macos") || strings.Contains(strings.ToLower(r.Name), "mac-") {
			host = "mac"
		}
		up := r.FullName
		if up == "" {
			up = o.Owner + "/" + r.Name
		}
		p := Project{
			Name:     name,
			Upstream: up,
			Host:     host,
			Workflow: guessWorkflow(lang),
		}
		if o.DryRun {
			added = append(added, fmt.Sprintf("%s\thost=%s\t%s", name, host, up))
			continue
		}
		if err := AddProject(p, o.Sync); err != nil {
			return added, skipped, fmt.Errorf("%s: %w", name, err)
		}
		added = append(added, name)
		have[name] = true
	}
	return added, skipped, nil
}

func guessWorkflow(lang string) string {
	switch lang {
	case "go":
		return "ci/netductor.yml" // preferred if present; else FindDefault on build
	default:
		return "" // auto .github/workflows
	}
}


func githubTokenLocal() string {
	if v := os.Getenv("NETDUCTOR_GITHUB_TOKEN"); v != "" {
		return strings.TrimSpace(v)
	}
	if v := os.Getenv("GITHUB_TOKEN"); v != "" {
		return strings.TrimSpace(v)
	}
	for _, p := range []string{"/etc/netductor/secrets/github_token", "/etc/netductor/github_token"} {
		b, err := os.ReadFile(p)
		if err == nil {
			return strings.TrimSpace(string(b))
		}
	}
	return ""
}
