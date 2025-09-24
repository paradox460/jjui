package revisions

import (
	"bytes"
	"fmt"
	"log"
	"reflect"
	"slices"
	"strings"
	"sync/atomic"

	"github.com/idursun/jjui/internal/config/script"
	"github.com/idursun/jjui/internal/ui/common/list"
	"github.com/idursun/jjui/internal/ui/operations/ace_jump"
	"github.com/idursun/jjui/internal/ui/operations/duplicate"
	"github.com/idursun/jjui/internal/ui/operations/revert"
	"github.com/idursun/jjui/internal/ui/operations/set_parents"

	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/ui/operations/describe"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	appContext "github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/graph"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/operations/abandon"
	"github.com/idursun/jjui/internal/ui/operations/bookmark"
	"github.com/idursun/jjui/internal/ui/operations/details"
	"github.com/idursun/jjui/internal/ui/operations/evolog"
	"github.com/idursun/jjui/internal/ui/operations/rebase"
	"github.com/idursun/jjui/internal/ui/operations/squash"
)

var _ list.IList = (*Model)(nil)
var _ list.IListCursor = (*Model)(nil)
var _ common.Focusable = (*Model)(nil)
var _ common.Editable = (*Model)(nil)

type Model struct {
	*common.Sizeable
	rows             []parser.Row
	tag              atomic.Uint64
	revisionToSelect string
	offScreenRows    []parser.Row
	streamer         *graph.GraphStreamer
	hasMore          bool
	op               tea.Model
	cursor           int
	context          *appContext.MainContext
	keymap           config.KeyMappings[key.Binding]
	output           string
	err              error
	quickSearch      string
	previousOpLogId  string
	isLoading        bool
	renderer         *revisionListRenderer
	textStyle        lipgloss.Style
	dimmedStyle      lipgloss.Style
	selectedStyle    lipgloss.Style
	waiter           chan tea.Msg
}

func (m *Model) Cursor() int {
	return m.cursor
}

func (m *Model) SetCursor(index int) {
	if index >= 0 && index < len(m.rows) {
		m.cursor = index
	}
}

func (m *Model) Len() int {
	return len(m.rows)
}

func (m *Model) GetItemRenderer(index int) list.IItemRenderer {
	var (
		before, after, renderOverDescription, beforeCommitId, beforeChangeId string
	)
	row := m.rows[index]
	inLane := m.renderer.tracer.IsInSameLane(index)
	isHighlighted := index == m.cursor

	if op, ok := m.op.(operations.Operation); ok {
		before = op.Render(row.Commit, operations.RenderPositionBefore)
		after = op.Render(row.Commit, operations.RenderPositionAfter)
		renderOverDescription = ""
		if isHighlighted {
			renderOverDescription = op.Render(row.Commit, operations.RenderOverDescription)
		}
		beforeCommitId = op.Render(row.Commit, operations.RenderBeforeCommitId)
		beforeChangeId = op.Render(row.Commit, operations.RenderBeforeChangeId)
	}

	return &itemRenderer{
		row:            row,
		before:         before,
		after:          after,
		description:    renderOverDescription,
		beforeChangeId: beforeChangeId,
		beforeCommitId: beforeCommitId,
		isHighlighted:  isHighlighted,
		SearchText:     m.quickSearch,
		textStyle:      m.textStyle,
		dimmedStyle:    m.dimmedStyle,
		selectedStyle:  m.selectedStyle,
		isChecked:      m.renderer.selections[row.Commit.GetChangeId()],
		isGutterInLane: func(lineIndex, segmentIndex int) bool {
			return m.renderer.tracer.IsGutterInLane(index, lineIndex, segmentIndex)
		},
		updateGutterText: func(lineIndex, segmentIndex int, text string) string {
			return m.renderer.tracer.UpdateGutterText(index, lineIndex, segmentIndex, text)
		},
		inLane: inLane,
		op:     m.op.(operations.Operation),
	}
}

type revisionsMsg struct {
	msg tea.Msg
}

// Allow a message to be targetted to this component.
func RevisionsCmd(msg tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return revisionsMsg{msg: msg}
	}
}

