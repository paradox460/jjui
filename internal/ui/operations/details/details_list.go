package details

import (
	"fmt"
	"io"

	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/models"
	"github.com/idursun/jjui/internal/ui/common/list"
)

var _ list.IList = (*DetailsList)(nil)

type DetailsList struct {
	*list.CheckableList[*models.RevisionFileItem]
	renderer            *list.ListRenderer
	selectedHint        string
	unselectedHint      string
	isVirtuallySelected bool
	styles              styles
}

var _ list.IItemRenderer = (*itemRenderer)(nil)

type itemRenderer struct {
	item                *models.RevisionFileItem
	styles              styles
	style               lipgloss.Style
	selectedHint        string
	unselectedHint      string
	isChecked           bool
	isVirtuallySelected bool
	hint                string
}

func (i itemRenderer) showHint() bool {
	return i.selectedHint != "" || i.unselectedHint != ""
}

func (i itemRenderer) Render(w io.Writer, _ int) {
	title := i.item.Title()
	if i.item.IsChecked() {
		title = "✓" + title
	} else {
		title = " " + title
	}

	_, _ = fmt.Fprint(w, i.style.PaddingRight(1).Render(title))
	if i.item.Conflict {
		_, _ = fmt.Fprint(w, i.styles.Conflict.Render("conflict "))
	}
	if i.hint != "" {
		_, _ = fmt.Fprint(w, i.styles.Dimmed.Render(i.hint))
	}
	_, _ = fmt.Fprintln(w)
}

func (i itemRenderer) Height() int {
	return 1
}

func (d *DetailsList) Len() int {
	return len(d.Items)
}

func (d *DetailsList) GetRenderer(index int) list.IItemRenderer {
	item := d.Items[index]
	var style lipgloss.Style
	switch item.Status {
	case models.Added:
		style = d.styles.Added
	case models.Deleted:
		style = d.styles.Deleted
	case models.Modified:
		style = d.styles.Modified
	case models.Renamed:
		style = d.styles.Renamed
	}

	if index == d.Cursor {
		style = style.Bold(true).Background(d.styles.Selected.GetBackground())
	} else {
		style = style.Background(d.styles.Text.GetBackground())
	}

	//title := ""
	hint := ""
	if d.showHint() {
		hint = d.unselectedHint
		if item.IsChecked() || (d.isVirtuallySelected && index == d.Cursor) {
			hint = d.selectedHint
			//title = "✓" + item.Title()
		}
	}
	r := itemRenderer{
		item:   item,
		styles: d.styles,
		style:  style,
		hint:   hint,
	}
	return r
}

func (d *DetailsList) showHint() bool {
	return d.selectedHint != "" || d.unselectedHint != ""
}
