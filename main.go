package main

import (
	"fmt"
	"os"
	"os/exec"

	respSearch "popstar/backend"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var screen = "main"

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
	searchBar          textinput.Model
	items              []list.Item
	isSearchBarFocused bool // Tracks if search bar is focused
}

func (m model) Init() tea.Cmd {
	return nil
}

var h int
var v int
var term_width int
var term_height int

// Check if the program is running as root (sudo)
func checkIfRunningAsRoot() bool {
	return os.Geteuid() == 0
}

// Run a command with sudo if not running as root
func runCommandWithSudo(command string, args []string) (string, error) {
	if !checkIfRunningAsRoot() {
		// If not running as root, prompt for sudo and re-run the program
		fmt.Println("This program needs sudo privileges. Please enter your password:")

		// Build the sudo command to re-run the program with sudo
		cmdArgs := append([]string{os.Args[0]}, os.Args[1:]...) // Keep the same arguments
		cmd := exec.Command("sudo", cmdArgs...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		// Run the program with sudo
		err := cmd.Run()
		if err != nil {
			return "", fmt.Errorf("error running program with sudo: %v", err)
		}
		// Program will restart with sudo, so no further code will run here.
		return "", nil
	}

	// Run the command normally when already running as root
	cmd := exec.Command(command, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error running command: %v", err)
	}
	return string(output), nil
}

// updateMain handles updates for the main screen
func updateMain(msg tea.Msg, m model) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "enter", "return":
			selectedIndex := m.list.Index()
			if selectedIndex == 0 {
				// Switch to search screen and focus on the search bar
				screen = "search"
				m.searchBar.Focus()

				// Manually update the size of the lists
				h, v = docStyle.GetFrameSize()
				m.list.SetSize(m.list.Width(), m.list.Height()) // Optional, for consistent sizing
				m.searchList.SetSize(m.searchList.Width(), m.searchList.Height())

				// Adjust the size for searchList to make it a bit lower
				searchListHeight := m.searchList.Height() - 3
				m.searchList.SetSize(m.searchList.Width(), searchListHeight)

				return m, nil
			}
		}
	case tea.WindowSizeMsg:
		term_width = msg.Width
		term_height = msg.Height
		h, v = docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
		m.searchList.SetSize(msg.Width-h-5, msg.Height-v)
	}

	// Update the list and capture any commands it returns
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// updateSearch handles updates for the search screen.
func updateSearch(msg tea.Msg, m model) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			// Switch back to the main screen
			screen = "main"
			return m, nil
		case "tab":
			// Toggle focus between the search bar and the search list
			m.isSearchBarFocused = !m.isSearchBarFocused
			if m.isSearchBarFocused {
				m.searchBar.Focus()
			} else {
				m.searchList.SetSize(m.searchList.Width(), m.searchList.Height())
			}
			return m, nil
		case "enter", "return":
			if m.isSearchBarFocused {
				// Handle search bar input (perform search)
				term := m.searchBar.Value()
				searchData, err := respSearch.Search(term)
				if err != nil {
					fmt.Println("Error:", err)
					return m, nil
				}

				// Populate search list with results
				var newItems []list.Item
				for _, result := range searchData {
					newItems = append(newItems, item{
						title: result.Name,
						desc:  result.Description,
					})
				}
				m.searchList.SetItems(newItems)
				m.isSearchBarFocused = !m.isSearchBarFocused
				if m.isSearchBarFocused {
					m.searchBar.Focus()
				} else {
					m.searchList.SetSize(m.searchList.Width(), m.searchList.Height())
				}
			} else {
				// Handle selection from the search list
				selectedIndex := m.searchList.Index()
				if selectedIndex >= 0 && selectedIndex < len(m.searchList.Items()) {
					selectedItem := m.searchList.Items()[selectedIndex].(item)
					tempUrl := "https://aur.archlinux.org/" + selectedItem.title + ".git"
					return m, tea.Batch(respSearch.Install(tempUrl))
				}
			}
			return m, nil
		}
	}

	// Update either the search bar or the list based on focus
	if m.isSearchBarFocused {
		m.searchBar, cmd = m.searchBar.Update(msg)
	} else {
		m.searchList, cmd = m.searchList.Update(msg)
	}

	return m, cmd
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch screen {
	case "main":
		return updateMain(msg, m)
	case "search":
		return updateSearch(msg, m)
	}
	return m, nil
}

func (m model) viewMain() string {
	return m.list.View()
}

func (m model) viewSearch() string {
	searchText := m.searchBar.View()
	if m.isSearchBarFocused {
		searchText = lipgloss.NewStyle().Bold(true).Render(searchText)
	}

	listView := m.searchList.View()
	if !m.isSearchBarFocused {
		listView = lipgloss.NewStyle().Bold(true).Render(listView)
	}

	return docStyle.Render(searchText + "\n" + listView)
}

func (m model) View() string {
	switch screen {
	case "main":
		return docStyle.Render(m.viewMain())
	case "search":
		return searchStyle.Render(m.viewSearch())
	default:
		panic("unknown screen")
	}
}

func main() {
	// Check if the program is running as root before proceeding

	mainItems := []list.Item{
		item{title: "Search", desc: "Search the aur"},
		item{title: "Remove", desc: "Remove a package :/"},
	}

	searchItems := []list.Item{}

	// Initialize search bar
	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.Focus()

	// Create the initial model
	m := model{
		items:              searchItems,
		list:               list.New(mainItems, list.NewDefaultDelegate(), 0, 0),
		searchList:         list.New(searchItems, list.NewDefaultDelegate(), 0, 0),
		searchBar:          ti,
		isSearchBarFocused: true, // Start with the search bar focused
	}

	// Set initial titles and sizes
	m.list.Title = "Popstar Repository Helper"
	m.list.SetSize(20, 20) // Initial size for the main list

	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
