package abandon

import (
	"bytes"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/test"
)

var commit = &jj.Commit{ChangeId: "a"}
var revisions = jj.NewSelectedRevisions(commit)

func Test_Accept(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Abandon(revisions, false))
	defer commandRunner.Verify()

	model := test.NewOperationHost(NewOperation(test.NewTestContext(commandRunner), revisions), commit)
	tm := teatest.NewTestModel(t, model)
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("abandon"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("closed"))
	})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func Test_Cancel(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	model := test.NewOperationHost(NewOperation(test.NewTestContext(commandRunner), revisions), commit)
	tm := teatest.NewTestModel(t, model)
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("closed"))
	})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}
