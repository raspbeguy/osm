package tui

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/paulmach/osm"

	"github.com/raspbeguy/osm/api"
)

type csMode int

const (
	csModeSummary csMode = iota
	csModeElements
	csModeXML
)

type changesetViewLoadedMsg struct {
	csID osm.ChangesetID
	cs   *osm.Changeset
	err  error
}

type changesetXMLLoadedMsg struct {
	csID osm.ChangesetID
	xml  string
	err  error
}

type changesetElement struct {
	Kind    string // "node", "way", "relation"
	ID      int64
	Version int
	Action  rune // '+', '~', '-'
	Tags    osm.Tags
}

type csElementItem struct{ e changesetElement }

func (i csElementItem) Title() string {
	return fmt.Sprintf("%c %s %d v%d", i.e.Action, i.e.Kind, i.e.ID, i.e.Version)
}

func (i csElementItem) Description() string {
	if len(i.e.Tags) == 0 {
		return "(no tags)"
	}
	parts := make([]string, 0, len(i.e.Tags))
	for _, t := range i.e.Tags {
		parts = append(parts, t.Key+"="+t.Value)
	}
	return strings.Join(parts, ", ")
}

func (i csElementItem) FilterValue() string { return i.Title() }

type changesetViewModel struct {
	client       *api.Client
	csID         osm.ChangesetID
	spinner      spinner.Model
	viewport     viewport.Model
	elementsList list.Model
	cs           *osm.Changeset
	err          error
	loading      bool
	xml          string
	xmlLoading   bool
	elements     []changesetElement
	mode         csMode
}

func newChangesetView(c *api.Client) changesetViewModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	l := list.New(nil, list.NewDefaultDelegate(), 60, 15)
	l.Title = "Elements"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	return changesetViewModel{
		client:       c,
		spinner:      s,
		viewport:     viewport.New(60, 15),
		elementsList: l,
	}
}

func (m changesetViewModel) Init() tea.Cmd { return nil }

func (m changesetViewModel) show(id int64) (changesetViewModel, tea.Cmd) {
	m.csID = osm.ChangesetID(id)
	m.loading = true
	m.err = nil
	m.cs = nil
	m.xml = ""
	m.xmlLoading = false
	m.elements = nil
	m.elementsList.SetItems(nil)
	m.mode = csModeSummary
	return m, tea.Batch(m.spinner.Tick, m.load())
}

func (m changesetViewModel) load() tea.Cmd {
	client := m.client
	id := m.csID
	return func() tea.Msg {
		cs, err := client.GetChangeset(context.Background(), id)
		return changesetViewLoadedMsg{csID: id, cs: cs, err: err}
	}
}

func (m changesetViewModel) loadXML() tea.Cmd {
	client := m.client
	id := m.csID
	return func() tea.Msg {
		x, err := client.DownloadChangeset(context.Background(), id)
		return changesetXMLLoadedMsg{csID: id, xml: x, err: err}
	}
}

// selectedElement returns the element under the cursor in elements mode.
// Falls back to an empty value when no element is active.
func (m changesetViewModel) selectedElement() changesetElement {
	if i, ok := m.elementsList.SelectedItem().(csElementItem); ok {
		return i.e
	}
	return changesetElement{}
}

func (m changesetViewModel) Update(msg tea.Msg) (changesetViewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case changesetViewLoadedMsg:
		if msg.csID != m.csID {
			return m, nil
		}
		m.loading = false
		m.cs = msg.cs
		m.err = msg.err
		m = m.rewrap()
		return m, nil
	case changesetXMLLoadedMsg:
		if msg.csID != m.csID {
			return m, nil
		}
		m.xmlLoading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.xml = msg.xml
		if elems, err := extractChangesetElements(m.xml); err == nil {
			m.elements = elems
			items := make([]list.Item, len(elems))
			for i, e := range elems {
				items[i] = csElementItem{e: e}
			}
			m.elementsList.SetItems(items)
		}
		m = m.rewrap()
		return m, nil
	case spinner.TickMsg:
		if !m.loading && !m.xmlLoading {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.KeyMsg:
		if m.cs != nil {
			switch msg.String() {
			case "s":
				m.mode = csModeSummary
				m = m.rewrap()
				return m, nil
			case "e":
				m.mode = csModeElements
				m = m.rewrap()
				if m.xml == "" && !m.xmlLoading {
					m.xmlLoading = true
					return m, tea.Batch(m.spinner.Tick, m.loadXML())
				}
				return m, nil
			case "x":
				m.mode = csModeXML
				m = m.rewrap()
				if m.xml == "" && !m.xmlLoading {
					m.xmlLoading = true
					return m, tea.Batch(m.spinner.Tick, m.loadXML())
				}
				return m, nil
			}
			if m.mode == csModeElements && msg.String() == "enter" {
				if e := m.selectedElement(); e.ID != 0 {
					return m, func() tea.Msg {
						return navigateMsg{to: screenItemView, itemID: e.ID, kind: e.Kind, parent: screenChangesetView}
					}
				}
			}
		}
	}
	var cmd tea.Cmd
	if m.mode == csModeElements {
		m.elementsList, cmd = m.elementsList.Update(msg)
	} else {
		m.viewport, cmd = m.viewport.Update(msg)
	}
	return m, cmd
}

