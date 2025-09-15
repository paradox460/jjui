package context

import (
	"strings"

	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/models"
	"github.com/idursun/jjui/internal/ui/common/list"
)

type MainContext struct {
	CommandRunner
	Preview        *PreviewContext
	Location       string
	CustomCommands map[string]CustomCommand
	Leader         LeaderMap
	JJConfig       *config.JJConfig
	DefaultRevset  string
	CurrentRevset  string
	History        HistoryContext
}

type HistoryContext struct {
	*config.Histories
}

func NewAppContext(commandRunner CommandRunner, location string) *MainContext {
	m := &MainContext{
		CommandRunner: commandRunner,
		Location:      location,
		History:       HistoryContext{Histories: config.NewHistories()},
	}
	m.Preview = NewPreviewContext(commandRunner)

	m.JJConfig = &config.JJConfig{}
	if output, err := m.RunCommandImmediate(jj.Args(jj.ConfigListAllArgs{})); err == nil {
		m.JJConfig, _ = config.DefaultConfig(output)
	}
	return m
}

func (ctx *MainContext) CreateRevisionsContext() *RevisionsContext {
	revisionContext := NewRevisionsContext(ctx)
	revisionContext.Location = ctx.Location
	revisionContext.CurrentRevset = ctx.CurrentRevset
	return revisionContext
}

func (ctx *MainContext) CreateEvologContext() *list.List[*models.RevisionItem] {
	return list.NewList[*models.RevisionItem]()
}

func (ctx *MainContext) CreateOplogContext() *OplogContext {
	return NewOplogContext(ctx)
}

// CreateReplacements context aware replacements for custom commands and exec input.
func (ctx *MainContext) CreateReplacements() map[string]string {
	replacements := make(map[string]string)
	replacements[jj.RevsetPlaceholder] = ctx.CurrentRevset

	//if current := ctx.revisions.Current(); current != nil {
	//	replacements[jj.ChangeIdPlaceholder] = current.Commit.ChangeId
	//	replacements[jj.CommitIdPlaceholder] = current.Commit.CommitId
	//}
	//if current := ctx.files.Current(); current != nil {
	//	replacements[jj.FilePlaceholder] = current.FileName
	//}
	//if current := ctx.OpLog.Current(); current != nil {
	//	replacements[jj.OperationIdPlaceholder] = current.OperationId
	//}
	//if current := ctx.Evolog.Current(); current != nil {
	//	replacements[jj.CommitIdPlaceholder] = current.Commit.CommitId
	//}

	var checkedRevisions []string
	//for _, item := range ctx.revisions.GetCheckedItems() {
	//	checkedRevisions = append(checkedRevisions, item.Commit.CommitId)
	//}

	if len(checkedRevisions) == 0 {
		replacements[jj.CheckedCommitIdsPlaceholder] = "none()"
	} else {
		replacements[jj.CheckedCommitIdsPlaceholder] = strings.Join(checkedRevisions, "|")
	}

	var checkedFiles []string
	//for _, item := range ctx.files.GetCheckedItems() {
	//	checkedFiles = append(checkedFiles, item.FileName)
	//}

	if len(checkedFiles) > 0 {
		replacements[jj.CheckedFilesPlaceholder] = strings.Join(checkedFiles, "\t")
	}

	return replacements
}
