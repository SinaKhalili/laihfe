package main

// A simple program demonstrating the text input component from the Bubbles
// component library.

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
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
	currTodo       []string
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

func initialModel() model {
	ti := textinput.NewModel()
	ti.Placeholder = "Hack the planet"
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20

	if _, err := os.Stat("./.todos.txt"); os.IsNotExist(err) {
		file, err := os.Create(".todos.txt")
		check(err)
		defer file.Close()
	}

	dat, err := ioutil.ReadFile("./.todos.txt")

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
		currTodo:  texts,
		currMode:  INSERT,
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
				m.currTodo = append(m.currTodo, m.textInput.Value())
				m.textInput.Reset()
				return m, cmd
			}
		}
		if m.currMode == NORMAL {
			switch msg.String() {
			case "q":
				f, err := os.OpenFile(".todos.txt", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
				check(err)
				for i, elem := range m.currTodo {
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
				if m.cursorPosition+1 < len(m.currTodo) {
					m.cursorPosition += 1
				}
				return m, cmd
			case "k":
				if m.cursorPosition > 0 {
					m.cursorPosition -= 1
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
	if m.currMode == NORMAL {
		// Do nothing
	}
	return m, cmd
}

func (m model) View() string {
	var finale strings.Builder
	if m.currMode == NORMAL {
		finale.WriteString("\nCurrmode: ⋉ Normal\n")
		finale.WriteString(
			"\n| q: quit | i: insert mode | j/k: up/down | enter: toggle completed |\n",
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

	for index, elem := range m.currTodo {
		if index == m.cursorPosition && m.currMode == NORMAL {
			fmt.Fprint(&finale, "👉")
		}
		checked := " "
		if _, ok := m.selected[index]; ok {
			checked = "x"
		}
		fmt.Fprintf(&finale, "[%s] %s\n", checked, elem)
	}

	return finale.String()
}
