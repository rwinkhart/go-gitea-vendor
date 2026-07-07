package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/config"
	gitclient "github.com/go-git/go-git/v6/plumbing/client"
	githttp "github.com/go-git/go-git/v6/plumbing/transport/http"
	"github.com/rwinkhart/go-boilerplate/back"
	"github.com/rwinkhart/go-boilerplate/other"
)

const workingDir = "repos-new"
const finishedDir = "repos"

type giteaRespT struct {
	Data []struct {
		MirrorInterval string `json:"mirror_interval"`
		Language       string `json:"language"`
		MasterURL      string `json:"original_url"`
		Name           string `json:"name"`
	}
}

func main() {
	// args
	if len(os.Args) < 3 {
		fmt.Println("Usage: " + os.Args[0] + " <base url> <username> [output organization]\n\nAPI token can be piped into stdin to run non-interactively!")
		os.Exit(0)
	}
	baseURL := os.Args[1]
	apiURL := baseURL + "/api/v1/repos/search?sort=updated&order=desc&limit=999"
	username := os.Args[2]
	var organization string
	if len(os.Args) == 4 {
		organization = os.Args[3]
	}
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
	for _, repo := range giteaResp.Data {
		if repo.Language == "Go" && repo.MirrorInterval != "" {
			//// clone
			currentRepoDir := workingDir + "/" + repo.Name + "-ggv"
			_, err = git.PlainClone(currentRepoDir, &git.CloneOptions{URL: repo.MasterURL, Progress: nil, Depth: 1})
			if err != nil {
				other.PrintError("Failed to clone "+repo.MasterURL+": "+err.Error(), 7)
			}
			//// vendor
			vendorCmd := exec.Command("go", "mod", "vendor")
			vendorCmd.Dir = currentRepoDir
			if err = vendorCmd.Run(); err != nil {
				other.PrintError("Failed to vendor "+repo.MasterURL+": "+err.Error(), 8)
			}
			//// re-init
			if err = os.RemoveAll(currentRepoDir + "/.git"); err != nil {
				other.PrintError("Failed to remove .git directory for "+repo.MasterURL, 9)
			}
			goRepo, err := git.PlainInit(currentRepoDir, false)
			if err != nil {
				other.PrintError("Failed to init in "+currentRepoDir, 10)
			}
			//// set origin
			if _, err = goRepo.CreateRemote(&config.RemoteConfig{
				Name: "origin",
				URLs: []string{baseURL + "/" + organization + "/" + repo.Name + "-ggv.git"},
			}); err != nil {
				other.PrintError("Failed to set origin for "+currentRepoDir+": "+err.Error(), 15)
			}
			//// add
			wt, err := goRepo.Worktree()
			if err != nil {
				other.PrintError("Failed to get worktree for "+currentRepoDir, 11)
			}
			if err = wt.AddGlob("."); err != nil {
				other.PrintError("Failed to add files in "+currentRepoDir+": "+err.Error(), 12)
			}
			//// commit
			if _, err = wt.Commit("go-gitea-vendor", &git.CommitOptions{}); err != nil {
				other.PrintError("Failed to commit in "+currentRepoDir+": "+err.Error(), 13)
			}
			//// push
			if err = goRepo.Push(&git.PushOptions{
				Force: true,
				ClientOptions: []gitclient.Option{
					gitclient.WithHTTPAuth(&githttp.BasicAuth{
						Username: username,
						Password: string(token),
					}),
				},
			}); err != nil {
				other.PrintError("Failed to push "+currentRepoDir+": "+err.Error(), 17)
			}
		}
	}

	// end cleanup
	isAccessible, err := back.TargetIsFile(finishedDir, false)
	if isAccessible && err == nil {
		if err = os.RemoveAll(finishedDir + ".bak"); err != nil {
			other.PrintError("Failed to remove old backup dir", 14)
		}
		if err = os.Rename(finishedDir, finishedDir+".bak"); err != nil {
			other.PrintError("Failed to backup old repos dir", 15)
		}
	}
	if err = os.Rename(workingDir, finishedDir); err != nil {
		other.PrintError("Failed to rename new repos dir", 16)
	}
}
