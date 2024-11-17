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

	tea "github.com/charmbracelet/bubbletea"
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

func Install(url string) tea.Cmd {
	return func() tea.Msg {
		fmt.Println("Exiting to normal terminal to install the package...")

		// Store the current working directory
		originalDir, err := os.Getwd()
		if err != nil {
			fmt.Println("Failed to get current directory:", err)
			return nil
		}

		// Clear the terminal
		clearCmd := exec.Command("clear")
		clearCmd.Stdout = os.Stdout
		clearCmd.Run()

		// Clone the repository
		repoName := getRepoName(url)
		cloneCmd := exec.Command("git", "clone", url)
		cloneCmd.Stdout = os.Stdout
		cloneCmd.Stderr = os.Stderr
		if err := cloneCmd.Run(); err != nil {
			fmt.Println("Error cloning repository:", err)
			promptToContinue()
			return nil
		}

		// Change to the repository directory
		if err := os.Chdir(repoName); err != nil {
			fmt.Println("Failed to change to repo directory:", err)
			promptToContinue()
			return nil
		}

		// Run the installation
		makeCmd := exec.Command("sh", "-c", "makepkg -si -S --noconfirm")
		makeCmd.Stdout = os.Stdout
		makeCmd.Stderr = os.Stderr
		makeCmd.Stdin = os.Stdin
		if err := makeCmd.Run(); err != nil {
			fmt.Println("Package installation failed:", err)
			promptToContinue()
			return nil
		}

		// Return to the original directory
		if err := os.Chdir(originalDir); err != nil {
			fmt.Println("Failed to return to original directory:", err)
			promptToContinue()
			return nil
		}

		// Clean up the cloned repository
		rmDir := exec.Command("rm", "-rf", repoName)
		if err := rmDir.Run(); err != nil {
			fmt.Println("Failed to remove the installer directory:", err)
			promptToContinue()
			return nil
		}

		// Print success message and wait before returning to Bubble Tea
		clearCmd.Run()

		fmt.Println("Package installed successfully!")
		promptToContinue()

		return nil // Stop further updates
	}
}

// Helper to extract repo name from URL
func getRepoName(url string) string {
	parts := strings.Split(url, "/")
	repoWithExt := parts[len(parts)-1]
	return strings.TrimSuffix(repoWithExt, ".git")
}

// Helper to pause and wait for user confirmation
func promptToContinue() {
	fmt.Println("Press Enter to return to the menu...")
	fmt.Scanln()
}
