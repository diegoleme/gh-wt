package gh

import (
	"fmt"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
)

// GraphQLClient returns an authenticated GraphQL client.
func GraphQLClient() (*api.GraphQLClient, error) {
	return api.DefaultGraphQLClient()
}

// Repo returns the current repository.
func Repo() (repository.Repository, error) {
	return repository.Current()
}

// RESTClient returns an authenticated REST client.
func RESTClient() (*api.RESTClient, error) {
	return api.DefaultRESTClient()
}

// DefaultBranch returns the default branch of the current repository.
func DefaultBranch() (string, error) {
	repo, err := Repo()
	if err != nil {
		return "", err
	}

	client, err := RESTClient()
	if err != nil {
		return "", err
	}

	var result struct {
		DefaultBranch string `json:"default_branch"`
	}
	err = client.Get(fmt.Sprintf("repos/%s/%s", repo.Owner, repo.Name), &result)
	if err != nil {
		return "", err
	}

	return result.DefaultBranch, nil
}