type updateRevisionsMsg struct {
	rows             []parser.Row
	selectedRevision string
}

type startRowsStreamingMsg struct {
	selectedRevision string
	tag              uint64
}

type appendRowsBatchMsg struct {
	rows    []parser.Row
	hasMore bool
	tag     uint64
}

func (m *Model) IsEditing() bool {
	if f, ok := m.op.(common.Editable); ok {
		return f.IsEditing()
	}
	return false
}

func (m *Model) IsFocused() bool {
	if f, ok := m.op.(common.Focusable); ok {
		return f.IsFocused()
	}
	return false
}

func (m *Model) InNormalMode() bool {
	if _, ok := m.op.(*operations.Default); ok {
		return true
	}
	return false
}

func (m *Model) ShortHelp() []key.Binding {
	if op, ok := m.op.(help.KeyMap); ok {
		return op.ShortHelp()
	}
	return (&operations.Default{}).ShortHelp()
}

func (m *Model) FullHelp() [][]key.Binding {
	if op, ok := m.op.(help.KeyMap); ok {
		return op.FullHelp()
	}
	return [][]key.Binding{m.ShortHelp()}
}

func (m *Model) SelectedRevision() *jj.Commit {
	if m.cursor >= len(m.rows) || m.cursor < 0 {
		return nil
	}
	return m.rows[m.cursor].Commit
}

func (m *Model) SelectedRevisions() jj.SelectedRevisions {
	var selected []*jj.Commit
	ids := make(map[string]bool)
	for _, ci := range m.context.CheckedItems {
		if rev, ok := ci.(appContext.SelectedRevision); ok {
			ids[rev.CommitId] = true
		}
	}
	for _, row := range m.rows {
		if _, ok := ids[row.Commit.CommitId]; ok {
			selected = append(selected, row.Commit)
		}
	}

	if len(selected) == 0 {
		return jj.NewSelectedRevisions(m.SelectedRevision())
	}
	return jj.NewSelectedRevisions(selected...)
}

func (m *Model) Init() tea.Cmd {
	return common.RefreshAndSelect("@")
}

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	if k, ok := msg.(revisionsMsg); ok {
		msg = k.msg
	}

	if msg, ok := msg.(script.ResumeScriptExecutionMsg); ok {
		switch step := msg.Execution.Current().(type) {
		case *script.UIStep:
			if step.UI.Action == "inline_describe" {
				var cmd tea.Cmd
				m.waiter, cmd = msg.Execution.Wait(10)
				m.op = describe.NewOperation(m.context, m.SelectedRevision().GetChangeId(), m.Width)
				return m, tea.Batch(m.op.Init(), cmd)
			}
		}
	}

	var cmd tea.Cmd
	var nm *Model
	nm, cmd = m.internalUpdate(msg)

	if curSelected := m.SelectedRevision(); curSelected != nil {
		if op, ok := m.op.(operations.TracksSelectedRevision); ok {
			op.SetSelectedRevision(curSelected)
		}
	}

	return nm, cmd
}

