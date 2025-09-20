package details

import (
	"bufio"
	"fmt"
	"path"
	"reflect"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/confirmation"
	"github.com/idursun/jjui/internal/ui/context"
)

type status uint8

var (
	Added    status = 0
	Deleted  status = 1
	Modified status = 2
	Renamed  status = 3
)

type item struct {
	status   status
	name     string
	fileName string
	selected bool
	conflict bool
}

func (f item) Title() string {
	status := "M"
	switch f.status {
	case Added:
		status = "A"
	case Deleted:
		status = "D"
	case Modified:
		status = "M"
	case Renamed:
		status = "R"
	}

	return fmt.Sprintf("%s %s", status, f.name)
}
func (f item) Description() string { return "" }
func (f item) FilterValue() string { return f.name }

type styles struct {
	Added    lipgloss.Style
	Deleted  lipgloss.Style
	Modified lipgloss.Style
	Renamed  lipgloss.Style
	Selected lipgloss.Style
	Dimmed   lipgloss.Style
	Text     lipgloss.Style
	Conflict lipgloss.Style
}

type Model struct {
	*DetailsList
	revision     *jj.Commit
	mode         mode
	height       int
	confirmation *confirmation.Model
	context      *context.MainContext
	keyMap       config.KeyMappings[key.Binding]
	styles       styles
}

func (m *Model) ShortHelp() []key.Binding {
	if m.confirmation != nil {
		return m.confirmation.ShortHelp()
	}
	return []key.Binding{
		m.keyMap.Cancel,
		m.keyMap.Details.Diff,
		m.keyMap.Details.ToggleSelect,
		m.keyMap.Details.Split,
		m.keyMap.Details.Squash,
		m.keyMap.Details.Restore,
		m.keyMap.Details.Absorb,
		m.keyMap.Details.RevisionsChangingFile,
	}
}

func (m *Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{m.ShortHelp()}
}

type updateCommitStatusMsg struct {
	summary       string
	selectedFiles []string
}

func New(context *context.MainContext, revision *jj.Commit, height int) *Model {
	keyMap := config.Current.GetKeyMap()

	s := styles{
		Added:    common.DefaultPalette.Get("revisions details added"),
		Deleted:  common.DefaultPalette.Get("revisions details deleted"),
		Modified: common.DefaultPalette.Get("revisions details modified"),
		Renamed:  common.DefaultPalette.Get("revisions details renamed"),
		Selected: common.DefaultPalette.Get("revisions details selected"),
		Dimmed:   common.DefaultPalette.Get("revisions details dimmed"),
		Text:     common.DefaultPalette.Get("revisions details text"),
		Conflict: common.DefaultPalette.Get("revisions details conflict"),
	}

	l := NewDetailsList(s, common.NewSizeable(0, height))
	return &Model{
		revision:    revision,
		DetailsList: l,
		mode:        viewMode,
		context:     context,
		keyMap:      keyMap,
		styles:      s,
		height:      height,
	}
}

func (m *Model) Init() tea.Cmd {
	return m.load(m.revision.GetChangeId())
}

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case confirmation.CloseMsg:
		m.confirmation = nil
		m.selectedHint = ""
		m.unselectedHint = ""
		return m, nil
	case common.RefreshMsg:
		return m, m.load(m.revision.GetChangeId())
	case updateCommitStatusMsg:
		items := m.createListItems(msg.summary, msg.selectedFiles)
		var selectionChangedCmd tea.Cmd
		m.context.ClearCheckedItems(reflect.TypeFor[context.SelectedFile]())
		if len(items) > 0 {
			var first context.SelectedItem
			for _, it := range items {
				sel := context.SelectedFile{
					ChangeId: m.revision.GetChangeId(),
					CommitId: m.revision.CommitId,
					File:     it.fileName,
				}
				if first == nil {
					first = sel
				}
				if it.selected {
					m.context.AddCheckedItem(sel)
				}
			}
			selectionChangedCmd = m.context.SetSelectedItem(first)
		}
		m.setItems(items)
		return m, selectionChangedCmd
	default:
		oldCursor := m.cursor
		var cmd tea.Cmd
		var newModel *Model
		newModel, cmd = m.internalUpdate(msg)
		if m.cursor != oldCursor {
			cmd = tea.Batch(cmd, m.context.SetSelectedItem(context.SelectedFile{
				ChangeId: m.revision.GetChangeId(),
				CommitId: m.revision.CommitId,
				File:     m.current().fileName,
			}))
		}
		return newModel, cmd
	}
}

