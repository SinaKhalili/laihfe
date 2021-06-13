package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

var version string = "0.0.2"

// Global todo file path
var TodoFilePath string

func main() {
	isListModePtr := flag.Bool("l",
		false,
		"Set to only output the current todos and exit immediately",
	)
	versionPtr := flag.Bool("v",
		false,
		"Show version number and exit",
	)

	usr, _ := user.Current()
	dir := usr.HomeDir
	todoFilePathPtr := flag.String(
		"f",
		filepath.Join(dir, ".todos.txt"),
		"Path to the .todos.txt file which laihfe will read.\nWill default to `.todos.txt` in the current user's home folder.",
	)

	flag.Parse()
	TodoFilePath = *todoFilePathPtr

	if *versionPtr {
		fmt.Printf("%s\n", version)
		return
	}

	if *isListModePtr {
		if _, err := os.Stat(TodoFilePath); os.IsNotExist(err) {
			file, err := os.Create(TodoFilePath)
			check(err)
			defer file.Close()
		}
		dat, err := ioutil.ReadFile(TodoFilePath)
		check(err)

		for _, elem := range strings.Split(string(dat), "\n") {
			if elem != "" {
				parsedLine := elem[1:]
				if elem[0] == 'x' {
					fmt.Printf("[x] ")
				} else {
					fmt.Printf("[ ] ")
				}
				fmt.Printf("%s\n", parsedLine)
			}
		}
		return
	}

	p := tea.NewProgram(initialModel(), tea.WithAltScreen())

	if err := p.Start(); err != nil {
		log.Fatal(err)
	}
}

type tickMsg struct{}
type errMsg error

type Modes uint

const (
	NORMAL Modes = iota
	INSERT
)

type model struct {
	textInput      textinput.Model
	currTodo       int
	todos          []string
	currMode       Modes
	cursorPosition int
	selected       map[int]struct{}
	err            error
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func remove(slice []string, s int) []string {
	if len(slice) > 0 {
		return append(slice[:s], slice[s+1:]...)
	}
	return make([]string, 0)
}

func initialModel() model {
	ti := textinput.NewModel()
	ti.Placeholder = "Hack the planet"
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20

	if _, err := os.Stat(TodoFilePath); os.IsNotExist(err) {
		file, err := os.Create(TodoFilePath)
		check(err)
		defer file.Close()
	}

	dat, err := ioutil.ReadFile(TodoFilePath)

	selected := make(map[int]struct{})
	var texts []string

	for _, elem := range strings.Split(string(dat), "\n") {
		if elem != "" {
			parsedLine := elem[1:]
			texts = append(texts, parsedLine)
			if elem[0] == 'x' {
				selected[len(texts)-1] = struct{}{}
			}
		}
	}
	check(err)

	return model{
		textInput: ti,
		todos:     texts,
		currMode:  INSERT,
		currTodo:  -1,
		selected:  selected,
		err:       nil,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.currMode == INSERT {
			switch msg.Type {
			case tea.KeyCtrlC:
				return m, tea.Quit
			case tea.KeyEsc:
				m.textInput.Blur()
				m.currMode = NORMAL
				return m, cmd
			case tea.KeyEnter:
				if m.currTodo != -1 {
					m.todos[m.currTodo] = m.textInput.Value()
				} else {
					m.todos = append(m.todos, m.textInput.Value())
				}
				m.currTodo = -1
				m.textInput.Reset()
				return m, cmd
			}
		}
		if m.currMode == NORMAL {
			switch msg.String() {
			case "Q":
				return m, tea.Quit
			case "q":
				f, err := os.OpenFile(TodoFilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
				check(err)
				for i, elem := range m.todos {
					line := ""

					if _, ok := m.selected[i]; ok {
						line += "x"
					} else {
						line += " "
					}

					line += elem
					_, err2 := f.Write([]byte(line + "\n"))
					check(err2)
				}
				defer f.Close()
				return m, tea.Quit
			case "i":
				m.textInput.Focus()
				m.currMode = INSERT
				return m, cmd
			case "j":
				if m.cursorPosition+1 < len(m.todos) {
					m.cursorPosition += 1
				}
				return m, cmd
			case "k":
				if m.cursorPosition > 0 {
					m.cursorPosition -= 1
				}
				return m, cmd
			case "c":
				m.currTodo = m.cursorPosition
				m.textInput.SetValue(m.todos[m.currTodo])
				m.textInput.Focus()
				m.currMode = INSERT
				return m, cmd
			case "d":
				m.todos = remove(m.todos, m.cursorPosition)

				// TODO: make this more efficient
				newMap := make(map[int]struct{})
				for key, _ := range m.selected {
					if key >= m.cursorPosition {
						newMap[key-1] = struct{}{}
					} else {
						newMap[key] = struct{}{}
					}
				}
				m.selected = newMap

				if m.cursorPosition == len(m.todos) {
					m.cursorPosition--
				}
				return m, cmd
			case "enter", "l":
				_, ok := m.selected[m.cursorPosition]
				if ok {
					delete(m.selected, m.cursorPosition)
				} else {
					m.selected[m.cursorPosition] = struct{}{}
				}
			}
		}

	// We handle errors just like any other message
	case errMsg:
		m.err = msg
		return m, nil
	}

	if m.currMode == INSERT {
		m.textInput, cmd = m.textInput.Update(msg)
	}

	return m, cmd
}

func (m model) View() string {
	var finale strings.Builder
	if m.currMode == NORMAL {
		finale.WriteString("\nCurrmode: ⋉ Normal\n")
		finale.WriteString(
			"\n| q: quit | i: insert mode | j/k: up/down | enter: toggle completed | c: change item |\n",
		)
	}
	if m.currMode == INSERT {
		finale.WriteString("\nCurrmode: ⌨️  Insert\n")
		finale.WriteString("\n| esc: normal mode | enter: add todo |\n")
	}
	fmt.Fprintf(
		&finale,
		"\n\nWhat would you like to get done?\n\n%s\n",
		m.textInput.View(),
	)

	for index, elem := range m.todos {
		checked := " " // Default: it's not checked
		dirty := ""    // Default: it's not being changed
		pointer := " " // Default: it's not being pointed at

		if index == m.cursorPosition && m.currMode == NORMAL {
			pointer = "👉"
		}
		if index == m.currTodo {
			dirty = "*"
		}
		if _, ok := m.selected[index]; ok {
			checked = "x"
		}
		fmt.Fprintf(&finale, "%s[%s] %s%s\n", pointer, checked, elem, dirty)
	}

	return finale.String()
}
