package bookmark

import (
	"testing"
	"time"

	"github.com/idursun/jjui/internal/jj"

	"github.com/idursun/jjui/test"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

func TestSetBookmarkModel_Update(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListMovable("revision"))
	commandRunner.Expect(jj.BookmarkSet("revision", "name"))
	defer commandRunner.Verify()

	op, _ := NewSetBookmarkOperation(test.NewTestContext(commandRunner), "revision")
	host := test.OperationHost{Operation: op}
	tm := teatest.NewTestModel(t, host)
	tm.Type("name")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}
