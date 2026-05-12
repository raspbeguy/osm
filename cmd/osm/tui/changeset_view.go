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
	"github.com/charmbracelet/lipgloss"
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

type prevElemLoadedMsg struct {
	key  string
	prev *prevElement
	err  error
}

type memberDescr struct {
	Type string
	Ref  int64
	Role string
}

type changesetElement struct {
	Kind    string // "node", "way", "relation"
	ID      int64
	Version int
	Action  rune // '+', '~', '-'
	Tags    osm.Tags
	Lat     float64
	Lon     float64
	Nodes   []int64
	Members []memberDescr
}

type prevElement struct {
	Tags    osm.Tags
	Lat     float64
	Lon     float64
	Nodes   []int64
	Members []memberDescr
}

type csElementItem struct{ e changesetElement }

func (i csElementItem) Title() string {
	return fmt.Sprintf("%c %s %d v%d", i.e.Action, kindGlyph(i.e.Kind), i.e.ID, i.e.Version)
}

func (i csElementItem) Description() string { return "" }

func (i csElementItem) FilterValue() string { return i.Title() }

type changesetViewModel struct {
	client         *api.Client
	csID           osm.ChangesetID
	spinner        spinner.Model
	viewport       viewport.Model
	elementsList   list.Model
	detailViewport viewport.Model
	cs             *osm.Changeset
	err            error
	loading        bool
	xml            string
	xmlLoading     bool
	elements       []changesetElement
	mode           csMode
	focus          int // 0=list, 1=detail (only meaningful in elements mode)
	lastSelKey     string
	prevCache      map[string]*prevElement
	prevLoading    map[string]bool
}

func newChangesetView(c *api.Client) changesetViewModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	l := list.New(nil, newCompactDelegate(), 40, 20)
	l.Title = "Elements"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	return changesetViewModel{
		client:         c,
		spinner:        s,
		viewport:       viewport.New(60, 15),
		elementsList:   l,
		detailViewport: viewport.New(40, 20),
		prevCache:      map[string]*prevElement{},
		prevLoading:    map[string]bool{},
	}
}

func (m changesetViewModel) Init() tea.Cmd { return nil }