func (m *Model) internalUpdate(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case common.CloseViewMsg:
		m.op = operations.NewDefault()
		if m.waiter != nil {
			m.waiter <- "continue"
			close(m.waiter)
			m.waiter = nil
		}
		return m, m.updateSelection()
	case common.QuickSearchMsg:
		m.quickSearch = string(msg)
		m.cursor = m.search(0)
		m.op = operations.NewDefault()
		m.renderer.Reset()
		return m, nil
	case common.CommandCompletedMsg:
		m.output = msg.Output
		m.err = msg.Err
		return m, nil
	case common.AutoRefreshMsg:
		id, _ := m.context.RunCommandImmediate(jj.OpLogId(true))
		currentOperationId := string(id)
		log.Println("Previous operation ID:", m.previousOpLogId, "Current operation ID:", currentOperationId)
		if currentOperationId != m.previousOpLogId {
			m.previousOpLogId = currentOperationId
			return m, common.RefreshAndKeepSelections
		}
	case common.RefreshMsg:
		if !msg.KeepSelections {
			m.context.ClearCheckedItems(reflect.TypeFor[appContext.SelectedRevision]())
		}
		m.isLoading = true
		var cmd tea.Cmd
		m.op, cmd = m.op.Update(msg)
		if config.Current.Revisions.LogBatching {
			currentTag := m.tag.Add(1)
			return m, tea.Batch(m.loadStreaming(m.context.CurrentRevset, msg.SelectedRevision, currentTag), cmd)
		} else {
			return m, tea.Batch(m.load(m.context.CurrentRevset, msg.SelectedRevision), cmd)
		}
	case updateRevisionsMsg:
		m.isLoading = false
		m.updateGraphRows(msg.rows, msg.selectedRevision)
		return m, tea.Batch(m.highlightChanges, m.updateSelection(), func() tea.Msg {
			return common.UpdateRevisionsSuccessMsg{}
		})
	case startRowsStreamingMsg:
		m.offScreenRows = nil
		m.revisionToSelect = msg.selectedRevision

		// If the revision to select is not set, use the currently selected item
		if m.revisionToSelect == "" {
			switch selected := m.context.SelectedItem.(type) {
			case appContext.SelectedRevision:
				m.revisionToSelect = selected.CommitId
			case appContext.SelectedFile:
				m.revisionToSelect = selected.CommitId
			}
		}
		log.Println("Starting streaming revisions message received with tag:", msg.tag, "revision to select:", msg.selectedRevision)
		return m, m.requestMoreRows(msg.tag)
	case appendRowsBatchMsg:
		if msg.tag != m.tag.Load() {
			return m, nil
		}
		m.offScreenRows = append(m.offScreenRows, msg.rows...)
		m.hasMore = msg.hasMore
		m.isLoading = m.hasMore && len(m.offScreenRows) > 0

		if m.hasMore {
			// keep requesting rows until we reach the initial load count or the current cursor position
			if len(m.offScreenRows) < m.cursor+1 || len(m.offScreenRows) < m.renderer.ViewRange.LastRowIndex+1 {
				return m, m.requestMoreRows(msg.tag)
			}
		} else if m.streamer != nil {
			m.streamer.Close()
		}

		currentSelectedRevision := m.SelectedRevision()
		m.rows = m.offScreenRows
		if m.revisionToSelect != "" {
			m.cursor = m.selectRevision(m.revisionToSelect)
			m.revisionToSelect = ""
		}

		if m.cursor == -1 && currentSelectedRevision != nil {
			m.cursor = m.selectRevision(currentSelectedRevision.GetChangeId())
		}

		if (m.cursor < 0 || m.cursor >= len(m.rows)) && len(m.rows) > 0 {
			m.cursor = 0
		}

		cmds := []tea.Cmd{m.highlightChanges, m.updateSelection()}
		if !m.hasMore {
			cmds = append(cmds, func() tea.Msg {
				return common.UpdateRevisionsSuccessMsg{}
			})
		}
		return m, tea.Batch(cmds...)
	case common.StartSquashOperationMsg:
		return m.startSquash(jj.NewSelectedRevisions(msg.Revision), msg.Files)
	}

	if len(m.rows) == 0 {
		return m, nil
	}

	if op, ok := m.op.(common.Editable); ok && op.IsEditing() {
		var cmd tea.Cmd
		m.op, cmd = m.op.Update(msg)
		return m, cmd
	}

	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.Up):
			if m.cursor > 0 {
				m.cursor--
			}
			return m, m.updateSelection()
		case key.Matches(msg, m.keymap.Down):
			if m.cursor < len(m.rows)-1 {
				m.cursor++
			} else if m.hasMore {
				return m, m.requestMoreRows(m.tag.Load())
			}
			return m, m.updateSelection()
		case key.Matches(msg, m.keymap.JumpToParent):
			m.jumpToParent(m.SelectedRevisions())
			return m, m.updateSelection()
		case key.Matches(msg, m.keymap.JumpToChildren):
			immediate, _ := m.context.RunCommandImmediate(jj.GetFirstChild(m.SelectedRevision()))
			index := m.selectRevision(string(immediate))
			if index != -1 {
				m.cursor = index
			}
			return m, m.updateSelection()
		case key.Matches(msg, m.keymap.JumpToWorkingCopy):
			workingCopyIndex := m.selectRevision("@")
			if workingCopyIndex != -1 {
				m.cursor = workingCopyIndex
			}
			return m, m.updateSelection()
		case key.Matches(msg, m.keymap.AceJump):
			op := ace_jump.NewOperation(m, func(index int) parser.Row {
				return m.rows[index]
			}, m.renderer.FirstRowIndex, m.renderer.LastRowIndex)
			m.op = op
			return m, op.Init()
		default:
			if op, ok := m.op.(common.Focusable); ok && op.IsFocused() {
				m.op, cmd = m.op.Update(msg)
				break
			}

			switch {
			case key.Matches(msg, m.keymap.ToggleSelect):
				commit := m.rows[m.cursor].Commit
				changeId := commit.GetChangeId()
				item := appContext.SelectedRevision{ChangeId: changeId, CommitId: commit.CommitId}
				m.context.ToggleCheckedItem(item)
				immediate, _ := m.context.RunCommandImmediate(jj.GetParent(jj.NewSelectedRevisions(commit)))
				parentIndex := m.selectRevision(string(immediate))
				if parentIndex != -1 {
					m.cursor = parentIndex
				}
			case key.Matches(msg, m.keymap.Cancel):
				m.op = operations.NewDefault()
			case key.Matches(msg, m.keymap.QuickSearchCycle):
				m.cursor = m.search(m.cursor + 1)
				m.renderer.Reset()
				return m, nil
			case key.Matches(msg, m.keymap.Details.Mode):
				m.op = details.NewOperation(m.context, m.SelectedRevision(), m.Height)
				return m, m.op.Init()
			case key.Matches(msg, m.keymap.InlineDescribe.Mode):
				m.op = describe.NewOperation(m.context, m.SelectedRevision().GetChangeId(), m.Width)
				return m, m.op.Init()
			case key.Matches(msg, m.keymap.New):
				return m, m.context.RunCommand(jj.New(m.SelectedRevisions()), common.RefreshAndSelect("@"))
			case key.Matches(msg, m.keymap.Commit):
				return m, m.context.RunInteractiveCommand(jj.CommitWorkingCopy(), common.Refresh)
			case key.Matches(msg, m.keymap.Edit, m.keymap.ForceEdit):
				ignoreImmutable := key.Matches(msg, m.keymap.ForceEdit)
				return m, m.context.RunCommand(jj.Edit(m.SelectedRevision().GetChangeId(), ignoreImmutable), common.Refresh)
			case key.Matches(msg, m.keymap.Diffedit):
				changeId := m.SelectedRevision().GetChangeId()
				return m, m.context.RunInteractiveCommand(jj.DiffEdit(changeId), common.Refresh)
			case key.Matches(msg, m.keymap.Absorb):
				changeId := m.SelectedRevision().GetChangeId()
				return m, m.context.RunCommand(jj.Absorb(changeId), common.Refresh)
			case key.Matches(msg, m.keymap.Abandon):
				selections := m.SelectedRevisions()
				m.op = abandon.NewOperation(m.context, selections)
				return m, m.op.Init()
			case key.Matches(msg, m.keymap.Bookmark.Set):
				m.op = bookmark.NewSetBookmarkOperation(m.context, m.SelectedRevision().GetChangeId())
				return m, m.op.Init()
			case key.Matches(msg, m.keymap.Split):
				currentRevision := m.SelectedRevision().GetChangeId()
				return m, m.context.RunInteractiveCommand(jj.Split(currentRevision, []string{}), common.Refresh)
			case key.Matches(msg, m.keymap.Describe):
				selections := m.SelectedRevisions()
				return m, m.context.RunInteractiveCommand(jj.Describe(selections), common.Refresh)
			case key.Matches(msg, m.keymap.Evolog.Mode):
				m.op = evolog.NewOperation(m.context, m.SelectedRevision(), m.Width, m.Height)
				return m, m.op.Init()
			case key.Matches(msg, m.keymap.Diff):
				return m, func() tea.Msg {
					changeId := m.SelectedRevision().GetChangeId()
					output, _ := m.context.RunCommandImmediate(jj.Diff(changeId, ""))
					return common.ShowDiffMsg(output)
				}
			case key.Matches(msg, m.keymap.Refresh):
				return m, common.Refresh
			case key.Matches(msg, m.keymap.Squash.Mode):
				return m.startSquash(m.SelectedRevisions(), nil)
			case key.Matches(msg, m.keymap.Revert.Mode):
				m.op = revert.NewOperation(m.context, m.SelectedRevisions(), revert.TargetDestination)
				return m, m.op.Init()
			case key.Matches(msg, m.keymap.Rebase.Mode):
				m.op = rebase.NewOperation(m.context, m.SelectedRevisions(), rebase.SourceRevision, rebase.TargetDestination)
				return m, m.op.Init()
			case key.Matches(msg, m.keymap.Duplicate.Mode):
				m.op = duplicate.NewOperation(m.context, m.SelectedRevisions(), duplicate.TargetDestination)
				return m, m.op.Init()
			case key.Matches(msg, m.keymap.SetParents):
				m.op = set_parents.NewModel(m.context, m.SelectedRevision())
				return m, m.op.Init()
			}
		}
	}

	return m, cmd
}

