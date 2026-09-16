package main

import (
	"fmt"
	"time"

	"github.com/TAbelhaDev/tabelhatuiui"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	headerLines = 1
	footerLines = 1
	boxOverhead = 2 + 1 // border + title
	listLimit   = 500   // sane cap for the TUI feed; item.list --json has no such cap
)

// appModel is tarecap's single view: a chronological feed of items reported
// by other tools via `tarecap ipc item.add`. There is no pull/aggregation
// here — items only ever come from the store other processes already wrote.
type appModel struct {
	store   *Store
	openErr string

	items      []Item // currently loaded (filtered) items, in table row order
	sources    []string
	unseenOnly bool
	sourceIdx  int // 0 = "todas as fontes", 1..len(sources) = sources[i-1]

	tbl table.Model

	width  int
	height int
	status string

	helpModal     *tuiui.HelpModal
	settingsModal *tuiui.SettingsModal
}

func newModel() appModel {
	_ = reg.Load()

	m := appModel{
		helpModal: tuiui.NewHelpModal(tuiui.HelpSection{
			Title:      "Atalhos",
			BindingsFn: reg.Bindings,
		}),
		settingsModal: tuiui.NewSettingsModal(reg),
	}

	m.tbl = table.New(table.WithFocused(true))
	// Placeholder columns: SetRows renders eagerly and panics (index out of
	// range) if the row shape doesn't match the current columns. Real widths
	// come from layout() once the first WindowSizeMsg arrives; the column
	// *count* here must already match refreshTable's 4-cell rows.
	m.tbl.SetColumns([]table.Column{
		{Title: "", Width: 1},
		{Title: "Fonte", Width: 18},
		{Title: "Novidade", Width: 40},
		{Title: "Quando", Width: 12},
	})
	m.applyStyles()

	store, err := openStore()
	if err != nil {
		m.openErr = err.Error()
		return m
	}
	m.store = store
	m.reload()
	return m
}

func (m *appModel) applyStyles() {
	styles := table.DefaultStyles()
	styles.Header = styles.Header.
		Foreground(colSubtext0).
		Background(colMantle).
		Bold(true)
	styles.Selected = styles.Selected.
		Foreground(colBase).
		Background(colPrimary).
		Bold(true)
	// Cell intentionally has no Background: bubbles/table renders each cell
	// individually and only wraps the whole row in Selected afterwards, same
	// reasoning as tabelharadar's table styling — a background baked into
	// every cell's own ANSI codes would win over Selected's on the row.
	m.tbl.SetStyles(styles)
}

// currentSourceFilter returns "" (todas as fontes) or the selected source.
func (m *appModel) currentSourceFilter() string {
	if m.sourceIdx <= 0 || m.sourceIdx > len(m.sources) {
		return ""
	}
	return m.sources[m.sourceIdx-1]
}

// reload re-reads the store: the full source list (for the cycle filter) and
// the currently filtered item feed. Called after every mutation (seen/
// seen-all) and on "r", since any external process may have added items
// since the last read.
func (m *appModel) reload() {
	if m.store == nil {
		return
	}
	if sources, err := m.store.sources(); err == nil {
		m.sources = sources
	}
	if m.sourceIdx > len(m.sources) {
		m.sourceIdx = 0
	}

	items, err := m.store.list(m.unseenOnly, m.currentSourceFilter(), listLimit)
	if err != nil {
		m.status = "erro: " + err.Error()
		return
	}
	m.items = items
	m.refreshTable()
	m.setStatus()
}

func (m *appModel) setStatus() {
	unseen := 0
	for _, it := range m.items {
		if it.SeenAt == nil {
			unseen++
		}
	}
	filter := "todas as fontes"
	if src := m.currentSourceFilter(); src != "" {
		filter = "fonte: " + src
	}
	scope := "tudo"
	if m.unseenOnly {
		scope = "não vistos"
	}
	m.status = fmt.Sprintf("%d novidades não vistas · %s · %s", unseen, scope, filter)
}

func (m *appModel) refreshTable() {
	rows := make([]table.Row, len(m.items))
	for i, it := range m.items {
		marker := "○"
		if it.SeenAt == nil {
			marker = "●"
		}
		rows[i] = table.Row{marker, it.Source, it.Title, humanizeAgo(it.CreatedAt)}
	}
	m.tbl.SetRows(rows)
	if m.tbl.Cursor() >= len(rows) && len(rows) > 0 {
		m.tbl.SetCursor(len(rows) - 1)
	}
}

func (m *appModel) current() *Item {
	idx := m.tbl.Cursor()
	if idx < 0 || idx >= len(m.items) {
		return nil
	}
	return &m.items[idx]
}

