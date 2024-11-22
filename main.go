package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	respSearch "popstar/backend"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var screen = "main"
var og_wd, _ = os.Getwd()

// STYLES
var docStyle = lipgloss.NewStyle().
	Margin(1, 2).
	Border(lipgloss.RoundedBorder()).
	BorderTop(true).
	BorderLeft(true)
var searchStyle = lipgloss.NewStyle().
	Margin(1, 2)

type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

type model struct {
	list               list.Model
	searchList         list.Model
	packageList        list.Model
	searchBar          textinput.Model
	items              []list.Item
	isSearchBarFocused bool // Tracks if search bar is focused
}

func (m model) Init() tea.Cmd {
	return nil
}

var h, v, term_width, term_height int

func checkIfRunningAsRoot() bool {
	return os.Geteuid() == 0
}

func runCommandWithSudo(command string, args []string) (string, error) {
	if !checkIfRunningAsRoot() {
		fmt.Println("This program needs sudo privileges. Please enter your password:")
		cmdArgs := append([]string{os.Args[0]}, os.Args[1:]...)
		cmd := exec.Command("sudo", cmdArgs...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		err := cmd.Run()
		if err != nil {
			return "", fmt.Errorf("error running program with sudo: %v", err)
		}
		return "", nil
	}

	cmd := exec.Command(command, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error running command: %v", err)
	}
	return string(output), nil
}

func updateMain(msg tea.Msg, m model) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "enter", "return":
			m.searchBar.Focus()
			if m.list.Index() == 0 {
				// Switch to "Search" screen
				screen = "search"
				m.searchBar.Focus()
				return m, nil
			} else if m.list.Index() == 1 {
				// Switch to "Remove" screen and load installed packages
				installedPackages, err := respSearch.GetInstalledPackages()
				if err != nil {
					log.Println("Error fetching installed packages:", err)
					return m, nil
				}

				// Populate packageList with installed packages
				var newItems []list.Item
				for _, pkg := range installedPackages {
					newItems = append(newItems, item{
						title: pkg,
						desc:  "Installed package",
					})
				}
				m.packageList.SetItems(newItems)

				// Set focus to the search bar on the "Remove" screen
				screen = "remove"
				m.searchBar.Focus()
				return m, nil
			}
			m.searchBar.Focus()
		}
	case tea.WindowSizeMsg:
		term_width, term_height = msg.Width, msg.Height
		h, v = docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
		m.searchList.SetSize(msg.Width-h-5, msg.Height-v)
		m.packageList.SetSize(msg.Width-h-5, msg.Height-v)
	}
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func updateSearch(msg tea.Msg, m model) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			screen = "main"
			m.searchBar.Reset()
			return m, nil
		case "tab":
			m.isSearchBarFocused = !m.isSearchBarFocused
			if m.isSearchBarFocused {
				m.searchBar.Focus()
			} else {
				m.searchList.SetSize(m.searchList.Width(), m.searchList.Height())
			}
			return m, nil
		case "enter", "return":
			if m.isSearchBarFocused {
				term := m.searchBar.Value()
				searchData, err := respSearch.Search(term)
				if err != nil {
					log.Println("Error during search:", err)
					return m, nil
				}
				var newItems []list.Item
				for _, result := range searchData {
					newItems = append(newItems, item{
						title: result.Name,
						desc:  result.Description,
					})
				}
				m.searchList.SetItems(newItems)
			} else {
				selectedIndex := m.searchList.Index()
				if selectedIndex >= 0 {
					selectedItem := m.searchList.Items()[selectedIndex].(item)
					tempUrl := "https://aur.archlinux.org/" + selectedItem.title + ".git"
					baseDir := filepath.Join(os.TempDir(), "popstarTemp")
					if err := os.RemoveAll(baseDir); err != nil {
						log.Fatalf("Error cleaning temp directory: %v", err)
					}
					respSearch.Get_Files(tempUrl)
					return m, openEditor()
				}
			}
		}
	}
	if m.isSearchBarFocused {
		m.searchBar, cmd = m.searchBar.Update(msg)
	} else {
		m.searchList, cmd = m.searchList.Update(msg)
	}
	return m, cmd
}