func (m *Model) startSquash(selectedRevisions jj.SelectedRevisions, files []string) (*Model, tea.Cmd) {
	parent, _ := m.context.RunCommandImmediate(jj.GetParent(selectedRevisions))
	parentIdx := m.selectRevision(string(parent))
	if parentIdx != -1 {
		m.cursor = parentIdx
	} else if m.cursor < len(m.rows)-1 {
		m.cursor++
	}
	m.op = squash.NewOperation(m.context, selectedRevisions, squash.WithFiles(files))
	return m, m.op.Init()
}

func (m *Model) updateSelection() tea.Cmd {
	if selectedRevision := m.SelectedRevision(); selectedRevision != nil {
		return m.context.SetSelectedItem(appContext.SelectedRevision{
			ChangeId: selectedRevision.GetChangeId(),
			CommitId: selectedRevision.CommitId,
		})
	}
	return nil
}

func (m *Model) highlightChanges() tea.Msg {
	if m.err != nil || m.output == "" {
		return nil
	}

	changes := strings.Split(m.output, "\n")
	for _, change := range changes {
		if !strings.HasPrefix(change, " ") {
			continue
		}
		line := strings.Trim(change, "\n ")
		if line == "" {
			continue
		}
		parts := strings.Split(line, " ")
		if len(parts) > 0 {
			for i := range m.rows {
				row := &m.rows[i]
				if row.Commit.GetChangeId() == parts[0] {
					row.IsAffected = true
					break
				}
			}
		}
	}
	return nil
}

