package revset

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/common/autocompletion"
	appContext "github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/view"
)

var actionMap = map[string]actions.Action{
	"esc": {Id: "revset.cancel", Next: []actions.Action{{Id: "switch revisions"}}},
	"enter": {Id: "revset.accept", Next: []actions.Action{
		{Id: "switch revisions"},
	}},
}

type EditRevSetMsg struct {
	Clear bool
}

var _ view.IHasActionMap = (*Model)(nil)
var _ help.KeyMap = (*Model)(nil)

type Model struct {
	*common.Sizeable
	Editing         bool
	autoComplete    *autocompletion.AutoCompletionInput
	keymap          keymap
	History         []string
	historyIndex    int
	currentInput    string
	historyActive   bool
	MaxHistoryItems int
	context         *appContext.MainContext
	styles          styles
}

func (m *Model) Read(value string) string {
	if value == "$revset" {
		return m.context.CurrentRevset
	}
	return ""
}

func (m *Model) ShortHelp() []key.Binding {
	return m.keymap.ShortHelp()
}

func (m *Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{m.keymap.ShortHelp()}
}

func (m *Model) GetActionMap() map[string]actions.Action {
	return actionMap
}

type styles struct {
	promptStyle lipgloss.Style
	textStyle   lipgloss.Style
}

func (m *Model) IsFocused() bool {
	return m.Editing
}

type keymap struct{}

func (k keymap) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "complete")),
		key.NewBinding(key.WithKeys("ctrl+n"), key.WithHelp("ctrl+n", "next")),
		key.NewBinding(key.WithKeys("ctrl+p"), key.WithHelp("ctrl+p", "prev")),
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "accept")),
		key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "quit")),
	}
}

func (k keymap) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp()}
}

func New(context *appContext.MainContext) *Model {
	styles := styles{
		promptStyle: common.DefaultPalette.Get("revset title"),
		textStyle:   common.DefaultPalette.Get("revset text"),
	}

	revsetAliases := context.JJConfig.RevsetAliases
	completionProvider := NewCompletionProvider(revsetAliases)
	autoComplete := autocompletion.New(completionProvider, autocompletion.WithStylePrefix("revset"))

	autoComplete.SetValue(context.DefaultRevset)
	autoComplete.Focus()

	return &Model{
		Sizeable:        &common.Sizeable{Width: 0, Height: 0},
		context:         context,
		Editing:         false,
		keymap:          keymap{},
		autoComplete:    autoComplete,
		History:         []string{},
		historyIndex:    -1,
		MaxHistoryItems: 50,
		styles:          styles,
	}
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) AddToHistory(input string) {
	if input == "" {
		return
	}

	for i, item := range m.History {
		if item == input {
			m.History = append(m.History[:i], m.History[i+1:]...)
			break
		}
	}

	m.History = append([]string{input}, m.History...)

	if len(m.History) > m.MaxHistoryItems && m.MaxHistoryItems > 0 {
		m.History = m.History[:m.MaxHistoryItems]
	}

	m.historyIndex = -1
	m.historyActive = false
}

func (m *Model) SetHistory(history []string) {
	m.History = history
	m.historyIndex = -1
	m.historyActive = false
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case actions.InvokeActionMsg:
		switch msg.Action.Id {
		case "revset.cancel":
			m.Editing = false
			m.autoComplete.Blur()
			return m, nil
		case "revset.accept":
			if m.Editing {
				m.Editing = false
				m.autoComplete.Blur()
				value := m.autoComplete.Value()
				return m, common.UpdateRevSet(value)
			}
			return m, nil
		case "revset.set":
			rs := msg.Action.Get("revset", m.context.CurrentRevset).(string)
			//example syntax: $revset | ancestors($change_id, 1)
			// it should read $revset and $change_id variables from the active views and then replace them with actual values
			rs = strings.ReplaceAll(rs, "$revset", m.context.Read("$revset"))
			rs = strings.ReplaceAll(rs, "$change_id", m.context.Read("$change_id"))
			m.context.CurrentRevset = rs
			return m, common.RefreshAndSelect("")
		case "revset.edit":
			shouldClear := msg.Action.Get("clear", false).(bool)
			m.Editing = true
			m.autoComplete.Focus()
			if shouldClear {
				m.autoComplete.SetValue("")
			}
			m.historyActive = false
			m.historyIndex = -1
			return m, m.autoComplete.Init()
		}
	case tea.KeyMsg:
		if !m.Editing {
			return m, nil
		}
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.Editing = false
			m.autoComplete.Blur()
			return m, nil
		case tea.KeyEnter:
			m.Editing = false
			m.autoComplete.Blur()
			value := m.autoComplete.Value()
			return m, tea.Batch(common.Close, common.UpdateRevSet(value))
		case tea.KeyUp:
			if len(m.History) > 0 {
				if !m.historyActive {
					m.currentInput = m.autoComplete.Value()
					m.historyActive = true
				}

				if m.historyIndex < len(m.History)-1 {
					m.historyIndex++
					m.autoComplete.SetValue(m.History[m.historyIndex])
					m.autoComplete.CursorEnd()
				}
				return m, nil
			}
		case tea.KeyDown:
			if m.historyActive {
				if m.historyIndex > 0 {
					m.historyIndex--
					m.autoComplete.SetValue(m.History[m.historyIndex])
				} else {
					m.historyIndex = -1
					m.historyActive = false
					m.autoComplete.SetValue(m.currentInput)
				}
				m.autoComplete.CursorEnd()
				return m, nil
			}
		}
	case common.UpdateRevSetMsg:
		if m.Editing {
			m.Editing = false
		}
	}

	var cmd tea.Cmd
	m.autoComplete, cmd = m.autoComplete.Update(msg)
	return m, cmd
}

func (m *Model) View() string {
	var w strings.Builder
	w.WriteString(m.styles.promptStyle.PaddingRight(1).Render("revset:"))
	if m.Editing {
		w.WriteString(m.autoComplete.View())
	} else {
		revset := m.context.DefaultRevset
		if m.context.CurrentRevset != "" {
			revset = m.context.CurrentRevset
		}
		w.WriteString(m.styles.textStyle.Render(revset))
	}
	return lipgloss.Place(m.Width, m.Height, 0, 0, w.String(), lipgloss.WithWhitespaceBackground(m.styles.textStyle.GetBackground()))
}