func humanizeAgo(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "agora"
	case d < time.Hour:
		return fmt.Sprintf("%dm atrás", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh atrás", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd atrás", int(d.Hours()/24))
	default:
		return fmt.Sprintf("%dm atrás", int(d.Hours()/24/30))
	}
}

func (m *appModel) layout() {
	if m.width == 0 || m.height == 0 {
		return
	}
	// -4: Panel's border (2) + its own Padding(0,1) (2) — the exact off-by-4
	// tabelharadar's layout comment already documents for the same box.
	innerW := m.width - 4
	if innerW < 20 {
		innerW = 20
	}
	// marker(1) + source(18) + age(12), the rest goes to title. Each column
	// carries bubbles/table's own Padding(0,1) on top of these widths.
	markerW, sourceW, ageW := 1, 18, 12
	titleW := innerW - markerW - sourceW - ageW - 4*2 // 4 columns' worth of padding
	if titleW < 10 {
		titleW = 10
	}
	m.tbl.SetColumns([]table.Column{
		{Title: "", Width: markerW},
		{Title: "Fonte", Width: sourceW},
		{Title: "Novidade", Width: titleW},
		{Title: "Quando", Width: ageW},
	})
	m.tbl.SetWidth(innerW)

	bodyHeight := m.height - headerLines - footerLines - boxOverhead
	if bodyHeight < 3 {
		bodyHeight = 3
	}
	m.tbl.SetHeight(bodyHeight)
}

func (m appModel) Init() tea.Cmd { return nil }

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if sizeMsg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width, m.height = sizeMsg.Width, sizeMsg.Height
		m.helpModal.SetSize(sizeMsg.Width, sizeMsg.Height)
		m.settingsModal.SetSize(sizeMsg.Width, sizeMsg.Height)
		m.layout()
		return m, nil
	}

	if m.settingsModal.Update(msg) {
		return m, nil
	}
	if m.helpModal.Update(msg) {
		return m, nil
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m.forwardToTable(msg)
	}

	switch {
	case key.Matches(keyMsg, resolve("quit")):
		return m, tea.Quit
	case key.Matches(keyMsg, resolve("help")):
		m.helpModal.Toggle()
		return m, nil
	case key.Matches(keyMsg, resolve("settings")):
		m.settingsModal.Toggle()
		return m, nil
	case key.Matches(keyMsg, resolve("refresh")):
		m.reload()
		m.layout()
		return m, nil
	case key.Matches(keyMsg, resolve("seen")):
		m.markCurrentSeen()
		return m, nil
	case key.Matches(keyMsg, resolve("seen_all")):
		m.markAllSeen()
		return m, nil
	case key.Matches(keyMsg, resolve("toggle_unseen")):
		m.unseenOnly = !m.unseenOnly
		m.reload()
		return m, nil
	case key.Matches(keyMsg, resolve("cycle_source")):
		m.sourceIdx = (m.sourceIdx + 1) % (len(m.sources) + 1)
		m.reload()
		return m, nil
	}

	return m.forwardToTable(msg)
}

func (m appModel) forwardToTable(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.tbl, cmd = m.tbl.Update(msg)
	return m, cmd
}

func (m *appModel) markCurrentSeen() {
	if m.store == nil {
		return
	}
	item := m.current()
	if item == nil {
		return
	}
	if _, err := m.store.markSeen(item.ID); err != nil {
		m.status = "erro: " + err.Error()
		return
	}
	m.reload()
}

func (m *appModel) markAllSeen() {
	if m.store == nil {
		return
	}
	if _, err := m.store.markAllSeen(); err != nil {
		m.status = "erro: " + err.Error()
		return
	}
	m.reload()
}

func (m appModel) View() string {
	if m.width == 0 {
		return ""
	}

	header := theme.Header(m.width).Render("tarecap — novidades")
	footer := tuiui.NewFooter(reg.Bindings()...).Status(m.status).Render(m.width, theme)

	var body string
	if m.openErr != "" {
		body = theme.Panel(true).Render(tuiui.PadLines(
			theme.Error().Render("erro abrindo o banco: "+m.openErr), m.width-4,
		))
	} else if len(m.items) == 0 {
		body = theme.Panel(true).Render(tuiui.PadLines(
			theme.Dim().Render("nenhuma novidade ainda — outras ferramentas registram itens via `tarecap ipc item.add`"), m.width-4,
		))
	} else {
		body = theme.Panel(true).Render(m.tbl.View())
	}

	view := lipgloss.JoinVertical(lipgloss.Left, header, body, footer)

	if m.settingsModal.Visible() {
		return m.settingsModal.View(theme)
	}
	if m.helpModal.Visible() {
		return m.helpModal.View(theme)
	}
	return view
}
