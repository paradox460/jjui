package status

import (
	"strings"
	"time"

	"github.com/idursun/jjui/internal/ui/view"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/ui/common"
)

type commandStatus int

const (
	none commandStatus = iota
	commandRunning
	commandCompleted
	commandFailed
)

var _ view.IViewModel = (*Model)(nil)

type Model struct {
	*view.ViewNode
	spinner spinner.Model
	command string
	status  commandStatus
	running bool
	width   int
	styles  styles
}

func (m *Model) Mount(v *view.ViewNode) {
	m.ViewNode = v
	v.Id = m.GetId()
	v.NeedsRefresh = true
}

func (m *Model) GetId() view.ViewId {
	return view.StatusViewId
}

type styles struct {
	shortcut lipgloss.Style
	dimmed   lipgloss.Style
	text     lipgloss.Style
	title    lipgloss.Style
	success  lipgloss.Style
	error    lipgloss.Style
}

const CommandClearDuration = 3 * time.Second

type clearMsg string

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case clearMsg:
		if m.command == string(msg) {
			m.command = ""
			m.status = none
		}
		return m, nil
	case common.CommandRunningMsg:
		m.command = string(msg)
		m.status = commandRunning
		return m, m.spinner.Tick
	case common.CommandCompletedMsg:
		if msg.Err != nil {
			m.status = commandFailed
		} else {
			m.status = commandCompleted
		}
		commandToBeCleared := m.command
		return m, tea.Tick(CommandClearDuration, func(time.Time) tea.Msg {
			return clearMsg(commandToBeCleared)
		})
	default:
		var cmd tea.Cmd
		if m.status == commandRunning {
			m.spinner, cmd = m.spinner.Update(msg)
		}
		return m, cmd
	}
}

func (m *Model) View() string {
	commandStatusMark := m.styles.text.Render(" ")
	if m.status == commandRunning {
		commandStatusMark = m.styles.text.Render(m.spinner.View())
	} else if m.status == commandFailed {
		commandStatusMark = m.styles.error.Render("✗ ")
	} else if m.status == commandCompleted {
		commandStatusMark = m.styles.success.Render("✓ ")
	} else {

		// Get the ID of the currently focused view
		if focusedView := m.ViewManager.GetFocusedView(); focusedView != nil {
			if keymap, ok := focusedView.Model.(help.KeyMap); ok {
				commandStatusMark = m.helpView(keymap)
			}
		}
		commandStatusMark = lipgloss.PlaceHorizontal(m.width, 0, commandStatusMark, lipgloss.WithWhitespaceBackground(m.styles.text.GetBackground()))
	}

	var mode string
	// Get the ID of the currently focused view
	if focusedView := m.ViewManager.GetFocusedView(); focusedView != nil {
		mode = string(focusedView.Id)
	}

	modeWidth := max(10, len(mode)+2)
	ret := m.styles.text.Render(strings.ReplaceAll(m.command, "\n", "⏎"))

	mode = m.styles.title.Width(modeWidth).Render("", mode)
	ret = lipgloss.JoinHorizontal(lipgloss.Left, mode, m.styles.text.Render(" "), commandStatusMark, ret)
	height := lipgloss.Height(ret)
	return lipgloss.Place(m.width, height, 0, 0, ret, lipgloss.WithWhitespaceBackground(m.styles.text.GetBackground()))
}

func (m *Model) helpView(keyMap help.KeyMap) string {
	if keyMap == nil {
		return ""
	}
	shortHelp := keyMap.ShortHelp()
	var entries []string
	for _, binding := range shortHelp {
		if !binding.Enabled() {
			continue
		}
		h := binding.Help()
		entries = append(entries, m.styles.shortcut.Render(h.Key)+m.styles.dimmed.PaddingLeft(1).Render(h.Desc))
	}
	help := strings.Join(entries, m.styles.dimmed.Render(" • "))
	return help
}

func New() view.IViewModel {
	styles := styles{
		shortcut: common.DefaultPalette.Get("status shortcut"),
		dimmed:   common.DefaultPalette.Get("status dimmed"),
		text:     common.DefaultPalette.Get("status text"),
		title:    common.DefaultPalette.Get("status title"),
		success:  common.DefaultPalette.Get("status success"),
		error:    common.DefaultPalette.Get("status error"),
	}
	s := spinner.New()
	s.Spinner = spinner.Dot

	m := Model{
		spinner: s,
		command: "",
		status:  none,
		styles:  styles,
	}
	return &m
}
