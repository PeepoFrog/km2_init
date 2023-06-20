package adapters

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

// GitHubAdapter is a struct to hold the GitHub client
type GitHubAdapter struct {
	client *github.Client
}
type Repository struct {
	Owner   string
	Repo    string
	Version string
}

type Repositories struct {
	repos []Repository
}

// Add a new Repository to Repositories, version can be = ""
func (r *Repositories) Set(owner, repo, version string) {
	newRepo := Repository{Owner: owner, Repo: repo, Version: version}
	r.repos = append(r.repos, newRepo)
}
func (r *Repositories) Get() []Repository {
	return r.repos
}

// Iterate over Repositories and perform an action on each Repository
// func (r *Repositories) Iterate(action func(repo Repository)) {
// 	for _, repo := range r.repos {
// 		action(repo)
// 	}
// }

func Fetch(r Repositories, accessToken string) Repositories {
	adapter := NewGitHubAdapter(accessToken)

	var wg sync.WaitGroup
	results := make(chan Repository)

	for _, repo := range r.repos {
		wg.Add(1)
		go func(owner, repo string) {
			defer wg.Done()

			latestRelease, err := adapter.GetLatestRelease(owner, repo)
			if err != nil {
				log.Printf("Error fetching latest release for %s/%s: %v\n", owner, repo, err)
				return
			}

			results <- Repository{Owner: owner, Repo: repo, Version: *latestRelease.TagName}
		}(repo.Owner, repo.Repo)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var updatedRepos []Repository
	for result := range results {
		updatedRepos = append(updatedRepos, result)
	}

	return Repositories{repos: updatedRepos}
}

// NewGitHubAdapter initializes a new GitHubAdapter instance
func NewGitHubAdapter(accessToken string) *GitHubAdapter {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	tc := oauth2.NewClient(context.Background(), ts)

	return &GitHubAdapter{
		client: github.NewClient(tc),
	}
}

// GetLatestRelease fetches the latest release from the specified repository
func (gh *GitHubAdapter) GetLatestRelease(owner, repo string) (*github.RepositoryRelease, error) {
	release, _, err := gh.client.Repositories.GetLatestRelease(context.Background(), owner, repo)
	if err != nil {
		return nil, err
	}
	return release, nil
}

func DownloadBinaryFromRepo(ctx context.Context, client *github.Client, owner, repo, binaryName, tag string) {
	var release *github.RepositoryRelease
	var err error
	log.Printf("downloading %s from %s/%s, tag:%s\n", binaryName, owner, repo, tag)
	switch tag {
	case "latest":
		release, _, err = client.Repositories.GetLatestRelease(ctx, owner, repo)
		if err != nil {
			log.Fatalf("Error fetching latest release: %v", err)
		}
	default:
		release, _, err = client.Repositories.GetReleaseByTag(ctx, owner, repo, tag)
		if err != nil {
			log.Fatalf("Error fetching latest release: %v", err)
		}
	}

	var asset *github.ReleaseAsset
	for _, a := range release.Assets {
		if *a.Name == binaryName {
			asset = &a
			break
		}
	}

	if asset == nil {
		log.Fatalf("Binary not found in the latest release: %s", binaryName)
	}

	resp, err := http.Get(*asset.BrowserDownloadURL)
	if err != nil {
		log.Fatalf("Error downloading binary: %v", err)
	}
	defer resp.Body.Close()

	out, err := os.Create(binaryName)
	if err != nil {
		log.Fatalf("Error creating binary file: %v", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		log.Fatalf("Error writing binary to file: %v", err)
	}

	fmt.Println("Binary file downloaded successfully")
}
