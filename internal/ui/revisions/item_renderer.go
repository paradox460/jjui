package revisions

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/models"
	"github.com/idursun/jjui/internal/ui/common/list"
	"github.com/idursun/jjui/internal/ui/operations"
)

var _ list.IItemRenderer = (*itemRenderer)(nil)

type itemRenderer struct {
	row              *models.RevisionItem
	before           string
	after            string
	description      string
	beforeChangeId   string
	beforeCommitId   string
	isHighlighted    bool
	selectedStyle    lipgloss.Style
	textStyle        lipgloss.Style
	dimmedStyle      lipgloss.Style
	isGutterInLane   func(lineIndex, segmentIndex int) bool
	updateGutterText func(lineIndex, segmentIndex int, text string) string
	inLane           bool
	op               operations.Operation
}

func (ir itemRenderer) writeSection(w io.Writer, current models.GraphGutter, extended models.GraphGutter, highlight bool, section string, width int) {
	isHighlighted := ir.isHighlighted
	lines := strings.Split(section, "\n")
	for _, sectionLine := range lines {
		lw := strings.Builder{}
		for _, segment := range current.Segments {
			fmt.Fprint(&lw, segment.Style.Inherit(ir.textStyle).Render(segment.Text))
		}

		fmt.Fprint(&lw, sectionLine)
		line := lw.String()
		if isHighlighted && highlight {
			fmt.Fprint(w, lipgloss.PlaceHorizontal(width, 0, line, lipgloss.WithWhitespaceBackground(ir.selectedStyle.GetBackground())))
		} else {
			fmt.Fprint(w, lipgloss.PlaceHorizontal(width, 0, line, lipgloss.WithWhitespaceBackground(ir.textStyle.GetBackground())))
		}
		fmt.Fprintln(w)
		current = extended
	}
}

func (ir itemRenderer) Render(w io.Writer, width int) {
	row := ir.row
	isHighlighted := ir.isHighlighted

	// render: before
	if ir.before != "" {
		extended := models.GraphGutter{}
		if row.Previous != nil {
			extended = row.Previous.Extend()
		}
		ir.writeSection(w, extended, extended, false, ir.before, width)
	}

	// render: description overlay
	requiresDescriptionRendering := ir.description != ""
	descriptionRendered := false

	// Each line has a flag:
	// Revision: the line contains a change id and commit id (which is assumed to be the first line of the row)
	// Highlightable: the line can be highlighted (e.g. revision line and description line)
	// Elided: this is usually the last line of the row, it is not highlightable
	for lineIndex := 0; lineIndex < len(row.Lines); lineIndex++ {
		segmentedLine := row.Lines[lineIndex]
		if segmentedLine.Flags&models.Elided == models.Elided {
			break
		}
		lw := strings.Builder{}
		if isHighlighted && segmentedLine.Flags&models.Revision != models.Revision {
			if requiresDescriptionRendering {
				ir.writeSection(w, segmentedLine.Gutter, row.Extend(), true, ir.description, width)
				descriptionRendered = true
				// skip all remaining highlightable lines
				for lineIndex < len(row.Lines) {
					if row.Lines[lineIndex].Flags&models.Highlightable == models.Highlightable {
						lineIndex++
						continue
					} else {
						break
					}
				}
				continue
			}
		}

		// render: gutter
		for i, segment := range segmentedLine.Gutter.Segments {
			gutterInLane := ir.isGutterInLane(lineIndex, i)
			text := ir.updateGutterText(lineIndex, i, segment.Text)
			style := segment.Style
			if gutterInLane {
				style = style.Inherit(ir.textStyle)
			} else {
				style = style.Inherit(ir.dimmedStyle).Faint(true)
			}
			fmt.Fprint(&lw, style.Render(text))
		}

		// render: before change id
		if segmentedLine.Flags&models.Revision == models.Revision {
			if ir.beforeChangeId != "" {
				fmt.Fprint(&lw, ir.beforeChangeId)
			}
		}

		// render: after change id
		for _, segment := range segmentedLine.Segments {
			if segment.Text == row.Commit.CommitId && ir.beforeCommitId != "" {
				fmt.Fprint(&lw, ir.beforeCommitId)
			}

			style := segment.Style
			if isHighlighted {
				style = style.Inherit(ir.selectedStyle)
			} else if ir.inLane {
				style = style.Inherit(ir.textStyle)
			} else {
				style = style.Inherit(ir.dimmedStyle).Faint(true)
			}

			op := ir.op
			if sr, ok := op.(operations.SegmentRenderer); ok {
				rendered := sr.RenderSegment(style, segment, row)
				if rendered != "" {
					fmt.Fprint(&lw, style.Render(rendered))
					continue
				}
			}

			// if the SegmentRenderer did not render anything, fall back to default rendering
			fmt.Fprint(&lw, style.Render(segment.Text))
		}

		// render: affected by last operation
		if segmentedLine.Flags&models.Revision == models.Revision && row.IsAffected {
			style := ir.dimmedStyle
			if isHighlighted {
				style = ir.dimmedStyle.Background(ir.selectedStyle.GetBackground())
			}
			fmt.Fprint(&lw, style.Render(" (affected by last operation)"))
		}
		line := lw.String()
		if isHighlighted && segmentedLine.Flags&models.Highlightable == models.Highlightable {
			fmt.Fprint(w, lipgloss.PlaceHorizontal(width, 0, line, lipgloss.WithWhitespaceBackground(ir.selectedStyle.GetBackground())))
		} else {
			fmt.Fprint(w, lipgloss.PlaceHorizontal(width, 0, line, lipgloss.WithWhitespaceBackground(ir.textStyle.GetBackground())))
		}
		fmt.Fprint(w, "\n")
	}

	// render: description overlay if not yet rendered
	if requiresDescriptionRendering && !descriptionRendered {
		ir.writeSection(w, row.Extend(), row.Extend(), true, ir.description, width)
	}

	if row.Commit.IsRoot() {
		return
	}

	// render: after
	if ir.after != "" {
		extended := row.Extend()
		ir.writeSection(w, extended, extended, false, ir.after, width)
	}

	// render: remaining lines (non-highlightable)
	for lineIndex, segmentedLine := range row.RowLinesIter(models.Excluding(models.Highlightable)) {
		var lw strings.Builder
		for i, segment := range segmentedLine.Gutter.Segments {
			gutterInLane := ir.isGutterInLane(lineIndex, i)
			text := ir.updateGutterText(lineIndex, i, segment.Text)
			style := segment.Style
			if gutterInLane {
				style = style.Inherit(ir.textStyle)
			} else {
				style = style.Inherit(ir.dimmedStyle).Faint(true)
			}
			fmt.Fprint(&lw, style.Render(text))
		}
		for _, segment := range segmentedLine.Segments {
			fmt.Fprint(&lw, segment.Style.Inherit(ir.textStyle).Render(segment.Text))
		}
		line := lw.String()
		fmt.Fprint(w, lipgloss.PlaceHorizontal(width, 0, line, lipgloss.WithWhitespaceBackground(ir.textStyle.GetBackground())))
		fmt.Fprint(w, "\n")
	}
}

func (ir itemRenderer) Height() int {
	return len(ir.row.Lines)
}
