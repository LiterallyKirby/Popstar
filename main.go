package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	respSearch "popstar/backend"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var screen = "main"

// STYLES
var docStyle = lipgloss.NewStyle().
	Margin(2, 2).
	Padding(2,0,1,0).
	Border(lipgloss.RoundedBorder()).
	BorderTop(true).
	BorderLeft(true)

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
				screen = "search"
				m.searchBar.Focus()
				return m, nil
			} else if m.list.Index() == 1 {
				installedPackages, err := respSearch.GetInstalledPackages()
				if err != nil {
					log.Println("Error fetching installed packages:", err)
					return m, nil
				}

				var newItems []list.Item
				for _, pkg := range installedPackages {
					newItems = append(newItems, item{
						title: pkg,
						desc:  "Installed package",
					})
				}
				m.packageList.SetItems(newItems)
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

				// Sorting the things idfk
				sort.Slice(searchData, func(i, j int) bool {
					return searchData[i].Popularity > searchData[j].Popularity
				})

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
var firstWord string
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
			m.isSearchBarFocused = !m.isSearchBarFocused
			if m.isSearchBarFocused {
				m.searchBar.Focus()
			} else {
				m.packageList.SetSize(m.packageList.Width(), m.packageList.Height())
			}
			return m, nil
		case "enter", "return":
	if m.isSearchBarFocused {
		// Filter installed packages by search term
		term := strings.ToLower(strings.TrimSpace(m.searchBar.Value()))

		installedPackages, err := respSearch.GetInstalledPackages()
		if err != nil {
			log.Println("Error fetching installed packages:", err)
			return m, nil
		}

		var newItems []list.Item
		for _, pkg := range installedPackages {
			if strings.Contains(strings.ToLower(pkg), term) {
				newItems = append(newItems, item{
					title: pkg,
					desc:  "Installed package",
				})
			}
		}

		m.packageList.SetItems(newItems)
	} else {
		selectedIndex := m.packageList.Index()
		if selectedIndex >= 0 {
			// Retrieve selected item
			selectedItem := m.packageList.Items()[selectedIndex].(item)

			// Extract the first word from the title
			firstWord := ""
			words := strings.Fields(selectedItem.title)
			if len(words) > 0 {
				firstWord = words[0]
			}

			fmt.Printf("Removing package: %s\n", firstWord)
			return m, removePackage(firstWord)
		}
	}

		}
	}

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

func customDelegateForScreen(screen string) list.DefaultDelegate {
	delegate := list.NewDefaultDelegate()

	// Set help text dynamically
	switch screen {
	case "main":
		delegate.ShortHelpFunc = func() []key.Binding {
			return []key.Binding{
				key.NewBinding(key.WithKeys("↑/↓"), key.WithHelp("↑/↓", "")),
				key.NewBinding(key.WithKeys("enter"), key.WithHelp("Enter", "select")),
				key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("Ctrl+C", "quit")),
			}
		}
	case "search":
		delegate.ShortHelpFunc = func() []key.Binding {
			return []key.Binding{
				key.NewBinding(key.WithKeys("tab"), key.WithHelp("Tab", "")),
				key.NewBinding(key.WithKeys("enter"), key.WithHelp("Enter", "search/select")),
				key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "back to main")),
			}
		}
	case "remove":
		delegate.ShortHelpFunc = func() []key.Binding {
			return []key.Binding{
				key.NewBinding(key.WithKeys("tab"), key.WithHelp("Tab", "")),
				key.NewBinding(key.WithKeys("enter"), key.WithHelp("Enter", "remove package")),
				key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "back to main")),
			}
		}
	default:
		delegate.ShortHelpFunc = func() []key.Binding {
			return []key.Binding{
				key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("Ctrl+C", "quit")),
			}
		}
	}

	return delegate
}

func (m model) View() string {
	var helpText string
	switch screen {
	case "main":
		helpText = customDelegateForScreen("main").ShortHelpFunc()[0].Help().Desc
	case "search":
		helpText = customDelegateForScreen("search").ShortHelpFunc()[0].Help().Desc
	case "remove":
		helpText = customDelegateForScreen("remove").ShortHelpFunc()[0].Help().Desc
	}

	switch screen {
	case "main":
		return docStyle.Render(m.list.View()) + "\n" + helpText
	case "search":
		return docStyle.Render(m.searchBar.View()+"\n"+m.searchList.View()) + "\n" + helpText
	case "remove":
		return docStyle.Render(m.searchBar.View()+"\n"+m.packageList.View()) + "\n" + helpText
	default:
		return "Unknown screen"
	}
}

func main() {
	mainItems := []list.Item{
		item{title: "Search", desc: "Search the AUR"},
		item{title: "Remove", desc: "Remove a package"},
	}

	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.Focus()

	m := model{
		items:              []list.Item{},
		list:               list.New(mainItems, customDelegateForScreen("main"), 20, 20),
		searchList:         list.New([]list.Item{}, customDelegateForScreen("search"), 0, 0),
		packageList:        list.New([]list.Item{}, customDelegateForScreen("remove"), 0, 0),
		searchBar:          ti,
		isSearchBarFocused: true,
	}
	m.list.Title = "Popstar Repository Helper"

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running program: %v", err)
	}
}
