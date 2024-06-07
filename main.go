package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/bitfield/script"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
)

var kwStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("204")).Background(lipgloss.Color("235"))

func main() {
	var data []byte
	var err error

	if !isatty.IsTerminal(os.Stdin.Fd()) && !isatty.IsCygwinTerminal(os.Stdin.Fd()) {
		data, err = io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	} else {
		fmt.Fprintln(os.Stderr, "you need to pipe a json to the program")
		os.Exit(1)
	}
	var tmpJSON map[string]interface{}
	err = json.Unmarshal(data, &tmpJSON)
	if err != nil {
		fmt.Fprintln(os.Stderr, "invalid json:", err)
		os.Exit(1)
	}
	data, _ = json.MarshalIndent(tmpJSON, "", "  ")
	p := tea.NewProgram(initialModel(data), tea.WithAltScreen())

	if m, err := p.Run(); err != nil {
		log.Fatal(err)
	} else {
		fmt.Println(m.(model).jsonpath.Value())
	}
}

func beautifyJSON(js string) string {
	var tmpJSON interface{}
	_ = json.Unmarshal([]byte(js), &tmpJSON)
	data, _ := json.MarshalIndent(tmpJSON, "", "  ")
	return string(data)
}

type errMsg error

type model struct {
	jsonpath textinput.Model
	viewport viewport.Model
	text     string
	out      string
	oldPath  string
	// oldTxt   string
	width  int
	height int
	err    error
}

func initialModel(input []byte) model {
	ti := textinput.New()
	ti.Placeholder = "jsonpath here"

	oldPath := "."
	vp := viewport.New(70, 20)
	ti.Focus()
	return model{
		jsonpath: ti,
		text:     string(input),
		oldPath:  oldPath,
		err:      nil,
		viewport: vp,
	}
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyCtrlC:
			return m, tea.Quit
		default:
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
		m.sizeInputs()
	case errMsg:
		m.err = msg
		return m, nil
	}
	m.jsonpath, cmd = m.jsonpath.Update(msg)
	cmds = append(cmds, cmd)

	newPath := m.jsonpath.Value()
	if newPath != m.oldPath {
		m.out, m.err = script.Echo(m.text).JQ(newPath).String()
		m.viewport.SetContent(beautifyJSON(m.out))
		m.oldPath = newPath
	}
	return m, tea.Batch(cmds...)
}

func (m *model) sizeInputs() {
	m.viewport.Width = m.width
	m.viewport.Height = m.height - 2
}

func (m model) View() string {
	err := ""
	if m.err != nil {
		err = m.err.Error()
	}
	return fmt.Sprintf(
		"%s\n    %s\n%s",
		m.jsonpath.View(),
		kwStyle.Render(err),
		m.viewport.View(),
	)
}