func (m *Model) internalUpdate(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.confirmation != nil {
			model, cmd := m.confirmation.Update(msg)
			m.confirmation = model
			return m, cmd
		}
		switch {
		case key.Matches(msg, m.keyMap.Up):
			m.cursorUp()
			return m, nil
		case key.Matches(msg, m.keyMap.Down):
			m.cursorDown()
			return m, nil
		case key.Matches(msg, m.keyMap.Cancel), key.Matches(msg, m.keyMap.Details.Close):
			return m, common.Close
		case key.Matches(msg, m.keyMap.Details.Diff):
			selected := m.current()
			if selected == nil {
				return m, nil
			}
			return m, func() tea.Msg {
				output, _ := m.context.RunCommandImmediate(jj.Diff(m.revision.GetChangeId(), selected.fileName))
				return common.ShowDiffMsg(output)
			}
		case key.Matches(msg, m.keyMap.Details.Split):
			selectedFiles := m.getSelectedFiles()
			m.selectedHint = "stays as is"
			m.unselectedHint = "moves to the new revision"
			model := confirmation.New(
				[]string{"Are you sure you want to split the selected files?"},
				confirmation.WithStylePrefix("revisions"),
				confirmation.WithOption("Yes",
					tea.Batch(m.context.RunInteractiveCommand(jj.Split(m.revision.GetChangeId(), selectedFiles), common.Refresh), common.Close),
					key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "yes"))),
				confirmation.WithOption("No",
					confirmation.Close,
					key.NewBinding(key.WithKeys("n", "esc"), key.WithHelp("n/esc", "no"))),
			)
			m.confirmation = model
			return m, m.confirmation.Init()
		case key.Matches(msg, m.keyMap.Details.Squash):
			m.mode = squashTargetMode
			return m, common.JumpToParent(m.revision)
		case key.Matches(msg, m.keyMap.Details.Restore):
			selectedFiles := m.getSelectedFiles()
			m.selectedHint = "gets restored"
			m.unselectedHint = "stays as is"
			model := confirmation.New(
				[]string{"Are you sure you want to restore the selected files?"},
				confirmation.WithStylePrefix("revisions"),
				confirmation.WithOption("Yes",
					m.context.RunCommand(jj.Restore(m.revision.GetChangeId(), selectedFiles), common.Refresh, confirmation.Close),
					key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "yes"))),
				confirmation.WithOption("No",
					confirmation.Close,
					key.NewBinding(key.WithKeys("n", "esc"), key.WithHelp("n/esc", "no"))),
			)
			m.confirmation = model
			return m, m.confirmation.Init()
		case key.Matches(msg, m.keyMap.Details.Absorb):
			selectedFiles := m.getSelectedFiles()
			m.selectedHint = "might get absorbed into parents"
			m.unselectedHint = "stays as is"
			model := confirmation.New(
				[]string{"Are you sure you want to absorb changes from the selected files?"},
				confirmation.WithStylePrefix("revisions"),
				confirmation.WithOption("Yes",
					m.context.RunCommand(jj.Absorb(m.revision.GetChangeId(), selectedFiles...), common.Refresh, confirmation.Close),
					key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "yes"))),
				confirmation.WithOption("No",
					confirmation.Close,
					key.NewBinding(key.WithKeys("n", "esc"), key.WithHelp("n/esc", "no"))),
			)
			m.confirmation = model
			return m, m.confirmation.Init()
		case key.Matches(msg, m.keyMap.Details.ToggleSelect):
			if current := m.current(); current != nil {
				isChecked := !current.selected
				current.selected = isChecked

				checkedFile := context.SelectedFile{
					ChangeId: m.revision.GetChangeId(),
					CommitId: m.revision.CommitId,
					File:     current.fileName,
				}
				if isChecked {
					m.context.AddCheckedItem(checkedFile)
				} else {
					m.context.RemoveCheckedItem(checkedFile)
				}

				m.cursorDown()
			}
			return m, nil
		case key.Matches(msg, m.keyMap.Details.RevisionsChangingFile):
			if current := m.current(); current != nil {
				return m, tea.Batch(common.Close, common.UpdateRevSet(fmt.Sprintf("files(%s)", jj.EscapeFileName(current.fileName))))
			}
		}
	}
	return m, nil
}

