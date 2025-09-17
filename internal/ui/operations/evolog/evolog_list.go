package evolog

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/models"
	"github.com/idursun/jjui/internal/ui/common/list"
)

var _ list.IListProvider = (*EvologList)(nil)
var _ list.IList = (*EvologList)(nil)

type EvologList struct {
	*list.List[*models.RevisionItem]
	selectedStyle lipgloss.Style
	textStyle     lipgloss.Style
	dimmedStyle   lipgloss.Style
	commitIdStyle lipgloss.Style
	changeIdStyle lipgloss.Style
	markerStyle   lipgloss.Style
}

var _ list.IItemRenderer = (*renderer)(nil)

type renderer struct {
	row           *models.RevisionItem
	styleOverride lipgloss.Style
	width         int
}

func (r renderer) Render(w io.Writer, width int) {
	row := r.row
	for lineIndex := 0; lineIndex < len(row.Lines); lineIndex++ {
		segmentedLine := row.Lines[lineIndex]

		lw := strings.Builder{}
		for _, segment := range segmentedLine.Gutter.Segments {
			style := segment.Style
			fmt.Fprint(&lw, style.Render(segment.Text))
		}

		for _, segment := range segmentedLine.Segments {
			style := segment.Style.Inherit(r.styleOverride)
			fmt.Fprint(&lw, style.Render(segment.Text))
		}
		line := lw.String()
		fmt.Fprint(w, lipgloss.PlaceHorizontal(r.width, 0, line, lipgloss.WithWhitespaceBackground(r.styleOverride.GetBackground())))
		fmt.Fprint(w, "\n")
	}
}

func (r renderer) Height() int {
	return len(r.row.Lines)
}

func (e *EvologList) Len() int {
	return len(e.Items)
}

func (e *EvologList) GetRenderer(index int) list.IItemRenderer {
	row := e.Items[index]
	selected := index == e.Cursor
	styleOverride := e.textStyle
	if selected {
		styleOverride = e.selectedStyle
	}
	return &renderer{
		row:           row,
		styleOverride: styleOverride,
		//TODO: fix this
		width: 50,
	}
}

func (e *EvologList) CurrentItem() models.IItem {
	return e.Current()
}

func (e *EvologList) CheckedItems() []models.IItem {
	return nil
}