func (m *Model) updateGraphRows(rows []parser.Row, selectedRevision string) {
	if rows == nil {
		rows = []parser.Row{}
	}

	currentSelectedRevision := selectedRevision
	if cur := m.SelectedRevision(); currentSelectedRevision == "" && cur != nil {
		currentSelectedRevision = cur.GetChangeId()
	}
	m.rows = rows

	if len(m.rows) > 0 {
		m.cursor = m.selectRevision(currentSelectedRevision)
		if m.cursor == -1 {
			m.cursor = m.selectRevision("@")
		}
		if m.cursor == -1 {
			m.cursor = 0
		}
	} else {
		m.cursor = 0
	}
}

func (m *Model) View() string {
	if len(m.rows) == 0 {
		if m.isLoading {
			return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, "loading")
		}
		return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, "(no matching revisions)")
	}

	if config.Current.UI.Tracer.Enabled {
		start, end := m.renderer.FirstRowIndex, m.renderer.LastRowIndex+1 // +1 because the last row is inclusive in the view range
		log.Println("Visible row range:", start, end, "Cursor:", m.cursor, "Total rows:", len(m.rows))
		m.renderer.tracer = parser.NewTracer(m.rows, m.cursor, start, end)
	} else {
		m.renderer.tracer = parser.NewNoopTracer()
	}

	m.renderer.selections = m.context.GetSelectedRevisions()

	output := m.renderer.Render(m.cursor)
	output = m.textStyle.MaxWidth(m.Width).Render(output)
	return lipgloss.Place(m.Width, m.Height, 0, 0, output)
}