func (m *Model) createListItems(content string, selectedFiles []string) []*item {
	var items []*item
	scanner := bufio.NewScanner(strings.NewReader(content))
	var conflicts []bool
	if scanner.Scan() {
		conflictsLine := strings.Split(scanner.Text(), " ")
		for _, c := range conflictsLine {
			conflicts = append(conflicts, c == "true")
		}
	} else {
		return items
	}

	index := 0
	for scanner.Scan() {
		file := strings.TrimSpace(scanner.Text())
		if file == "" {
			continue
		}
		var status status
		switch file[0] {
		case 'A':
			status = Added
		case 'D':
			status = Deleted
		case 'M':
			status = Modified
		case 'R':
			status = Renamed
		}
		fileName := file[2:]

		actualFileName := fileName
		if status == Renamed && strings.Contains(actualFileName, "{") {
			for strings.Contains(actualFileName, "{") {
				start := strings.Index(actualFileName, "{")
				end := strings.Index(actualFileName, "}")
				if end == -1 {
					break
				}
				replacement := actualFileName[start+1 : end]
				parts := strings.Split(replacement, " => ")
				replacement = parts[1]
				actualFileName = path.Clean(actualFileName[:start] + replacement + actualFileName[end+1:])
			}
		}
		items = append(items, &item{
			status:   status,
			name:     fileName,
			fileName: actualFileName,
			selected: slices.ContainsFunc(selectedFiles, func(s string) bool { return s == actualFileName }),
			conflict: conflicts[index],
		})
		index++
	}
	return items
}

func (m *Model) getSelectedFiles() []string {
	selectedFiles := make([]string, 0)
	if len(m.files) == 0 {
		return selectedFiles
	}

	for _, f := range m.files {
		if f.selected {
			selectedFiles = append(selectedFiles, f.fileName)
		}
	}
	if len(selectedFiles) == 0 {
		selectedFiles = append(selectedFiles, m.current().fileName)
		return selectedFiles
	}
	return selectedFiles
}

func (m *Model) View() string {
	confirmationView := ""
	ch := 0
	if m.confirmation != nil {
		confirmationView = m.confirmation.View()
		ch = lipgloss.Height(confirmationView)
	}
	if m.Len() == 0 {
		return m.styles.Dimmed.Render("No changes\n")
	}
	m.SetHeight(min(m.height-5-ch, m.Len()))
	filesView := m.renderer.Render(m.cursor)

	view := lipgloss.JoinVertical(lipgloss.Top, filesView, confirmationView)
	// We are trimming spaces from each line to prevent visual artefacts
	// Empty lines use the default background colour, and it looks bad if the user has a custom background colour
	var lines []string
	scanner := bufio.NewScanner(strings.NewReader(view))
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		lines = append(lines, line)
	}
	view = strings.Join(lines, "\n")
	w, h := lipgloss.Size(view)
	return lipgloss.Place(w, h, 0, 0, view, lipgloss.WithWhitespaceBackground(m.styles.Text.GetBackground()))
}

func (m *Model) load(revision string) tea.Cmd {
	output, err := m.context.RunCommandImmediate(jj.Snapshot())
	if err == nil {
		output, err = m.context.RunCommandImmediate(jj.Status(revision))
		if err == nil {
			return func() tea.Msg {
				summary := string(output)
				selectedFiles := m.getSelectedFiles()
				return updateCommitStatusMsg{summary, selectedFiles}
			}
		}
	}
	return func() tea.Msg {
		return common.CommandCompletedMsg{
			Output: string(output),
			Err:    err,
		}
	}
}