type editorFinishedMsg struct{ err error }

func openEditor() tea.Cmd {
	return tea.Batch(
		// Step 1: Run the installation with makepkg
		tea.ExecProcess(exec.Command("makepkg", "-si", "--noconfirm"), func(err error) tea.Msg {
			if err != nil {
				fmt.Printf("Error during makepkg: %v\n", err)
				return editorFinishedMsg{err}
			}
			fmt.Println("Installation completed successfully.")
			return nil
		}),
	)
}

func removePackage(item string) tea.Cmd {
	return tea.Batch(
		// Step 1: Run the installation with makepkg
		tea.ExecProcess(exec.Command("sudo", "pacman", "-Rns", item), func(err error) tea.Msg {
			if err != nil {
				fmt.Printf("Error during makepkg: %v\n", err)
				return editorFinishedMsg{err}
			}
			fmt.Println("Installation completed successfully.")
			return nil
		}),
	)
}

func updateRemove(msg tea.Msg, m model) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			m.searchBar.Reset()
			screen = "main"
			return m, nil
		case "tab":
			// Toggle focus between the search bar and package list
			m.isSearchBarFocused = !m.isSearchBarFocused
			if m.isSearchBarFocused {
				m.searchBar.Focus()
			} else {
				m.packageList.SetSize(m.packageList.Width(), m.packageList.Height())
			}
			return m, nil
		case "enter", "return":
			if m.isSearchBarFocused {
				// Search for packages based on search term
				term := m.searchBar.Value()
				installedPackages, err := respSearch.GetInstalledPackages()
				if err != nil {
					log.Println("Error fetching installed packages:", err)
					return m, nil
				}

				// Filter installed packages by search term
				var newItems []list.Item
				for _, pkg := range installedPackages {
					if strings.Contains(pkg, term) {
						newItems = append(newItems, item{
							title: pkg,
							desc:  "Installed package",
						})
					}
				}

				// Update package list with filtered items
				m.packageList.SetItems(newItems)
			} else {
				// Remove the selected package
				selectedIndex := m.packageList.Index()
				if selectedIndex >= 0 {
					selectedItem := m.packageList.Items()[selectedIndex].(item)

					toRemove := strings.Fields(selectedItem.title)
					fmt.Printf("Selected package for removal: %s\n", toRemove[0])

					// Run the command to remove the package
					return m, removePackage(toRemove[0])
				}
			}
		}
	}

	// After the package is removed, refresh the list of installed packages
	installedPackages, err := respSearch.GetInstalledPackages()
	if err != nil {
		log.Println("Error fetching installed packages:", err)
		return m, nil
	}

	// Update package list with remaining items
	var newItems []list.Item
	for _, pkg := range installedPackages {
		newItems = append(newItems, item{
			title: pkg,
			desc:  "Installed package",
		})
	}

	m.packageList.SetItems(newItems)

	// Update either the search bar or the package list based on focus
	if m.isSearchBarFocused {
		m.searchBar, cmd = m.searchBar.Update(msg)
	} else {
		m.packageList, cmd = m.packageList.Update(msg)
	}

	return m, cmd
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch screen {
	case "main":
		return updateMain(msg, m)
	case "search":
		return updateSearch(msg, m)
	case "remove":
		return updateRemove(msg, m)
	}
	return m, nil
}

func (m model) View() string {
	switch screen {
	case "main":
		return docStyle.Render(m.list.View())
	case "search":
		return docStyle.Render(m.searchBar.View() + "\n" + m.searchList.View())
	case "remove":
		return docStyle.Render(m.searchBar.View() + "\n" + m.packageList.View())
	default:
		return "Unknown screen"
	}
}

func main() {
	mainItems := []list.Item{
		item{title: "Search", desc: "Search the AUR"},
		item{title: "Remove", desc: "Remove a package :/"},
	}

	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.Focus()

	m := model{
		items:              []list.Item{},
		list:               list.New(mainItems, list.NewDefaultDelegate(), 20, 20),
		searchList:         list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0),
		packageList:        list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0),
		searchBar:          ti,
		isSearchBarFocused: true,
	}
	m.list.Title = "Popstar Repository Helper"

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running program: %v", err)
	}
}
