package respSearch

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

// AUR API URL for package search
const aurURL = "https://aur.archlinux.org/rpc/?v=5&type=search&arg="

// Structs for JSON parsing
type PackageInfo struct {
	Name        string `json:"Name"`
	Description string `json:"Description"`
	Version     string `json:"Version"`
	URL         string `json:"URL"`
}

type ApiResponse struct {
	Results []PackageInfo `json:"results"`
}

// Function to search for packages on AUR
func Search(term string) ([]PackageInfo, error) {
	resp, err := http.Get(aurURL + term)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to AUR: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to fetch data from AUR")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read AUR response: %w", err)
	}

	var apiResponse ApiResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse AUR response: %w", err)
	}

	return apiResponse.Results, nil
}

// Function to install a package by cloning and running makepkg
func Install(url string) error {
	// Clone the repository
	cmd := exec.Command("git", "clone", url)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	// Determine repository name from URL
	repoName := getRepoName(url)
	if err := os.Chdir(repoName); err != nil {
		return fmt.Errorf("failed to change to repo directory: %w", err)
	}

	// Run makepkg to build and install
	makeCmd := exec.Command("sh", "-c", "makepkg -si --noconfirm")
	makeCmd.Stdout = os.Stdout
	makeCmd.Stderr = os.Stderr

	if err := makeCmd.Run(); err != nil {
		return fmt.Errorf("makepkg failed: %w", err)
	}

	return nil
}

// Helper to extract repo name from URL
func getRepoName(url string) string {
	parts := strings.Split(url, "/")
	repoWithExt := parts[len(parts)-1]
	return strings.TrimSuffix(repoWithExt, ".git")
}