func (m *Model) load(revset string, selectedRevision string) tea.Cmd {
	return func() tea.Msg {
		output, err := m.context.RunCommandImmediate(jj.Log(revset, config.Current.Limit))
		if err != nil {
			return common.UpdateRevisionsFailedMsg{
				Err:    err,
				Output: string(output),
			}
		}
		rows := parser.ParseRows(bytes.NewReader(output))
		return updateRevisionsMsg{rows, selectedRevision}
	}
}

func (m *Model) loadStreaming(revset string, selectedRevision string, tag uint64) tea.Cmd {
	if m.tag.Load() != tag {
		return nil
	}

	if m.streamer != nil {
		m.streamer.Close()
		m.streamer = nil
	}

	m.hasMore = false

	var notifyErrorCmd tea.Cmd
	streamer, err := graph.NewGraphStreamer(m.context, revset)
	if err != nil {
		notifyErrorCmd = func() tea.Msg {
			return common.UpdateRevisionsFailedMsg{
				Err:    err,
				Output: fmt.Sprintf("%v", err),
			}
		}
	}
	m.streamer = streamer
	m.hasMore = true
	m.offScreenRows = nil
	log.Println("Starting streaming revisions with tag:", tag)
	startStreamingCmd := func() tea.Msg {
		return startRowsStreamingMsg{selectedRevision, tag}
	}

	return tea.Batch(startStreamingCmd, notifyErrorCmd)
}

func (m *Model) requestMoreRows(tag uint64) tea.Cmd {
	return func() tea.Msg {
		if m.streamer == nil || !m.hasMore {
			return nil
		}
		if tag == m.tag.Load() {
			batch := m.streamer.RequestMore()
			return appendRowsBatchMsg{batch.Rows, batch.HasMore, tag}
		}
		return nil
	}
}

func (m *Model) selectRevision(revision string) int {
	eqFold := func(other string) bool {
		return strings.EqualFold(other, revision)
	}

	idx := slices.IndexFunc(m.rows, func(row parser.Row) bool {
		if revision == "@" {
			return row.Commit.IsWorkingCopy
		}
		return eqFold(row.Commit.GetChangeId()) || eqFold(row.Commit.ChangeId) || eqFold(row.Commit.CommitId)
	})
	return idx
}

func (m *Model) search(startIndex int) int {
	if m.quickSearch == "" {
		return m.cursor
	}

	n := len(m.rows)
	for i := startIndex; i < n+startIndex; i++ {
		c := i % n
		row := &m.rows[c]
		for _, line := range row.Lines {
			for _, segment := range line.Segments {
				if segment.Text != "" && strings.Contains(segment.Text, m.quickSearch) {
					return c
				}
			}
		}
	}
	return m.cursor
}

func (m *Model) CurrentOperation() operations.Operation {
	return m.op.(operations.Operation)
}

func (m *Model) GetCommitIds() []string {
	var commitIds []string
	for _, row := range m.rows {
		commitIds = append(commitIds, row.Commit.CommitId)
	}
	return commitIds
}

func New(c *appContext.MainContext) *Model {
	keymap := config.Current.GetKeyMap()
	m := Model{
		Sizeable:      &common.Sizeable{Width: 0, Height: 0},
		context:       c,
		keymap:        keymap,
		rows:          nil,
		offScreenRows: nil,
		op:            operations.NewDefault(),
		cursor:        0,
		textStyle:     common.DefaultPalette.Get("revisions text"),
		dimmedStyle:   common.DefaultPalette.Get("revisions dimmed"),
		selectedStyle: common.DefaultPalette.Get("revisions selected"),
	}
	m.renderer = newRevisionListRenderer(&m, m.Sizeable)
	return &m
}

func (m *Model) jumpToParent(revisions jj.SelectedRevisions) {
	immediate, _ := m.context.RunCommandImmediate(jj.GetParent(revisions))
	parentIndex := m.selectRevision(string(immediate))
	if parentIndex != -1 {
		m.cursor = parentIndex
	}
}
