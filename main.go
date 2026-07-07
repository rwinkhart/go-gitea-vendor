package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"

	"github.com/rwinkhart/go-boilerplate/back"
	"github.com/rwinkhart/go-boilerplate/other"
)

type giteaRespT struct {
	Data []struct {
		Language  string `json:"language"`
		MasterURL string `json:"original_url"`
		Name      string `json:"name"`
	}
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: " + os.Args[0] + " <base url>\n\nAPI token can be piped into stdin to run non-interactively!")
		os.Exit(0)
	}

	url := os.Args[1] + "/api/v1/repos/search?sort=updated&order=desc&limit=999"
	fmt.Print("Input token: ")
	token := back.ReadFromStdin()
	if len(token) == 0 {
		other.PrintError("No token provided; API token must be provided via stdin!", 1)
	}
	fmt.Print("\r            \r")

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		other.PrintError("Failed to create request: "+err.Error(), 2)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "token "+string(token))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		other.PrintError("Failed to make request: "+err.Error(), 3)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		other.PrintError("Failed to read response body: "+err.Error(), 4)
	}

	var giteaResp giteaRespT
	if err = json.Unmarshal(body, &giteaResp); err != nil {
		other.PrintError("Failed to unmarshal response: "+err.Error(), 5)
	}

	const workingDir = "repos-new"
	const finishedDir = "repos"
	os.Mkdir(workingDir, 0755)
	for _, repo := range giteaResp.Data {
		if repo.Language == "Go" {
			cloneCmd := exec.Command("git", "clone", "--depth=1", repo.MasterURL)
			cloneCmd.Dir = workingDir
			if err = cloneCmd.Run(); err != nil {
				other.PrintError("Failed to clone "+repo.MasterURL+": "+err.Error(), 6)
			}
			vendorCmd := exec.Command("go", "mod", "vendor")
			vendorCmd.Dir = workingDir + "/" + repo.Name
			if err = vendorCmd.Run(); err != nil {
				other.PrintError("Failed to vendor "+repo.MasterURL+": "+err.Error(), 7)
			}
			if err = os.RemoveAll(workingDir + "/" + repo.Name + "/.git"); err != nil {
				other.PrintError("Failed to remove .git directory for "+repo.MasterURL, 8)
			}
		}
	}
	isAccessible, err := back.TargetIsFile(finishedDir, false)
	if isAccessible && err == nil {
		if err = os.RemoveAll(finishedDir + ".bak"); err != nil {
			other.PrintError("Failed to remove old backup dir", 9)
		}
		if err = os.Rename(finishedDir, finishedDir+".bak"); err != nil {
			other.PrintError("Failed to backup old repos dir", 10)
		}
	}
	if err = os.Rename(workingDir, finishedDir); err != nil {
		other.PrintError("Failed to rename new repos dir", 11)
	}
}
