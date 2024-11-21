package respSearch

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/creack/pty"
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

func Get_Files(url string) tea.Cmd {
	// Clone the repository
	repoName := GetRepoName(url)
	if err := runCommandWithPty("git", "clone", url); err != nil {
		fmt.Println("Error cloning repository:", err)
		promptToContinue()
		return tea.Quit
	}

	// Change to the repository directory
	if err := os.Chdir(repoName); err != nil {
		fmt.Println("Failed to change to repo directory:", err)
		promptToContinue()
		return tea.Quit
	}
	// Restart Bubble Tea program
	return tea.Quit
}

// Helper to run commands with pty
func runCommandWithPty(name string, args ...string) error {
	cmd := exec.Command(name, args...)

	// Create a new pty
	pty, err := pty.Start(cmd)
	if err != nil {
		return fmt.Errorf("failed to start pty for command %s: %w", name, err)
	}
	defer pty.Close()

	// Copy pty output to the real terminal
	go func() {
		_, _ = io.Copy(os.Stdout, pty)
	}()

	// Wait for the command to complete
	err = cmd.Wait()
	if err != nil {
		return fmt.Errorf("command %s failed: %w", name, err)
	}

	return nil
}

// Helper to extract repo name from URL
func GetRepoName(url string) string {
	parts := strings.Split(url, "/")
	repoWithExt := parts[len(parts)-1]
	return strings.TrimSuffix(repoWithExt, ".git")
}

// Helper to pause and wait for user confirmation
func promptToContinue() {
	fmt.Println("Press Enter to return to the menu...")
	fmt.Scanln()
}
