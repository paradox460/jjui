package details

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

type status uint8

var (
	Added    status = 0
	Deleted  status = 1
	Modified status = 2
	Renamed  status = 3
)

type item struct {
	status   status
	name     string
	fileName string
	selected bool
	conflict bool
}

func (f item) Title() string {
	status := "M"
	switch f.status {
	case Added:
		status = "A"
	case Deleted:
		status = "D"
	case Modified:
		status = "M"
	case Renamed:
		status = "R"
	}

	return fmt.Sprintf("%s %s", status, f.name)
}
func (f item) Description() string { return "" }
func (f item) FilterValue() string { return f.name }

type styles struct {
	Added    lipgloss.Style
	Deleted  lipgloss.Style
	Modified lipgloss.Style
	Renamed  lipgloss.Style
	Selected lipgloss.Style
	Dimmed   lipgloss.Style
	Text     lipgloss.Style
	Conflict lipgloss.Style
}
