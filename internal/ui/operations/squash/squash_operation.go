package squash

import (
	"slices"

	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/ui/view"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/operations"
)

var _ operations.Operation = (*Operation)(nil)
var _ common.Focusable = (*Operation)(nil)
var _ view.IHasActionMap = (*Operation)(nil)

type Operation struct {
	context     *context.MainContext
	from        jj.SelectedRevisions
	files       []string
	current     *jj.Commit
	keyMap      config.KeyMappings[key.Binding]
	keepEmptied bool
	interactive bool
	styles      styles
}

func (s *Operation) GetActionMap() map[string]common.Action {
	return map[string]common.Action{
		"j":     {Id: "revisions.down"},
		"k":     {Id: "revisions.up"},
		"esc":   {Id: "close squash", Args: nil, Switch: common.ScopeRevisions},
		"enter": {Id: "squash.apply", Args: nil, Switch: common.ScopeRevisions},
	}
}

func (s *Operation) IsFocused() bool {
	return true
}

type styles struct {
	dimmed       lipgloss.Style
	sourceMarker lipgloss.Style
	targetMarker lipgloss.Style
}

func (s *Operation) Init() tea.Cmd {
	return nil
}

func (s *Operation) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(common.InvokeActionMsg); ok {
		switch msg.Action.Id {
		case "squash.apply":
			return s, tea.Batch(common.InvokeAction(common.Action{Id: "close squash"}), s.context.RunInteractiveCommand(jj.Squash(s.from, s.current.GetChangeId(), s.files, s.keepEmptied, s.interactive), common.RefreshAndSelect(s.current.GetChangeId())))
		case "squash.keep_emptied":
			s.keepEmptied = !s.keepEmptied
		case "squash.interactive":
			s.interactive = !s.interactive
		}
	}
	return s, nil
}

func (s *Operation) View() string {
	return ""
}

func (s *Operation) SetSelectedRevision(commit *jj.Commit) {
	s.current = commit
}

func (s *Operation) Render(commit *jj.Commit, pos operations.RenderPosition) string {
	if pos != operations.RenderBeforeChangeId {
		return ""
	}

	isSelected := s.current != nil && s.current.GetChangeId() == commit.GetChangeId()
	if isSelected {
		return s.styles.targetMarker.Render("<< into >>")
	}
	sourceIds := s.from.GetIds()
	if slices.Contains(sourceIds, commit.ChangeId) {
		marker := "<< from >>"
		if s.keepEmptied {
			marker = "<< keep empty >>"
		}
		if s.interactive {
			marker += " (interactive)"
		}
		return s.styles.sourceMarker.Render(marker)
	}
	return ""
}

func (s *Operation) Name() string {
	return "squash"
}

func (s *Operation) ShortHelp() []key.Binding {
	return []key.Binding{
		s.keyMap.Apply,
		s.keyMap.Cancel,
		s.keyMap.Squash.KeepEmptied,
		s.keyMap.Squash.Interactive,
	}
}

func (s *Operation) FullHelp() [][]key.Binding {
	return [][]key.Binding{s.ShortHelp()}
}

type Option func(*Operation)

func WithFiles(files []string) Option {
	return func(op *Operation) {
		op.files = files
	}
}

func NewOperation(context *context.MainContext, from jj.SelectedRevisions, opts ...Option) *Operation {
	styles := styles{
		dimmed:       common.DefaultPalette.Get("squash dimmed"),
		sourceMarker: common.DefaultPalette.Get("squash source_marker"),
		targetMarker: common.DefaultPalette.Get("squash target_marker"),
	}
	o := &Operation{
		context: context,
		keyMap:  config.Current.GetKeyMap(),
		from:    from,
		styles:  styles,
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}
