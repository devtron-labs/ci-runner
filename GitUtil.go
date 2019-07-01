package main

import (
	"fmt"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	"log"
	"os"
)

func CloneAndCheckout(ciRequest *CiRequest) error {
	for _, prj := range ciRequest.CiProjectDetails {
		// git clone
		log.Println("------> git cloning " + prj.GitRepository)
		if _, err := os.Stat(prj.CheckoutPath); os.IsNotExist(err) {
			mErr := os.Mkdir(prj.CheckoutPath, os.ModeDir)
			if mErr != nil {
				log.Println(err)
				os.Exit(2)
			}
		}

		var r *git.Repository
		var cErr error
		if prj.Branch == "" || prj.Branch == "master" {
			log.Println("------> " + prj.GitRepository + " cloning master")
			r, cErr = git.PlainClone(prj.CheckoutPath, false, &git.CloneOptions{
				Auth: &http.BasicAuth{
					Username: prj.GitOptions.UserName,
					Password: prj.GitOptions.Password,
				},
				URL:      prj.GitRepository,
				Progress: os.Stdout,
			})
		} else {
			log.Println("------> " + prj.GitRepository + " checking branch " + prj.Branch)
			r, cErr = git.PlainClone(prj.CheckoutPath, false, &git.CloneOptions{
				Auth: &http.BasicAuth{
					Username: prj.GitOptions.UserName,
					Password: prj.GitOptions.Password,
				},
				URL:           prj.GitRepository,
				Progress:      os.Stdout,
				ReferenceName: plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", prj.Branch)),
				SingleBranch:  true,
			})
		}
		if cErr != nil {
			log.Println(cErr)
			return cErr
		}

		w, wErr := r.Worktree()
		if wErr != nil {
			log.Println(wErr)
			return wErr
		}

		if prj.CommitHash != "" {
			log.Println("------> " + prj.GitRepository + " git checking out " + prj.CommitHash)
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
	log.Println("checking out hash ", hash)
	err := workTree.Checkout(&git.CheckoutOptions{
		Hash: plumbing.NewHash(hash),
	})
	return err
}
