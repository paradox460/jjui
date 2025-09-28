package copy

import (
	"os/exec"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	appContext "github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/operations"
)

type CopyTarget int

const (
	CopyChangeId CopyTarget = iota
	CopyCommitId
	CopyDescription
	CopyFullInfo
)

type styles struct {
	sourceMarker lipgloss.Style
}

var _ operations.Operation = (*Operation)(nil)
var _ common.Focusable = (*Operation)(nil)

type Operation struct {
	context *appContext.MainContext
	From    *jj.Commit
	target  CopyTarget
	keyMap  config.KeyMappings[key.Binding]
	styles  styles
}

func (c *Operation) IsFocused() bool {
	return true
}

func (c *Operation) Init() tea.Cmd {
	return nil
}

func (c *Operation) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		return c, c.HandleKey(msg)
	}
	return c, nil
}

func (c *Operation) View() string {
	return ""
}

func (c *Operation) HandleKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, c.keyMap.Copy.ChangeId):
		c.target = CopyChangeId
		return c.copyToClipboard()
	case key.Matches(msg, c.keyMap.Copy.CommitId):
		c.target = CopyCommitId
		return c.copyToClipboard()
	case key.Matches(msg, c.keyMap.Copy.Description):
		c.target = CopyDescription
		return c.copyToClipboard()
	case key.Matches(msg, c.keyMap.Copy.FullInfo):
		c.target = CopyFullInfo
		return c.copyToClipboard()
	case key.Matches(msg, c.keyMap.Cancel):
		return common.Close
	}
	return nil
}

func (c *Operation) copyToClipboard() tea.Cmd {
	if c.From == nil {
		return common.Close
	}

	var content string
	switch c.target {
	case CopyChangeId:
		// Get full change ID using jj command
		changeIdOutput, _ := c.context.RunCommandImmediate([]string{"show", c.From.GetChangeId(), "--color", "never", "--no-patch", "--quiet", "--template", "change_id"})
		content = strings.TrimSpace(string(changeIdOutput))
	case CopyCommitId:
		// Get full commit ID using jj command
		commitIdOutput, _ := c.context.RunCommandImmediate([]string{"show", c.From.GetChangeId(), "--color", "never", "--no-patch", "--quiet", "--template", "commit_id"})
		content = strings.TrimSpace(string(commitIdOutput))
	case CopyDescription:
		// Get description from the commit
		descOutput, _ := c.context.RunCommandImmediate(jj.GetDescription(c.From.GetChangeId()))
		content = string(descOutput)
	case CopyFullInfo:
		// Get full IDs, description, author, and committer
		output, _ := c.context.RunCommandImmediate([]string{"show", c.From.GetChangeId(), "--color", "never", "--quiet", "--no-patch", "--template", "'Change ID: ' ++ change_id ++ '\nCommit ID: ' ++ commit_id ++ '\nAuthor: ' ++ author ++ '\nCommitter: ' ++ committer ++ '\nDescription: ' ++ description"})
		content = strings.TrimSpace(string(output))
	}

	// Copy to clipboard using system command
	return tea.Batch(
		func() tea.Msg {
			// Detect clipboard command based on OS
			var cmd []string
			switch runtime.GOOS {
			case "darwin":
				cmd = []string{"pbcopy"}
			case "linux":
				// On Linux, prefer wl-copy for Wayland, fallback to xclip
				if _, err := exec.LookPath("wl-copy"); err == nil {
					cmd = []string{"wl-copy"}
				} else {
					cmd = []string{"xclip", "-sel", "clipboard"}
				}
			case "windows":
				cmd = []string{"clip"}
			default:
				// Unsupported OS
				return nil
			}

			if len(cmd) > 0 {
				// Create the command and pipe content to it
				command := exec.Command(cmd[0], cmd[1:]...)
				command.Stdin = strings.NewReader(content)
				if command.Run() == nil {
					// Successfully copied
					return nil
				}
			}

			// Failed to copy
			return nil
		},
		common.Close,
	)
}

func (c *Operation) SetSelectedRevision(commit *jj.Commit) {
	c.From = commit
}

func (c *Operation) ShortHelp() []key.Binding {
	return []key.Binding{
		c.keyMap.Copy.ChangeId,
		c.keyMap.Copy.CommitId,
		c.keyMap.Copy.Description,
		c.keyMap.Copy.FullInfo,
		c.keyMap.Cancel,
	}
}

func (c *Operation) FullHelp() [][]key.Binding {
	return [][]key.Binding{c.ShortHelp()}
}

func (c *Operation) Render(commit *jj.Commit, pos operations.RenderPosition) string {
	if pos == operations.RenderBeforeChangeId && commit == c.From {
		return c.styles.sourceMarker.Render("<< copy >>")
	}
	return ""
}

func (c *Operation) Name() string {
	return "copy"
}

func NewOperation(context *appContext.MainContext, from *jj.Commit) *Operation {
	palette := common.DefaultPalette

	return &Operation{
		context: context,
		From:    from,
		target:  CopyChangeId, // Default to copying change ID
		keyMap:  config.Current.GetKeyMap(),
		styles: styles{
			sourceMarker: palette.Get("copy source_marker"),
		},
	}
}
