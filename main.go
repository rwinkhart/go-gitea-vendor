package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/config"
	"github.com/go-git/go-git/v6/plumbing"
	gitclient "github.com/go-git/go-git/v6/plumbing/client"
	"github.com/go-git/go-git/v6/plumbing/object"
	githttp "github.com/go-git/go-git/v6/plumbing/transport/http"
	"github.com/go-git/go-git/v6/storage/memory"
	"github.com/rwinkhart/go-boilerplate/back"
	"github.com/rwinkhart/go-boilerplate/other"
)

const workingDir = "repos-new"
const finishedDir = "repos"
const hashName = ".gvv-hash"

type giteaRespT struct {
	Data []struct {
		Owner struct {
			Organization string `json:"username"`
		} `json:"owner"`
		Language  string `json:"language"`
		MasterURL string `json:"original_url"`
		Name      string `json:"name"`
	}
}

func logError(message string) {
	fmt.Println(back.AnsiWarning + message + back.AnsiReset)
}

func main() {
	// args
	if len(os.Args) < 3 {
		fmt.Println("Usage: " + os.Args[0] + " <base url> <output organization>\n\nAPI token can be piped into stdin to run non-interactively!")
		os.Exit(0)
	}
	baseURL := os.Args[1]
	apiURL := baseURL + "/api/v1/repos/search?sort=updated&order=asc&limit=999"
	outputOrganization := os.Args[2]
	fmt.Print("Input token: ")
	token := back.ReadFromStdin()
	if len(token) == 0 {
		other.PrintError("No token provided; API token must be provided via stdin!", 1)
	}
	fmt.Print("\r            \r")

	// start cleanup
	if err := os.RemoveAll(workingDir); err != nil {
		other.PrintError("Failed to remove "+workingDir+": "+err.Error(), 2)
	}

	// api call
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		other.PrintError("Failed to create request: "+err.Error(), 3)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "token "+string(token))
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		other.PrintError("Failed to make request: "+err.Error(), 4)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		other.PrintError("Failed to read response body: "+err.Error(), 5)
	}

	// cloning and vendoring
	var giteaResp giteaRespT
	if err = json.Unmarshal(body, &giteaResp); err != nil {
		other.PrintError("Failed to unmarshal response: "+err.Error(), 6)
	}
	os.Mkdir(workingDir, 0755)
repoLoop:
	for _, repoResp := range giteaResp.Data {
		if repoResp.Owner.Organization != outputOrganization && repoResp.Language == "Go" {
			currentRepoDir := workingDir + "/" + repoResp.Name + "-ggv"
			//// skip if local copy is already up-to-date
			targetRepoDir := finishedDir + "/" + repoResp.Name + "-ggv"
			if localHash, err := os.ReadFile(targetRepoDir + "/" + hashName); err == nil {
				rem := git.NewRemote(memory.NewStorage(), &config.RemoteConfig{
					Name: "origin",
					URLs: []string{repoResp.MasterURL},
				})
				if refs, err := rem.List(&git.ListOptions{}); err == nil {
					for _, ref := range refs {
						if ref.Name() == plumbing.HEAD {
							// HEAD is a symbolic ref; resolve to its target branch
							target := ref.Target()
							for _, ref2 := range refs {
								if ref2.Name() == target {
									if strings.TrimSpace(string(localHash)) == ref2.Hash().String() {
										fmt.Println("Skipping " + repoResp.MasterURL + " (already up-to-date)")
										if err = os.MkdirAll(currentRepoDir, 0755); err != nil {
											logError("Failed to create dummy repo for " + repoResp.MasterURL + ": " + err.Error())
											continue repoLoop
										}
										if err = os.WriteFile(currentRepoDir+"/"+hashName, localHash, 0644); err != nil {
											logError("Failed to write dummy head hash for " + repoResp.MasterURL + ": " + err.Error())
										}
										continue repoLoop
									}
									break
								}
							}
							break
						}
					}
				} else {
					logError("Failed to fetch remote refs for " + repoResp.MasterURL + ": " + err.Error())
					continue repoLoop
				}
			}
			//// clone
			oldRepo, err := git.PlainClone(currentRepoDir, &git.CloneOptions{URL: repoResp.MasterURL, Progress: nil, Depth: 1})
			if err != nil {
				logError("Failed to clone " + repoResp.MasterURL + ": " + err.Error())
				continue repoLoop
			}
			//// store commit hash
			head, err := oldRepo.Head()
			if err != nil {
				logError("Failed to get head of " + repoResp.MasterURL + ": " + err.Error())
				continue repoLoop
			}
			if err = os.WriteFile(currentRepoDir+"/"+hashName, []byte(head.Hash().String()), 0644); err != nil {
				logError("Failed to write head hash for " + repoResp.MasterURL + ": " + err.Error())
				continue repoLoop
			}
			//// vendor
			vendorCmd := exec.Command("go", "mod", "vendor")
			vendorCmd.Dir = currentRepoDir
			if err = vendorCmd.Run(); err != nil {
				logError("Failed to vendor " + repoResp.MasterURL + ": " + err.Error())
				continue repoLoop
			}
			//// re-init
			if err = os.RemoveAll(currentRepoDir + "/.git"); err != nil {
				logError("Failed to remove .git directory for " + repoResp.MasterURL + ": " + err.Error())
				continue repoLoop
			}
			newRepo, err := git.PlainInit(currentRepoDir, false)
			if err != nil {
				logError("Failed to init in " + currentRepoDir + ": " + err.Error())
				continue repoLoop
			}
			//// set origin
			if _, err = newRepo.CreateRemote(&config.RemoteConfig{
				Name: "origin",
				URLs: []string{baseURL + "/" + outputOrganization + "/" + repoResp.Name + "-ggv.git"},
			}); err != nil {
				logError("Failed to set origin for " + currentRepoDir + ": " + err.Error())
				continue repoLoop
			}
			//// add
			wt, err := newRepo.Worktree()
			if err != nil {
				logError("Failed to get worktree for " + currentRepoDir + ": " + err.Error())
				continue repoLoop
			}
			if err = wt.AddGlob("."); err != nil {
				logError("Failed to add files in " + currentRepoDir + ": " + err.Error())
				continue repoLoop
			}
			//// commit
			if _, err = wt.Commit("go-gitea-vendor", &git.CommitOptions{Author: &object.Signature{Name: "ggv", Email: "gvv@local.local", When: time.Now()}}); err != nil {
				logError("Failed to commit in " + currentRepoDir + ": " + err.Error())
				continue repoLoop
			}
			//// push
			if err = newRepo.Push(&git.PushOptions{
				Force: true,
				ClientOptions: []gitclient.Option{
					gitclient.WithHTTPAuth(&githttp.BasicAuth{
						Username: "null",
						Password: string(token),
					}),
				},
			}); err != nil {
				logError("Failed to push " + currentRepoDir + ": " + err.Error())
				continue repoLoop
			}
		}
	}

	// end cleanup
	if err = os.RemoveAll(finishedDir); err != nil {
		other.PrintError("Failed to remove old repos dir: "+err.Error(), 7)
	}
	if err = os.Rename(workingDir, finishedDir); err != nil {
		other.PrintError("Failed to rename new repos dir: "+err.Error(), 8)
	}
}
