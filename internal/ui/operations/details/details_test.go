package details

import (
	"bytes"
	"testing"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/models"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/view"

	"github.com/idursun/jjui/test"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

const (
	StatusOutput = "false false\nM file.txt\nA newfile.txt\n"
)

var revision = models.RevisionItem{
	Checkable: nil,
	Row: models.Row{
		Commit: &models.Commit{ChangeId: "abc", CommitId: "123"},
	},
	IsAffected: false,
}

var file = models.RevisionFileItem{
	Checkable: &models.Checkable{Checked: false},
	Status:    0,
	Name:      "file.txt",
	FileName:  "file.txt",
	Conflict:  false,
}

func TestModel_Init_ExecutesStatusCommand(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.SnapshotArgs{}.GetArgs())
	commandRunner.Expect(jj.StatusArgs{Revision: revision}.GetArgs()).SetOutput([]byte(StatusOutput))
	defer commandRunner.Verify()

	revisionsContext := context.NewRevisionsContext(commandRunner)
	revisionsContext.SetItems(
		[]*models.RevisionItem{&revision},
	)
	revisionsContext.SetCursor(0)
	model := NewOperation(revisionsContext, context.NewDetailsContext(commandRunner, revisionsContext))
	viewManager := view.NewViewManager()
	_ = viewManager.CreateView(model)
	tm := teatest.NewTestModel(t, model)
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("file.txt"))
	})
}

func TestModel_Update_RestoresSelectedFiles(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.SnapshotArgs{}.GetArgs())
	commandRunner.Expect(jj.StatusArgs{Revision: revision}.GetArgs()).SetOutput([]byte(StatusOutput))
	commandRunner.Expect(jj.RestoreArgs{Revision: revision, Files: []models.RevisionFileItem{file}}.GetArgs())
	defer commandRunner.Verify()

	revisionsContext := context.NewRevisionsContext(commandRunner)
	revisionsContext.SetItems([]*models.RevisionItem{
		&revision,
	})
	revisionsContext.Cursor = 0
	detailsContext := context.NewDetailsContext(commandRunner, revisionsContext)
	detailsContext.SetItems([]*models.RevisionFileItem{
		{
			Checkable: &models.Checkable{Checked: false},
			Status:    0,
			Name:      "file.txt",
			FileName:  "file.txt",
			Conflict:  false,
		},
	})
	detailsContext.Cursor = 0
	model := NewOperation(revisionsContext, detailsContext)
	viewManager := view.NewViewManager()
	_ = viewManager.CreateView(model)
	viewManager.FocusView(model.Id)
	tm := teatest.NewTestModel(t, model)
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("file.txt"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeySpace})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return model.confirmation == nil
	})
	tm.Quit()
}

func TestModel_Update_SplitsSelectedFiles(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.SnapshotArgs{}.GetArgs())
	commandRunner.Expect(jj.StatusArgs{Revision: revision}.GetArgs()).SetOutput([]byte(StatusOutput))
	commandRunner.Expect(jj.SplitArgs{Revision: revision, Files: []models.RevisionFileItem{file}}.GetArgs())
	defer commandRunner.Verify()

	revisionsContext := context.NewRevisionsContext(commandRunner)
	revisionsContext.SetItems([]*models.RevisionItem{
		&revision,
	})
	revisionsContext.Cursor = 0

	model := NewOperation(revisionsContext, revisionsContext.CreateDetailsContext())
	viewManager := view.NewViewManager()
	_ = viewManager.CreateView(model)
	viewManager.FocusView(model.Id)
	tm := teatest.NewTestModel(t, model)
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("file.txt"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeySpace})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return !viewManager.IsFocused(model.Id)
	})
	tm.Quit()
}

func TestModel_Update_HandlesMovedFiles(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.SnapshotArgs{}.GetArgs())
	commandRunner.Expect(jj.StatusArgs{Revision: revision}.GetArgs()).SetOutput([]byte("false false\nR internal/ui/{revisions => }/file.go\nR {file => sub/newfile}\n"))
	files := []models.RevisionFileItem{
		{
			Checkable: &models.Checkable{Checked: false},
			Status:    2,
			Name:      "internal/ui/{revisions => }/file.go",
			FileName:  "internal/ui/file.go",
			Conflict:  false,
		},
		{
			Checkable: &models.Checkable{Checked: false},
			Status:    2,
			Name:      "R {file => sub/newfile}",
			FileName:  "sub/newfile",
			Conflict:  false,
		},
	}
	commandRunner.Expect(jj.RestoreArgs{Revision: revision, Files: files}.GetArgs())
	defer commandRunner.Verify()

	revisionsContext := context.NewRevisionsContext(commandRunner)
	revisionsContext.SetItems([]*models.RevisionItem{
		&revision,
	})
	revisionsContext.Cursor = 0

	detailsContext := context.NewDetailsContext(commandRunner, revisionsContext)

	model := NewOperation(revisionsContext, detailsContext)
	viewManager := view.NewViewManager()
	_ = viewManager.CreateView(model)
	viewManager.FocusView(model.Id)
	tm := teatest.NewTestModel(t, model)
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("file.go"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeySpace})
	tm.Send(tea.KeyMsg{Type: tea.KeySpace})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return model.confirmation == nil
	})
	tm.Quit()
}
