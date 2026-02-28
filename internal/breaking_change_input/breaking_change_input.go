package breaking_change_input

import (
	"io"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/muesli/termenv"
	"github.com/skalt/git-cc/internal/config"
	"github.com/skalt/git-cc/internal/helpbar"
	"github.com/skalt/git-cc/internal/utils"
)

type Model struct {
	input   textinput.Model
	helpBar helpbar.Model
}

var helpBar = termenv.String(strings.Join(
	[]string{config.HelpSubmit, config.HelpBack, config.HelpCancel}, "; "),
).Faint().String()

func (m Model) Value() string {
	return m.input.Value()
}

func (m Model) Render(b io.StringWriter) {
	_ = utils.Must(b.WriteString(m.input.View()))
	_ = utils.Must(b.WriteString("\n\n"))
	_ = utils.Must(b.WriteString(helpBar))
	_ = utils.Must(b.WriteString("\n"))
}

func (m Model) Update(msg tea.Msg) (out Model, cmd tea.Cmd) {
	m.input, cmd = m.input.Update(msg)
	out = m
	return
}

func NewModel() Model {
	input := textinput.New()
	input.Prompt = termenv.String("Breaking changes: ").Faint().String()
	input.Placeholder = "if any."
	input.Focus()
	return Model{
		input,
		helpbar.NewModel(config.HelpSubmit, config.HelpBack, config.HelpCancel),
	}
}
