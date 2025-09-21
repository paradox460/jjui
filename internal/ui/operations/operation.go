package operations

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/screen"
)

type RenderPosition int

const (
	RenderPositionNil RenderPosition = iota
	RenderPositionAfter
	RenderPositionBefore
	RenderBeforeChangeId
	RenderBeforeCommitId
	RenderOverDescription
)

type Operation interface {
	Render(commit *jj.Commit, renderPosition RenderPosition) string
	Name() string
}

type OperationWithOverlay interface {
	Operation
	Update(msg tea.Msg) (OperationWithOverlay, tea.Cmd)
}

type TracksSelectedRevision interface {
	SetSelectedRevision(commit *jj.Commit)
}

type HandleKey interface {
	HandleKey(msg tea.KeyMsg) tea.Cmd
}

type SegmentRenderer interface {
	RenderSegment(currentStyle lipgloss.Style, segment *screen.Segment, row parser.Row) string
}
