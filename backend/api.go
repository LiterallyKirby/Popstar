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

	"path/filepath"

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
	// Get the user's home directory
	_, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error getting home directory:", err)
		return tea.Quit
	}

	// Define the target directory within the user's home directory
	baseDir := filepath.Join(os.TempDir(), "popstarTemp")

	// Ensure the target directory exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		fmt.Println("Error creating directory:", err)
		return tea.Quit
	}

	// Change to the target base directory
	if err := os.Chdir(baseDir); err != nil {
		fmt.Println("Error changing to base directory:", err)
		return tea.Quit
	}

	// Extract repository name from the URL
	repoName := GetRepoName(url)

	// Clone the repository
	if err := runCommandWithPty("git", "clone", url); err != nil {
		fmt.Println("Error cloning repository:", err)
		return tea.Quit
	}

	// Change to the cloned repository directory
	if err := os.Chdir(filepath.Join(baseDir, repoName)); err != nil {
		fmt.Println("Failed to change to repository directory:", err)
		return tea.Quit
	}

	fmt.Println("Repository cloned successfully!")
	return nil
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