func (m changesetViewModel) show(id int64) (changesetViewModel, tea.Cmd) {
	m.csID = osm.ChangesetID(id)
	m.loading = true
	m.err = nil
	m.cs = nil
	m.xml = ""
	m.xmlLoading = true
	m.elements = nil
	m.elementsList.SetItems(nil)
	m.mode = csModeElements
	m.focus = 0
	m.lastSelKey = ""
	m.prevCache = map[string]*prevElement{}
	m.prevLoading = map[string]bool{}
	return m, tea.Batch(m.spinner.Tick, m.load(), m.loadXML())
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

func (m changesetViewModel) selectedElement() changesetElement {
	if i, ok := m.elementsList.SelectedItem().(csElementItem); ok {
		return i.e
	}
	return changesetElement{}
}

func prevKey(kind string, id int64, version int) string {
	return fmt.Sprintf("%s/%d/%d", kind, id, version)
}

func (m changesetViewModel) fetchPrev(e changesetElement) tea.Cmd {
	if e.Version <= 1 || e.Action == '+' {
		return nil
	}
	key := prevKey(e.Kind, e.ID, e.Version-1)
	if _, ok := m.prevCache[key]; ok {
		return nil
	}
	if m.prevLoading[key] {
		return nil
	}
	m.prevLoading[key] = true
	client := m.client
	kind, id, v := e.Kind, e.ID, e.Version-1
	return tea.Batch(m.spinner.Tick, func() tea.Msg {
		prev, err := fetchPreviousVersion(client, kind, id, v)
		return prevElemLoadedMsg{key: key, prev: prev, err: err}
	})
}

func fetchPreviousVersion(c *api.Client, kind string, id int64, version int) (*prevElement, error) {
	ctx := context.Background()
	switch kind {
	case "node":
		n, err := c.GetNodeVersion(ctx, osm.NodeID(id), version)
		if err != nil {
			return nil, err
		}
		return &prevElement{Tags: n.Tags, Lat: n.Lat, Lon: n.Lon}, nil
	case "way":
		w, err := c.GetWayVersion(ctx, osm.WayID(id), version)
		if err != nil {
			return nil, err
		}
		refs := make([]int64, len(w.Nodes))
		for i, n := range w.Nodes {
			refs[i] = int64(n.ID)
		}
		return &prevElement{Tags: w.Tags, Nodes: refs}, nil
	case "relation":
		r, err := c.GetRelationVersion(ctx, osm.RelationID(id), version)
		if err != nil {
			return nil, err
		}
		members := make([]memberDescr, len(r.Members))
		for i, mm := range r.Members {
			members[i] = memberDescr{Type: string(mm.Type), Ref: mm.Ref, Role: mm.Role}
		}
		return &prevElement{Tags: r.Tags, Members: members}, nil
	}
	return nil, fmt.Errorf("unknown kind %q", kind)
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
		if m.mode == csModeElements {
			cmd := m.fetchPrev(m.selectedElement())
			return m, cmd
		}
		return m, nil
	case prevElemLoadedMsg:
		delete(m.prevLoading, msg.key)
		if msg.err == nil && msg.prev != nil {
			m.prevCache[msg.key] = msg.prev
		}
		m = m.rewrap()
		return m, nil
	case spinner.TickMsg:
		if !m.loading && !m.xmlLoading && len(m.prevLoading) == 0 {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.KeyMsg:
		if m.cs != nil && m.elementsList.FilterState() != list.Filtering {
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
				return m, m.fetchPrev(m.selectedElement())
			case "x":
				m.mode = csModeXML
				m = m.rewrap()
				if m.xml == "" && !m.xmlLoading {
					m.xmlLoading = true
					return m, tea.Batch(m.spinner.Tick, m.loadXML())
				}
				return m, nil
			}
			if m.mode == csModeElements {
				switch msg.String() {
				case "tab":
					m.focus = 1 - m.focus
					return m, nil
				case "h":
					if e := m.selectedElement(); e.ID != 0 {
						return m, func() tea.Msg {
							return navigateMsg{to: screenHistory, itemID: e.ID, kind: e.Kind, parent: screenChangesetView}
						}
					}
				}
			}
		}
	}

	var cmd tea.Cmd
	if m.mode == csModeElements {
		prevKey := selKey(m.selectedElement())
		if m.focus == 0 {
			m.elementsList, cmd = m.elementsList.Update(msg)
		} else {
			m.detailViewport, cmd = m.detailViewport.Update(msg)
		}
		curKey := selKey(m.selectedElement())
		var fetchCmd tea.Cmd
		if curKey != prevKey {
			fetchCmd = m.fetchPrev(m.selectedElement())
			m = m.rewrap()
		}
		return m, tea.Batch(cmd, fetchCmd)
	}
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func selKey(e changesetElement) string {
	if e.ID == 0 {
		return ""
	}
	return fmt.Sprintf("%s/%d/%d", e.Kind, e.ID, e.Version)
}

func (m changesetViewModel) rewrap() changesetViewModel {
	switch m.mode {
	case csModeXML:
		if m.xml != "" {
			m.viewport.SetContent(wrapText(m.xml, m.viewport.Width))
		}
	case csModeElements:
		m.detailViewport.SetContent(m.renderDetail())
	default:
		if m.cs != nil {
			m.viewport.SetContent(wrapText(formatChangesetBody(m.cs), m.viewport.Width))
		}
	}
	return m
}

func (m changesetViewModel) renderDetail() string {
	e := m.selectedElement()
	if e.ID == 0 {
		return mutedStyle.Render("(no selection)")
	}
	action := "modified"
	switch e.Action {
	case '+':
		action = "created"
	case '-':
		action = "deleted"
	}
	var sb strings.Builder
	sb.WriteString(headerStyle.Render(fmt.Sprintf("%c %s %d v%d", e.Action, kindGlyph(e.Kind), e.ID, e.Version)) + "\n")
	sb.WriteString(mutedStyle.Render(action) + "\n\n")

	if len(e.Tags) > 0 {
		sb.WriteString(headerStyle.Render("Tags") + "\n")
		for _, t := range e.Tags {
			fmt.Fprintf(&sb, "  %s = %s\n", t.Key, t.Value)
		}
		sb.WriteString("\n")
	}

	if e.Action == '~' && e.Version > 1 {
		key := prevKey(e.Kind, e.ID, e.Version-1)
		if m.prevLoading[key] {
			sb.WriteString(mutedStyle.Render("loading previous version...") + "\n")
		} else if prev, ok := m.prevCache[key]; ok {
			sb.WriteString(headerStyle.Render(fmt.Sprintf("Diff vs v%d", e.Version-1)) + "\n")
			diff := formatTagDiff(e.Tags, prev.Tags)
			if diff == "" {
				sb.WriteString(mutedStyle.Render("  (no tag changes)") + "\n")
			} else {
				sb.WriteString(diff)
			}
			if nonTagChanged(e, prev) {
				sb.WriteString("\n" + mutedStyle.Render("  (geometry, refs, or members also changed)") + "\n")
			}
		}
	}
	return wrapText(sb.String(), m.detailViewport.Width)
}

func formatTagDiff(cur, prev osm.Tags) string {
	curMap := map[string]string{}
	for _, t := range cur {
		curMap[t.Key] = t.Value
	}
	prevMap := map[string]string{}
	for _, t := range prev {
		prevMap[t.Key] = t.Value
	}
	var sb strings.Builder
	for k, v := range curMap {
		pv, ok := prevMap[k]
		if !ok {
			fmt.Fprintf(&sb, "  + %s = %s\n", k, v)
		} else if pv != v {
			fmt.Fprintf(&sb, "  ~ %s: %s → %s\n", k, pv, v)
		}
	}
	for k, pv := range prevMap {
		if _, ok := curMap[k]; !ok {
			fmt.Fprintf(&sb, "  - %s = %s\n", k, pv)
		}
	}
	return sb.String()
}

func nonTagChanged(e changesetElement, prev *prevElement) bool {
	switch e.Kind {
	case "node":
		return e.Lat != prev.Lat || e.Lon != prev.Lon
	case "way":
		if len(e.Nodes) != len(prev.Nodes) {
			return true
		}
		for i := range e.Nodes {
			if e.Nodes[i] != prev.Nodes[i] {
				return true
			}
		}
	case "relation":
		if len(e.Members) != len(prev.Members) {
			return true
		}
		for i := range e.Members {
			if e.Members[i] != prev.Members[i] {
				return true
			}
		}
	}
	return false
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
			leftStyle, rightStyle := paneFocused, paneUnfocused
			if m.focus == 1 {
				leftStyle, rightStyle = paneUnfocused, paneFocused
			}
			left := leftStyle.Render(m.elementsList.View())
			right := rightStyle.Render(m.detailViewport.View())
			body = lipgloss.JoinHorizontal(lipgloss.Top, left, right)
		}
		footer = "esc back, tab swap pane, h history, s summary, x xml"
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
// create/modify/delete sections into an ordered slice with tag and structural
// info needed for diffing.
func extractChangesetElements(xmlStr string) ([]changesetElement, error) {
	type tagX struct {
		K string `xml:"k,attr"`
		V string `xml:"v,attr"`
	}
	type ndX struct {
		Ref int64 `xml:"ref,attr"`
	}
	type memX struct {
		Type string `xml:"type,attr"`
		Ref  int64  `xml:"ref,attr"`
		Role string `xml:"role,attr"`
	}
	type elemX struct {
		ID      int64   `xml:"id,attr"`
		Version int     `xml:"version,attr"`
		Lat     float64 `xml:"lat,attr"`
		Lon     float64 `xml:"lon,attr"`
		Tags    []tagX  `xml:"tag"`
		Nodes   []ndX   `xml:"nd"`
		Members []memX  `xml:"member"`
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
	toRefs := func(in []ndX) []int64 {
		out := make([]int64, len(in))
		for i, n := range in {
			out[i] = n.Ref
		}
		return out
	}
	toMembers := func(in []memX) []memberDescr {
		out := make([]memberDescr, len(in))
		for i, m := range in {
			out[i] = memberDescr{Type: m.Type, Ref: m.Ref, Role: m.Role}
		}
		return out
	}
	var elems []changesetElement
	collect := func(action rune, secs []sectionX) {
		for _, s := range secs {
			for _, n := range s.Nodes {
				elems = append(elems, changesetElement{Kind: "node", ID: n.ID, Version: n.Version, Action: action, Tags: toTags(n.Tags), Lat: n.Lat, Lon: n.Lon})
			}
			for _, w := range s.Ways {
				elems = append(elems, changesetElement{Kind: "way", ID: w.ID, Version: w.Version, Action: action, Tags: toTags(w.Tags), Nodes: toRefs(w.Nodes)})
			}
			for _, r := range s.Relations {
				elems = append(elems, changesetElement{Kind: "relation", ID: r.ID, Version: r.Version, Action: action, Tags: toTags(r.Tags), Members: toMembers(r.Members)})
			}
		}
	}
	collect('+', ch.Create)
	collect('~', ch.Modify)
	collect('-', ch.Delete)
	return elems, nil
}
