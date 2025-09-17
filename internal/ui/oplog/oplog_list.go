package oplog

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/models"
	"github.com/idursun/jjui/internal/ui/common/list"
)

var _ list.IList = (*OpLogList)(nil)

type OpLogList struct {
	*list.List[*models.OperationLogItem]
	selectedStyle lipgloss.Style
	textStyle     lipgloss.Style
}

var _ list.IItemRenderer = (*itemRenderer)(nil)

type itemRenderer struct {
	row   *models.OperationLogItem
	style lipgloss.Style
}

func (i itemRenderer) Render(w io.Writer, width int) {
	row := i.row

	for _, rowLine := range row.Lines {
		lw := strings.Builder{}
		for _, segment := range rowLine.Segments {
			fmt.Fprint(&lw, segment.Style.Inherit(i.style).Render(segment.Text))
		}
		line := lw.String()
		fmt.Fprint(w, lipgloss.PlaceHorizontal(width, 0, line, lipgloss.WithWhitespaceBackground(i.style.GetBackground())))
		fmt.Fprint(w, "\n")
	}
}

func (i itemRenderer) Height() int {
	return len(i.row.Lines)
}

func (o *OpLogList) Len() int {
	return len(o.Items)
}

func (o *OpLogList) GetRenderer(index int) list.IItemRenderer {
	item := o.Items[index]
	style := o.textStyle
	if index == o.Cursor {
		style = o.selectedStyle
	}
	return &itemRenderer{
		row:   item,
		style: style,
	}
}
