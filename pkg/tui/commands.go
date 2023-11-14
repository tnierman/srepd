package tui

import (
	"context"
	"errors"
	"log"
	"os"
	"os/exec"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/clcollins/srepd/pkg/pd"
)

type updateIncidentListMsg string
type updatedIncidentListMsg struct {
	incidents []pagerduty.Incident
	err       error
}

func updateIncidentList(p *pd.Config) tea.Cmd {
	return func() tea.Msg {
		var teams []string
		for _, t := range p.Teams {
			teams = append(teams, t.ID)
		}
		opts := pd.NewListIncidentOptsFromDefaults()
		opts.TeamIDs = teams

		i, err := pd.GetIncidents(p.Client, opts)
		return updatedIncidentListMsg{i, err}
	}
}

type getIncidentMsg string
type gotIncidentMsg struct {
	incident *pagerduty.Incident
	err      error
}

func getIncident(p *pd.Config, id string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		i, err := p.Client.GetIncidentWithContext(ctx, id)
		return gotIncidentMsg{i, err}
	}
}

type gotIncidentAlertsMsg struct {
	alerts []pagerduty.IncidentAlert
	err    error
}

func getIncidentAlerts(p *pd.Config, id string) tea.Cmd {
	return func() tea.Msg {
		a, err := pd.GetAlerts(p.Client, id, pagerduty.ListIncidentAlertsOptions{})
		return gotIncidentAlertsMsg{a, err}
	}
}

type gotIncidentNotesMsg struct {
	notes []pagerduty.IncidentNote
	err   error
}

func getIncidentNotes(p *pd.Config, id string) tea.Cmd {
	return func() tea.Msg {
		n, err := pd.GetNotes(p.Client, id)
		return gotIncidentNotesMsg{n, err}
	}
}

type getCurrentUserMsg string
type gotCurrentUserMsg struct {
	user *pagerduty.User
	err  error
}

func getCurrentUser(p *pd.Config) tea.Cmd {
	return func() tea.Msg {
		u, err := p.Client.GetCurrentUserWithContext(
			context.Background(),
			pagerduty.GetCurrentUserOptions{},
		)
		return gotCurrentUserMsg{u, err}
	}
}

func AssignedToAnyUsers(i pagerduty.Incident, ids []string) bool {
	for _, a := range i.Assignments {
		for _, id := range ids {
			if a.Assignee.ID == id {
				return true
			}
		}
	}
	return false
}

func AssignedToUser(i pagerduty.Incident, id string) bool {
	for _, a := range i.Assignments {
		if a.Assignee.ID == id {
			return true
		}
	}
	return false
}

type editorFinishedMsg struct {
	err  error
	file *os.File
}

// TODO: is this needed
// var defaultEditor = "/usr/bin/vim"

func openEditorCmd(editor string) tea.Cmd {
	file, err := os.CreateTemp(os.TempDir(), "")
	if err != nil {
		return func() tea.Msg {
			return errMsg{error: err}
		}
	}
	c := exec.Command(editor, file.Name())
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return editorFinishedMsg{err, file}
	})
}

func newIncidentViewer(content string) viewport.Model {

	vp := viewport.New(windowSize.Width, windowSize.Height-5)
	vp.MouseWheelEnabled = true
	vp.Style = lipgloss.NewStyle().
		Width(windowSize.Width - 10).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		PaddingRight(2)
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(windowSize.Width),
	)
	if err != nil {
		log.Fatal(err)
	}

	str, err := renderer.Render(content)
	if err != nil {
		log.Fatal(err)
	}

	vp.SetContent(str)
	return vp
}

type acknowledgeIncidentsMsg struct {
	incidents []pagerduty.Incident
}
type acknowledgedIncidentsMsg struct {
	incidents []pagerduty.Incident
	err       error
}
type waitForSelectedIncidentsThenAcknowledgeMsg string

func acknowledgeIncidents(p *pd.Config, incidents []*pagerduty.Incident) tea.Cmd {
	return func() tea.Msg {
		u, err := p.Client.GetCurrentUserWithContext(context.Background(), pagerduty.GetCurrentUserOptions{})
		if err != nil {
			return errMsg{err}
		}
		a, err := pd.AcknowledgeIncident(p.Client, incidents, u)
		if err != nil {
			return errMsg{err}
		}
		return acknowledgedIncidentsMsg{a, err}
	}
}

type reassignIncidentsMsg struct {
	incidents []*pagerduty.Incident
	users     []*pagerduty.User
}
type reassignedIncidentsMsg []pagerduty.Incident

func reassignIncidents(p *pd.Config, i []*pagerduty.Incident, user *pagerduty.User, users []*pagerduty.User) tea.Cmd {
	return func() tea.Msg {
		r, err := pd.ReassignIncidents(p.Client, i, user, users)
		if err != nil {
			return errMsg{err}
		}
		return reassignedIncidentsMsg(r)
	}
}

type silenceIncidentsMsg []*pagerduty.Incident
type waitForSelectedIncidentsThenSilenceMsg string

var errSilenceIncidentInvalidArgs = errors.New("silenceIncidents: invalid arguments")

func silenceIncidents(i []*pagerduty.Incident, u []*pagerduty.User) tea.Cmd {
	return func() tea.Msg {
		if len(i) == 0 || len(u) == 0 {
			return errMsg{errSilenceIncidentInvalidArgs}
		}
		return reassignIncidentsMsg{i, u}
	}
}

type clearSelectedIncidentsMsg string

type addIncidentNoteMsg string
type addedIncidentNoteMsg struct {
	note *pagerduty.IncidentNote
	err  error
}

func addNoteToIncident(p *pd.Config, incident *pagerduty.Incident, user *pagerduty.User, content *os.File) tea.Cmd {
	return func() tea.Msg {
		defer content.Close()

		bytes, err := os.ReadFile(content.Name())
		if err != nil {
			return errMsg{err}
		}
		content := string(bytes[:])

		n, err := pd.PostNote(p.Client, incident.ID, user, content)
		return addedIncidentNoteMsg{n, err}
	}
}
