package main

import (
	"fmt"
	"net/url"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	"gopkg.in/src-d/go-billy.v3/osfs"
	git "gopkg.in/src-d/go-git.v4"
	git_config "gopkg.in/src-d/go-git.v4/config"
	git_plumbing "gopkg.in/src-d/go-git.v4/plumbing"
	git_http "gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	"gopkg.in/src-d/go-git.v4/storage/memory"
)

type gitMount arvados.Mount

func (gm gitMount) validate() error {
	if gm.Path != "/" {
		return fmt.Errorf("cannot mount git_tree path %q -- only \"/\" is supported", gm.Path)
	}
	return nil
}

// ExtractTree extracts the specified tree into dir, which is an
// existing empty local directory.
func (gm gitMount) extractTree(ac IArvadosClient, dir string) error {
	err := gm.validate()
	if err != nil {
		return err
	}
	baseURL, err := ac.Discovery("gitUrl")
	if err != nil {
		return fmt.Errorf("discover gitUrl from API: %s", err)
	}
	u, err := url.Parse(baseURL.(string))
	if err != nil {
		return fmt.Errorf("parse gitUrl %q: %s", baseURL, err)
	}
	u, err = u.Parse("/" + gm.UUID + ".git")
	if err != nil {
		return fmt.Errorf("build git url from %q, %q: %s", baseURL, gm.UUID, err)
	}
	store := memory.NewStorage()
	repo, err := git.Init(store, osfs.New(dir))
	if err != nil {
		return fmt.Errorf("init repo: %s", err)
	}
	_, err = repo.CreateRemote(&git_config.RemoteConfig{
		Name: "origin",
		URLs: []string{u.String()},
	})
	if err != nil {
		return fmt.Errorf("create remote %q: %s", u.String(), err)
	}
	err = repo.Fetch(&git.FetchOptions{
		RemoteName: "origin",
		Auth:       git_http.NewBasicAuth("none", arvadostest.ActiveToken),
	})
	if err != nil {
		return fmt.Errorf("git fetch %q: %s", u.String(), err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("worktree failed: %s", err)
	}
	err = wt.Checkout(&git.CheckoutOptions{
		Hash: git_plumbing.NewHash(gm.Commit),
	})
	if err != nil {
		return fmt.Errorf("checkout failed: %s", err)
	}
	return nil
}
