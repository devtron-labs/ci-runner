package main

import (
	"fmt"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	"log"
	"os"
	"path/filepath"
)

func CloneAndCheckout(ciRequest *CiRequest) error {
	for _, prj := range ciRequest.CiProjectDetails {
		// git clone
		log.Println("-----> git cloning " + prj.GitRepository)

		if prj.CheckoutPath != "./" {
			if _, err := os.Stat(prj.CheckoutPath); os.IsNotExist(err) {
				_ = os.Mkdir(prj.CheckoutPath, os.ModeDir)
			}
		}

		var r *git.Repository
		var cErr error
		var auth *http.BasicAuth
		switch prj.GitOptions.AuthMode {
		case AUTH_MODE_USERNAME_PASSWORD:
			auth = &http.BasicAuth{Password: prj.GitOptions.Password, Username: prj.GitOptions.UserName}
		case AUTH_MODE_ACCESS_TOKEN:
			auth = &http.BasicAuth{Password: prj.GitOptions.AccessToken, Username: prj.GitOptions.UserName}
		}

		switch prj.SourceType {
		case SOURCE_TYPE_BRANCH_FIXED:
			if len(prj.SourceValue) == 0 {
				prj.SourceValue = "master"
			}
			log.Println("-----> " + prj.GitRepository + " checking out branch " + prj.SourceValue)
			r, cErr = git.PlainClone(filepath.Join(workingDir, prj.CheckoutPath), false, &git.CloneOptions{
				Auth:          auth,
				URL:           prj.GitRepository,
				Progress:      os.Stdout,
				ReferenceName: plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", prj.SourceValue)),
				SingleBranch:  true,
			})
		case SOURCE_TYPE_TAG_REGEX:
			log.Println("-----> " + prj.GitRepository + " checking out tag " + prj.SourceValue)
			r, cErr = git.PlainClone(filepath.Join(workingDir, prj.CheckoutPath), false, &git.CloneOptions{
				Auth:          auth,
				URL:           prj.GitRepository,
				Progress:      os.Stdout,
				ReferenceName: plumbing.ReferenceName(fmt.Sprintf("refs/tags/%s", prj.GitTag)),
				SingleBranch:  true,
			})
		}

		if cErr != nil {
			log.Fatal("could not clone branch ", " err ", cErr)
		}

		w, wErr := r.Worktree()
		if wErr != nil {
			log.Fatal(wErr)
		}

		if prj.CommitHash != "" {
			log.Println("-----> " + prj.GitRepository + " git checking out commit " + prj.CommitHash)
			cErr := checkoutHash(w, prj.CommitHash)
			if cErr != nil {
				log.Println(cErr)
				return cErr
			}
		}
	}
	return nil
}

func checkoutHash(workTree *git.Worktree, hash string) error {
	err := workTree.Checkout(&git.CheckoutOptions{
		Hash: plumbing.NewHash(hash),
	})
	return err
}