func (m changesetViewModel) rewrap() changesetViewModel {
	switch m.mode {
	case csModeXML:
		if m.xml != "" {
			m.viewport.SetContent(wrapText(m.xml, m.viewport.Width))
		}
	case csModeElements:
		// list manages its own layout
	default:
		if m.cs != nil {
			m.viewport.SetContent(wrapText(formatChangesetBody(m.cs), m.viewport.Width))
		}
	}
	return m
}

func (m changesetViewModel) View() string {
	if m.loading {
		return m.spinner.View() + " loading changeset..."
	}
	if m.err != nil {
		return errorStyle.Render("error: "+m.err.Error()) + "\n" + footerStyle.Render("esc back")
	}
	if m.cs == nil {
		return "no changeset\n" + footerStyle.Render("esc back")
	}
	closedAt := "(open)"
	if !m.cs.ClosedAt.IsZero() {
		closedAt = m.cs.ClosedAt.Format(time.RFC3339)
	}
	title := m.cs.Comment()
	if title == "" {
		title = fmt.Sprintf("changeset %d", m.cs.ID)
	}
	header := headerStyle.Render(title) + "\n" +
		mutedStyle.Render(fmt.Sprintf("id %d • %s • open %v • created %s • closed %s",
			m.cs.ID, m.cs.User, m.cs.Open, m.cs.CreatedAt.Format(time.RFC3339), closedAt))

	var body, footer string
	switch m.mode {
	case csModeElements:
		if m.xmlLoading {
			body = m.spinner.View() + " loading elements..."
		} else if len(m.elements) == 0 {
			body = mutedStyle.Render("(no elements found)")
		} else {
			body = m.elementsList.View()
		}
		footer = "esc back, enter open, s summary, x xml"
	case csModeXML:
		if m.xmlLoading {
			body = m.spinner.View() + " loading xml..."
		} else {
			body = m.viewport.View()
		}
		footer = "esc back, s summary, e elements"
	default:
		body = m.viewport.View()
		footer = "esc back, e elements, x xml"
	}
	return header + "\n\n" + body + "\n" + footerStyle.Render(footer)
}

func formatChangesetBody(cs *osm.Changeset) string {
	var sb strings.Builder
	sb.WriteString("Tags:\n")
	for _, t := range cs.Tags {
		fmt.Fprintf(&sb, "  %s = %s\n", t.Key, t.Value)
	}
	if cs.Discussion != nil && len(cs.Discussion.Comments) > 0 {
		sb.WriteString("\nComments:\n")
		for _, c := range cs.Discussion.Comments {
			fmt.Fprintf(&sb, "\n%s by %s:\n%s\n", c.Timestamp.Format(time.RFC3339), c.User, c.Text)
		}
	}
	return sb.String()
}

// extractChangesetElements parses the raw osmChange XML and flattens the
// create/modify/delete sections into a single ordered slice.
func extractChangesetElements(xmlStr string) ([]changesetElement, error) {
	type tagX struct {
		K string `xml:"k,attr"`
		V string `xml:"v,attr"`
	}
	type elemX struct {
		ID      int64  `xml:"id,attr"`
		Version int    `xml:"version,attr"`
		Tags    []tagX `xml:"tag"`
	}
	type sectionX struct {
		Nodes     []elemX `xml:"node"`
		Ways      []elemX `xml:"way"`
		Relations []elemX `xml:"relation"`
	}
	type changeX struct {
		XMLName xml.Name   `xml:"osmChange"`
		Create  []sectionX `xml:"create"`
		Modify  []sectionX `xml:"modify"`
		Delete  []sectionX `xml:"delete"`
	}
	var ch changeX
	if err := xml.Unmarshal([]byte(xmlStr), &ch); err != nil {
		return nil, fmt.Errorf("parse osmChange: %w", err)
	}
	toTags := func(in []tagX) osm.Tags {
		out := make(osm.Tags, len(in))
		for i, t := range in {
			out[i] = osm.Tag{Key: t.K, Value: t.V}
		}
		return out
	}
	var elems []changesetElement
	collect := func(action rune, secs []sectionX) {
		for _, s := range secs {
			for _, n := range s.Nodes {
				elems = append(elems, changesetElement{Kind: "node", ID: n.ID, Version: n.Version, Action: action, Tags: toTags(n.Tags)})
			}
			for _, w := range s.Ways {
				elems = append(elems, changesetElement{Kind: "way", ID: w.ID, Version: w.Version, Action: action, Tags: toTags(w.Tags)})
			}
			for _, r := range s.Relations {
				elems = append(elems, changesetElement{Kind: "relation", ID: r.ID, Version: r.Version, Action: action, Tags: toTags(r.Tags)})
			}
		}
	}
	collect('+', ch.Create)
	collect('~', ch.Modify)
	collect('-', ch.Delete)
	return elems, nil
}
