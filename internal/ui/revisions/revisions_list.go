package revisions

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/models"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/ui/common/list"
	"github.com/idursun/jjui/internal/ui/operations"
)

var _ list.IListProvider = (*RevisionList)(nil)
var _ list.IList = (*RevisionList)(nil)

type RevisionList struct {
	*list.CheckableList[*models.RevisionItem]
	Tracer        parser.LaneTracer
	getOpFn       func() operations.Operation
	dimmedStyle   lipgloss.Style
	checkStyle    lipgloss.Style
	textStyle     lipgloss.Style
	selectedStyle lipgloss.Style
}

func (r *RevisionList) Len() int {
	return len(r.Items)
}

func (r *RevisionList) GetRenderer(index int) list.IItemRenderer {
	row := r.Items[index]
	inLane := r.Tracer.IsInSameLane(index)
	isHighlighted := index == r.Cursor
	before := r.RenderBefore(row.Commit)
	after := r.RenderAfter(row.Commit)
	renderOverDescription := ""
	op := r.getOpFn()
	if isHighlighted {
		renderOverDescription = op.Render(row.Commit, operations.RenderOverDescription)
	}
	beforeCommitId := op.Render(row.Commit, operations.RenderBeforeCommitId)
	beforeChangeId := op.Render(row.Commit, operations.RenderBeforeChangeId)
	return &itemRenderer{
		row:            row,
		before:         before,
		after:          after,
		description:    renderOverDescription,
		beforeChangeId: beforeChangeId,
		beforeCommitId: beforeCommitId,
		isHighlighted:  isHighlighted,
		textStyle:      r.textStyle,
		dimmedStyle:    r.dimmedStyle,
		selectedStyle:  r.selectedStyle,
		isGutterInLane: func(lineIndex, segmentIndex int) bool {
			return r.Tracer.IsGutterInLane(index, lineIndex, segmentIndex)
		},
		updateGutterText: func(lineIndex, segmentIndex int, text string) string {
			return r.Tracer.UpdateGutterText(index, lineIndex, segmentIndex, text)
		},
		inLane: inLane,
		op:     r.getOpFn(),
	}
}

func (r *RevisionList) CurrentItem() models.IItem {
	return r.Current()
}

func (r *RevisionList) CheckedItems() []models.IItem {
	checkedItems := r.GetCheckedItems()
	var items []models.IItem
	for _, item := range checkedItems {
		items = append(items, item)
	}
	return items
}

func (r *RevisionList) GetItemHeight(index int) int {
	return len(r.Items[index].Lines)
}

func (r *RevisionList) RenderBefore(commit *models.Commit) string {
	return r.getOpFn().Render(commit, operations.RenderPositionBefore)
}

func (r *RevisionList) RenderAfter(commit *models.Commit) string {
	return r.getOpFn().Render(commit, operations.RenderPositionAfter)
}

func (r *RevisionList) RenderBeforeChangeId(index int, item *models.RevisionItem) string {
	commit := item.Commit
	isSelected := item.IsChecked()
	isHighlighted := r.Cursor == index
	opMarker := r.getOpFn().Render(commit, operations.RenderBeforeChangeId)
	selectedMarker := ""
	if isSelected {
		if isHighlighted {
			selectedMarker = r.checkStyle.Background(r.selectedStyle.GetBackground()).Render("✓")
		} else {
			selectedMarker = r.checkStyle.Background(r.textStyle.GetBackground()).Render("✓")
		}
	}
	if opMarker == "" && selectedMarker == "" {
		return ""
	}
	var sections []string

	space := r.textStyle.Render(" ")
	if isHighlighted {
		space = r.selectedStyle.Render(" ")
	}

	if opMarker != "" {
		sections = append(sections, opMarker, space)
	}
	if selectedMarker != "" {
		sections = append(sections, selectedMarker, space)
	}
	return lipgloss.JoinHorizontal(0, sections...)
}

func (r *RevisionList) RenderBeforeCommitId(commit *models.Commit) string {
	return r.getOpFn().Render(commit, operations.RenderBeforeCommitId)
}
