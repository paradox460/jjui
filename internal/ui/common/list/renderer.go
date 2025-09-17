package list

import (
	"bytes"
	"io"
	"strings"

	"github.com/idursun/jjui/internal/ui/view"
)

type IList interface {
	Len() int
	GetRenderer(index int) IItemRenderer
}

type IItemRenderer interface {
	Render(w io.Writer, width int)
	Height() int
}

type ListRenderer struct {
	*view.ViewRange
	list             IList
	buffer           bytes.Buffer
	skippedLineCount int
	lineCount        int
}

func NewRenderer(list IList, size *view.Sizeable) *ListRenderer {
	return &ListRenderer{
		ViewRange: &view.ViewRange{Sizeable: size, Start: 0, End: size.Height, FirstRowIndex: -1, LastRowIndex: -1},
		list:      list,
		buffer:    bytes.Buffer{},
	}
}

func (r *ListRenderer) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	r.lineCount += bytes.Count(p, []byte("\n"))
	return r.buffer.Write(p)
}

func (r *ListRenderer) String(start, end int) string {
	start = start - r.skippedLineCount
	end = end - r.skippedLineCount
	lines := strings.Split(r.buffer.String(), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	if start < 0 {
		start = 0
	}
	if end < start {
		end = start
	}
	for end > len(lines) {
		lines = append(lines, "")
	}
	return strings.Join(lines[start:end], "\n")
}

func (r *ListRenderer) Reset() {
	r.buffer.Reset()
	r.lineCount = 0
	r.skippedLineCount = 0
}

func (r *ListRenderer) Render(focusIndex int) string {
	r.Reset()
	viewHeight := r.End - r.Start
	if viewHeight != r.Height {
		r.End = r.Start + r.Height
	}

	selectedLineStart := -1
	selectedLineEnd := -1
	firstRenderedRowIndex := -1
	lastRenderedRowIndex := -1
	for i := range r.list.Len() {
		isFocused := i == focusIndex
		itemRenderer := r.list.GetRenderer(i)
		if isFocused {
			selectedLineStart = r.totalLineCount()
			if selectedLineStart < r.Start {
				r.Start = selectedLineStart
			}
		} else {
			rowLineCount := itemRenderer.Height()
			if rowLineCount+r.totalLineCount() < r.Start {
				r.skipLines(rowLineCount)
				continue
			}
		}
		itemRenderer.Render(r, r.ViewRange.Width)
		if firstRenderedRowIndex == -1 {
			firstRenderedRowIndex = i
		}

		if isFocused {
			selectedLineEnd = r.totalLineCount()
		}
		if selectedLineEnd > 0 && r.totalLineCount() > r.End {
			lastRenderedRowIndex = i
			break
		}
	}

	if lastRenderedRowIndex == -1 {
		lastRenderedRowIndex = r.list.Len() - 1
	}

	r.FirstRowIndex = firstRenderedRowIndex
	r.LastRowIndex = lastRenderedRowIndex
	if selectedLineStart <= r.Start {
		r.Start = selectedLineStart
		r.End = selectedLineStart + r.Height
	} else if selectedLineEnd > r.End {
		r.End = selectedLineEnd
		r.Start = selectedLineEnd - r.Height
	}

	return r.String(r.Start, r.End)
}

func (r *ListRenderer) skipLines(amount int) {
	r.skippedLineCount = r.skippedLineCount + amount
}

func (r *ListRenderer) totalLineCount() int {
	return r.lineCount + r.skippedLineCount
}
