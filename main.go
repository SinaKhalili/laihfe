package main

import (
	"encoding/json"
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
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

var version string = "0.0.3"

// Global todo file path
var TodoFilePath string

// View styling
var (
	term    = termenv.ColorProfile()
	keyword = makeFgStyle("211")
	subtle  = makeFgStyle("241")
	dot     = colorFg(" • ", "236")
	special = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}
)

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
		filepath.Join(dir, ".todos.json"),
		"Path to the .todos.txt file which laihfe will read.\nWill default to `.todos.json` in the current user's home folder.",
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

		var todos ToDos
		json.Unmarshal(dat, &todos)

		printTodos(todos)
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

type TodoStates uint

const (
	DONE TodoStates = iota
	NOT_DONE
	TOMBSTONE
)

type ToDos struct {
	Items []ToDo `json:"todos"`
}

type ToDo struct {
	Text  string     `json:"text"`
	State TodoStates `json:"state"`
}

type model struct {
	textInput      textinput.Model
	currTodo       int
	todos          ToDos
	currMode       Modes
	cursorPosition int
	err            error
	undoList       []int
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

func printTodos(todos ToDos) {
	for index, elem := range todos.Items {
		checked := " "
		switch todos.Items[index].State {
		case DONE:
			checked = "x"
		case NOT_DONE: // Do nothing
		case TOMBSTONE:
			continue
		}
		fmt.Printf("[%s] %s\n", checked, elem.Text)
	}
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
	check(err)

	undoList := make([]int, 0)

	var todos ToDos
	json.Unmarshal(dat, &todos)

	return model{
		textInput: ti,
		todos:     todos,
		currMode:  INSERT,
		currTodo:  -1,
		err:       nil,
		undoList:  undoList,
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
				m.cursorPosition = moveDownToNextAliveTodo(m)
				return m, cmd
			case tea.KeyEnter:
				if m.currTodo != -1 {
					m.todos.Items[m.currTodo].Text = m.textInput.Value()
				} else {
					m.todos.Items = append(m.todos.Items, ToDo{
						Text:  m.textInput.Value(),
						State: NOT_DONE,
					})
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
				f, _ := json.MarshalIndent(m.todos, "", " ")
				_ = ioutil.WriteFile(TodoFilePath, f, 0644)
				return m, tea.Quit
			case "i":
				m.textInput.Focus()
				m.currMode = INSERT
				return m, cmd
			case "j":
				m.cursorPosition = moveDownToNextAliveTodo(m)
				return m, cmd
			case "k":
				m.cursorPosition = moveUpToNextAliveTodo(m)
				return m, cmd
			case "c":
				m.currTodo = m.cursorPosition
				m.textInput.SetValue(m.todos.Items[m.currTodo].Text)
				m.textInput.Focus()
				m.currMode = INSERT
				return m, cmd
			case "d":
				m.todos.Items[m.cursorPosition].State = TOMBSTONE
				m.undoList = append(m.undoList, m.cursorPosition)
				m.cursorPosition = moveDownToNextAliveTodo(m)
				if m.todos.Items[m.cursorPosition].State == TOMBSTONE {
					m.cursorPosition = moveUpToNextAliveTodo(m)
				}
				return m, cmd
			case "u":
				if len(m.undoList) > 0 {
					mostRecentKill := pop(&m.undoList)
					m.todos.Items[mostRecentKill].State = NOT_DONE
				}
				return m, cmd
			case "enter", "l":
				todostate := m.todos.Items[m.cursorPosition].State

				if todostate == DONE {
					m.todos.Items[m.cursorPosition].State = NOT_DONE
				} else if todostate == NOT_DONE {
					m.todos.Items[m.cursorPosition].State = DONE
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

func pop(list *[]int) int {
	// Fails on small length
	length := len(*list)
	last_elem := (*list)[length-1]
	*list = append((*list)[:length-1])
	return last_elem
}

func moveDownToNextAliveTodo(m model) int {
	// Helper to move down skipping the TOMBSTONE todos

	oldCurr := m.cursorPosition
	newCursor := m.cursorPosition + 1

	for newCursor < len(m.todos.Items) && m.todos.Items[newCursor].State == TOMBSTONE {
		newCursor++
	}
	if newCursor >= len(m.todos.Items) {
		newCursor = oldCurr
	}

	return newCursor
}

func moveUpToNextAliveTodo(m model) int {
	// Helper to move up through the TOMBSTONE todos

	oldCurr := m.cursorPosition
	newCursor := m.cursorPosition - 1

	for newCursor >= 0 && m.todos.Items[newCursor].State == TOMBSTONE {
		newCursor--
	}
	if newCursor < 0 {
		newCursor = oldCurr
	}

	return newCursor
}

func (m model) View() string {
	var finale strings.Builder
	if m.currMode == NORMAL {
		finale.WriteString("\nCurrmode: ⋉ Normal\n")
		finale.WriteString(
			"\n" +
				subtle("|") +
				"q: quit" +
				subtle("|") +
				"i: insert mode" +
				subtle("|") +
				"j/k: up/down" +
				subtle("|") +
				keyword("enter: toggle completed") +
				subtle("|") +
				"c: change item \n",
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

	for index, elem := range m.todos.Items {
		checked := " " // Default: it's not checked
		dirty := ""    // Default: it's not being changed
		pointer := " " // Default: it's not being pointed at

		if index == m.cursorPosition && m.currMode == NORMAL {
			pointer = "👉"
		}
		if index == m.currTodo {
			dirty = "*"
		}
		switch m.todos.Items[index].State {
		case DONE:
			checked = "x"
		case NOT_DONE: // Do nothing
		case TOMBSTONE:
			continue
		}
		fmt.Fprintf(&finale, "%s[%s] %s%s\n", pointer, checked, elem.Text, dirty)
	}

	return finale.String()
}

// Color a string foreground
func makeFgStyle(color string) func(string) string {
	return termenv.Style{}.Foreground(term.Color(color)).Styled
}

// Color a string's foreground with the given value.
func colorFg(val, color string) string {
	return termenv.String(val).Foreground(term.Color(color)).String()
}
