package graph

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/screen"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/stretchr/testify/assert"
)

var style = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

const width = 40

func TestDefaultRowIterator_Render(t *testing.T) {
	rows := []parser.Row{
		{
			Commit: &jj.Commit{
				ChangeId: "abc",
				CommitId: "123",
			},
			Lines: []*parser.GraphRowLine{
				{
					Gutter: parser.GraphGutter{
						Segments: []*screen.Segment{
							{Text: "@", Style: style},
							{Text: "  ", Style: style},
						},
					},
					Segments: []*screen.Segment{
						{Text: "abc", Style: style},
						{Text: " ", Style: style},
						{Text: "123", Style: style},
					},
					Flags: parser.Revision | parser.Highlightable,
				},
				{
					Gutter: parser.GraphGutter{
						Segments: []*screen.Segment{
							{Text: "|", Style: style},
							{Text: "  ", Style: style},
						},
					},
					Segments: []*screen.Segment{
						{Text: "description goes here", Style: style},
					},
					Flags: parser.Highlightable,
				},
				{
					Gutter: parser.GraphGutter{
						Segments: []*screen.Segment{
							{Text: "~", Style: style},
							{Text: "  ", Style: style},
						},
					},
					Segments: []*screen.Segment{
						{Text: "elided revisions", Style: style},
					},
					Flags: parser.Elided,
				},
			},
		},
	}
	iterator := NewDefaultRowIterator(rows, WithWidth(width))
	iterator.Next()
	var w strings.Builder
	iterator.Render(&w)
	expected := `@  abc 123
|  description goes here
~  elided revisions`
	expected = lipgloss.Place(width, 3, 0, 0, expected)
	assert.Equal(t, strings.Trim(expected, "\n"), strings.Trim(w.String(), "\n"))
}

func TestDefaultRowIterator_Render_WithDescriptionOverride(t *testing.T) {
	rows := []parser.Row{
		{
			Commit: &jj.Commit{
				ChangeId: "abc",
				CommitId: "123",
			},
			Lines: []*parser.GraphRowLine{
				{
					Gutter: parser.GraphGutter{
						Segments: []*screen.Segment{
							{Text: "@", Style: style},
							{Text: "  ", Style: style},
						},
					},
					Segments: []*screen.Segment{
						{Text: "abc", Style: style},
						{Text: " ", Style: style},
						{Text: "123", Style: style},
					},
					Flags: parser.Revision | parser.Highlightable,
				},
				{
					Gutter: parser.GraphGutter{
						Segments: []*screen.Segment{
							{Text: "|", Style: style},
							{Text: "  ", Style: style},
						},
					},
					Segments: []*screen.Segment{
						{Text: "description goes here", Style: style},
					},
					Flags: parser.Highlightable,
				},
				{
					Gutter: parser.GraphGutter{
						Segments: []*screen.Segment{
							{Text: "|", Style: style},
							{Text: "  ", Style: style},
						},
					},
					Segments: []*screen.Segment{
						{Text: "some extra description line", Style: style},
					},
					Flags: parser.Highlightable,
				},
				{
					Gutter: parser.GraphGutter{
						Segments: []*screen.Segment{
							{Text: "~", Style: style},
							{Text: "  ", Style: style},
						},
					},
					Segments: []*screen.Segment{
						{Text: "elided revisions", Style: style},
					},
					Flags: parser.Elided,
				},
			},
		},
	}
	iterator := NewDefaultRowIterator(rows, WithWidth(width))
	iterator.Op = testOp{renderLocation: operations.RenderOverDescription}
	iterator.Next()
	var w strings.Builder
	iterator.Render(&w)
	expected := `@  abc 123
|  test decoration
~  elided revisions`
	expected = lipgloss.Place(width, 3, 0, 0, expected)
	assert.Equal(t, strings.Trim(expected, "\n"), strings.Trim(w.String(), "\n"))
}

func TestDefaultRowIterator_Render_SingleRow_WithDescriptionOverride(t *testing.T) {
	rows := []parser.Row{
		{
			Commit: &jj.Commit{
				ChangeId: "abc",
				CommitId: "123",
			},
			Lines: []*parser.GraphRowLine{
				{
					Gutter: parser.GraphGutter{
						Segments: []*screen.Segment{
							{Text: "@", Style: style},
							{Text: "  ", Style: style},
						},
					},
					Segments: []*screen.Segment{
						{Text: "abc", Style: style},
						{Text: " description goes here ", Style: style},
						{Text: "123", Style: style},
					},
					Flags: parser.Revision | parser.Highlightable,
				},
			},
		},
	}
	iterator := NewDefaultRowIterator(rows, WithWidth(width))
	iterator.Op = testOp{renderLocation: operations.RenderOverDescription}
	iterator.Next()
	var w strings.Builder
	iterator.Render(&w)
	expected := `@  abc description goes here 123
  test decoration`
	expected = lipgloss.Place(width, 2, 0, 0, expected)
	assert.Equal(t, strings.Trim(expected, "\n"), strings.Trim(w.String(), "\n"))
}

func TestDefaultRowIterator_Render_WithSelection(t *testing.T) {
	rows := []parser.Row{
		{
			Commit: &jj.Commit{
				ChangeId: "abc",
				CommitId: "123",
			},
			Lines: []*parser.GraphRowLine{
				{
					Gutter: parser.GraphGutter{
						Segments: []*screen.Segment{
							{Text: "@", Style: style},
							{Text: "  ", Style: style},
						},
					},
					Segments: []*screen.Segment{
						{Text: "abc", Style: style},
						{Text: " ", Style: style},
						{Text: "123", Style: style},
					},
					Flags: parser.Revision | parser.Highlightable,
				},
			},
		},
	}
	selections := map[string]bool{
		"abc": true,
	}
	iterator := NewDefaultRowIterator(rows, WithWidth(width), WithStylePrefix(""), WithSelections(selections))
	iterator.Next()
	var w strings.Builder
	iterator.Render(&w)
	expected := `@  ✓ abc 123`
	assert.Contains(t, w.String(), expected)
}

func TestDefaultRowIterator_Render_Affected(t *testing.T) {
	rows := []parser.Row{
		{
			Commit: &jj.Commit{
				ChangeId: "abc",
				CommitId: "123",
			},
			IsAffected: true,
			Lines: []*parser.GraphRowLine{
				{
					Gutter: parser.GraphGutter{
						Segments: []*screen.Segment{
							{Text: "@", Style: style},
							{Text: "  ", Style: style},
						},
					},
					Segments: []*screen.Segment{
						{Text: "abc", Style: style},
						{Text: " ", Style: style},
						{Text: "123", Style: style},
					},
					Flags: parser.Revision | parser.Highlightable,
				},
			},
		},
	}
	iterator := NewDefaultRowIterator(rows, WithWidth(width), WithStylePrefix(""))
	iterator.Next()
	var w strings.Builder
	iterator.Render(&w)
	expected := `@  abc 123 (affected by last operation)`
	assert.Contains(t, w.String(), expected)
}

type testOp struct {
	renderLocation operations.RenderPosition
}

func (t testOp) Render(commit *jj.Commit, renderPosition operations.RenderPosition) string {
	if t.renderLocation == renderPosition {
		return "test decoration"
	}
	return ""
}

func (t testOp) Name() string {
	return "test"
}
