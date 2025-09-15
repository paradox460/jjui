package revisions

import (
	"testing"

	"github.com/idursun/jjui/internal/models"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/view"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
)

func TestModel_highlightChanges(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	revisionsContext := context.NewRevisionsContext(commandRunner)
	revisionsContext.SetItems([]*models.RevisionItem{
		{IsAffected: false, Row: models.Row{Commit: &models.Commit{ChangeId: "someother"}}},
		{IsAffected: false, Row: models.Row{Commit: &models.Commit{ChangeId: "nyqzpsmt"}}},
	})
	viewManager := view.NewViewManager()
	model := New(revisionsContext, viewManager).(*Model)
	model.output = `
Absorbed changes into these revisions:
  nyqzpsmt 8b1e95e3 change third file
Working copy now at: okrwsxvv 5233c94f (empty) (no description set)
Parent commit      : nyqzpsmt 8b1e95e3 change third file
`
	_ = model.highlightChanges()
	assert.False(t, revisionsContext.List.Items[0].IsAffected)
	assert.True(t, revisionsContext.List.Items[1].IsAffected)
}
