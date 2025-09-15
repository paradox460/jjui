package context

import (
	"bytes"
	"io"

	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/models"
	"github.com/idursun/jjui/internal/screen"
	"github.com/idursun/jjui/internal/ui/common/list"
)

type OplogContext struct {
	CommandRunner
	*list.List[*models.OperationLogItem]
}

func NewOplogContext(ctx *MainContext) *OplogContext {
	return &OplogContext{
		CommandRunner: ctx.CommandRunner,
		List:          list.NewList[*models.OperationLogItem](),
	}
}

func (ctx *OplogContext) Load() {
	output, err := ctx.RunCommandImmediate(jj.Args(jj.OpLogArgs{
		NoGraph:         false,
		Limit:           config.Current.OpLog.Limit,
		GlobalArguments: jj.GlobalArguments{IgnoreWorkingCopy: true, Color: "always"},
	}))

	if err == nil {
		rows := parseRows(bytes.NewReader(output))
		ctx.List.SetItems(rows)
	}
}

func newRowLine(segments []*screen.Segment) models.OperationLogRowLine {
	return models.OperationLogRowLine{Segments: segments}
}

func parseRows(reader io.Reader) []*models.OperationLogItem {
	var rows []*models.OperationLogItem
	var r models.OperationLogRow
	rawSegments := screen.ParseFromReader(reader)

	for segmentedLine := range screen.BreakNewLinesIter(rawSegments) {
		rowLine := newRowLine(segmentedLine)
		if opIdIdx := rowLine.FindIdIndex(); opIdIdx != -1 {
			if r.OperationId != "" {
				rows = append(rows, &models.OperationLogItem{OperationId: r.OperationId, OperationLogRow: r})
			}
			r = models.OperationLogRow{OperationId: rowLine.Segments[opIdIdx].Text}
		}
		r.Lines = append(r.Lines, &rowLine)
	}
	rows = append(rows, &models.OperationLogItem{OperationId: r.OperationId, OperationLogRow: r})
	return rows
}
