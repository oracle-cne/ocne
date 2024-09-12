// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helm

import (
	"bytes"
	"fmt"
	"github.com/go-git/go-billy/v5/memfs"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage/memory"
	"helm.sh/helm/v3/pkg/getter"
	"io"
	"strings"
)

// GitGetter is a struct that implements the Getter interface
type GitGetter struct {
}

// NewGitProvider returns a provider struct that contains a scheme and a construction for a GitGetter struct
func NewGitProvider() getter.Provider {
	p := getter.Provider{
		Schemes: []string{"git+http", "git+ssh", "git+https"},
		New:     NewGitGetter,
	}
	return p
}

// NewGitGetter constructs a valid git client as a Getter
func NewGitGetter(options ...getter.Option) (getter.Getter, error) {
	return &GitGetter{}, nil
}

// Get goes to a url in the format of git+http[s]://provider/organization/repo&/path-to-file?[tag,branch,commit]=value
// or in the format of git+ssh://username@provider/organization/repo&/path-to-file?[tag,branch,commit]=value
// and clones that repo, opens it in memory and returns the contents of the specified file
// This format was inspired from https://github.com/aslafy-z/helm-git
func (g GitGetter) Get(url string, options ...getter.Option) (*bytes.Buffer, error) {
	repoURL, filePath, valueType, value, err := parseGitURL(url)
	if err != nil {
		return nil, err
	}
	inMemoryFileSystem := memfs.New()
	r, err := git.Clone(memory.NewStorage(), inMemoryFileSystem, &git.CloneOptions{
		URL: repoURL,
	})
	workTree, err := r.Worktree()
	if err != nil {
		return nil, err
	}
	err = checkout(workTree, valueType, value)
	if err != nil {
		return nil, err
	}
	file, err := inMemoryFileSystem.Open(filePath)
	if err != nil {
		return nil, err
	}
	x, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return bytes.NewBuffer(x), nil
}

// parseGitURL takes in a git url, separates out the repo url, file path, whether a tag, branch, or commit is being returned
// and the corresponding value for that tag, branch, or commit
func parseGitURL(url string) (string, string, string, string, error) {
	// Parse out the git+ prefix
	url = url[4:]
	if !(strings.Contains(url, "?") && strings.Contains(url, "&") && strings.Contains(url, "=")) {
		return "", "", "", "", fmt.Errorf("the provided url is not in the correct format")
	}
	splitURL := strings.Split(url, "&")
	repoURL := splitURL[0]
	filePathSplit := strings.Split(splitURL[1], "?")
	valueSplit := strings.Split(filePathSplit[1], "=")

	return repoURL, filePathSplit[0], valueSplit[0], valueSplit[1], nil

}

// checkout is a helper function that preforms a git checkout, it takes in a string representing the type of the git object to be committed
// and the value of that git object, along with a pointer to a git worktree
func checkout(worktree *git.Worktree, valueType string, value string) error {
	var err error
	if valueType == "commit" {
		err = worktree.Checkout(&git.CheckoutOptions{
			Create: false,
			Hash:   plumbing.NewHash(value),
		})
		return err
	}
	var referenceName plumbing.ReferenceName
	if valueType == "tag" {
		referenceName = plumbing.NewTagReferenceName(value)
	} else if valueType == "branch" {
		referenceName = plumbing.NewRemoteReferenceName("origin", value)
	} else {
		return fmt.Errorf("A type of tag, commit, or branch was not specified in the url")
	}
	err = worktree.Checkout(&git.CheckoutOptions{
		Create: false,
		Branch: referenceName,
	})
	return err
}
