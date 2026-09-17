package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/TAbelhaDev/tabelhatuiui"
	"github.com/TAbelhaDev/tabelhatuiui/markdown"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	headerLines = 1
	footerLines = 1

	panelGap        = 1
	listBoxOverhead = 2 + 1 // border + title
	metaBoxOverhead = 2 + 1
	descBoxOverhead = 2 + 1
	minListRows     = 3
	minMetaLines    = 4
	minDescLines    = 4
	minListWidth    = 20
	minRightWidth   = 40
	// metaFixedLines is the default metadata panel content budget (fonte,
	// quando, status — link is conditional).
	metaFixedLines = 3

	listLimit = 500
)

// panelFocus selects which of the two interactive panels — the feed list,
// or the description panel — currently receives j/k. The metadata panel
// (top-right) is display-only, never a focus target.
type panelFocus int

const (
	focusList panelFocus = iota
	focusDesc
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

	tbl    table.Model
	focus  panelFocus
	descVP *markdown.Panel

	width  int
	height int
	status string

	// 3-panel layout budget, recomputed by layout() on every resize.
	threePanel      bool
	listInnerWidth  int
	rightInnerWidth int
	listRowsHeight  int
	metaLines       int
	descMaxLines    int

	helpModal     *tuiui.HelpModal
	settingsModal *tuiui.SettingsModal
}

func newModel() appModel {
	_ = reg.Load()

	m := appModel{
		descVP: markdown.NewPanel(),
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
	if m.descVP != nil {
		m.descVP.Viewport().Reset()
	}
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

// refreshTable rebuilds the table rows from m.items. Always produces
// 4-cell rows (marker, fonte, title, quando) so both 3-panel mode
// (3 columns, 4th cell ignored) and narrow fallback (4 columns) work.
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

// layout recomputes the list/metadata/description panel widths and heights
// so the whole 3-panel view always fits exactly within m.height — mirrors
// taglue/tui.go's layout(): fixed line budgets per panel instead of
// letting lipgloss stretch content past what fits.
func (m *appModel) layout() {
	if m.width == 0 || m.height == 0 {
		return
	}

	// Check if terminal is wide enough for 3-panel mode.
	totalRowWidth := m.width - panelGap
	minRow := (minListWidth + 4) + (minRightWidth + 4)
	m.threePanel = totalRowWidth >= minRow

	if !m.threePanel {
		// Narrow fallback: single full-width table, same as before.
		innerW := m.width - 4
		if innerW < 20 {
			innerW = 20
		}
		markerW, sourceW, ageW := 1, 18, 12
		titleW := innerW - markerW - sourceW - ageW - 4*2
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
		bodyHeight := m.height - headerLines - footerLines - listBoxOverhead
		if bodyHeight < 3 {
			bodyHeight = 3
		}
		m.tbl.SetHeight(bodyHeight)
		m.refreshTable()
		return
	}

	// 3-panel mode: left list + right meta/desc (taglue layout).
	listBoxWidth := totalRowWidth / 3
	rightBoxWidth := totalRowWidth - listBoxWidth

	m.listInnerWidth = listBoxWidth - 4
	if m.listInnerWidth < minListWidth {
		m.listInnerWidth = minListWidth
	}
	m.rightInnerWidth = rightBoxWidth - 4
	if m.rightInnerWidth < minRightWidth {
		m.rightInnerWidth = minRightWidth
	}

	bodyHeight := m.height - headerLines - footerLines
	minBody := metaBoxOverhead + minMetaLines + descBoxOverhead + minDescLines
	if minListBody := listBoxOverhead + minListRows; minListBody > minBody {
		minBody = minListBody
	}
	if bodyHeight < minBody {
		bodyHeight = minBody
	}

	// Dynamic meta sizing: use the selected item's actual content length,
	// clamped between minMetaLines and (bodyHeight - minDescLines).
	metaContentLines := metaFixedLines
	item := m.current()
	if item != nil {
		metaContentLines = len(m.computeMetaLines())
	}
	metaBoxHeight := metaBoxOverhead + metaContentLines
	if maxMeta := bodyHeight - (descBoxOverhead + minDescLines); metaBoxHeight > maxMeta {
		metaBoxHeight = maxMeta
	}
	if metaBoxHeight < metaBoxOverhead+minMetaLines {
		metaBoxHeight = metaBoxOverhead + minMetaLines
	}
	descBoxHeight := bodyHeight - metaBoxHeight
	if descBoxHeight < descBoxOverhead+minDescLines {
		descBoxHeight = descBoxOverhead + minDescLines
	}

	m.metaLines = metaBoxHeight - metaBoxOverhead
	m.descMaxLines = descBoxHeight - descBoxOverhead
	m.descVP.Viewport().SetHeight(m.descMaxLines)

	// The list panel spans both right-column boxes stacked together, so its
	// row budget must match their combined (post-clamp) height exactly or
	// the borders won't line up at the bottom.
	m.listRowsHeight = (metaBoxHeight + descBoxHeight) - listBoxOverhead
	if m.listRowsHeight < minListRows {
		m.listRowsHeight = minListRows
	}

	// Set table columns and size for 3-panel mode (4 columns always;
	// Quando has width 0 so it's skipped — avoids row/cell count mismatch).
	markerW, sourceW, ageW := 1, 18, 0
	titleW := m.listInnerWidth - markerW - sourceW - 3*2 // 3 visible columns' padding
	if titleW < 10 {
		titleW = 10
	}
	m.tbl.SetColumns([]table.Column{
		{Title: "", Width: markerW},
		{Title: "Fonte", Width: sourceW},
		{Title: "Novidade", Width: titleW},
		{Title: "Quando", Width: ageW},
	})
	m.tbl.SetWidth(m.listInnerWidth)
	m.tbl.SetHeight(m.listRowsHeight)
}

// computeMetaLines builds the metadata panel lines for the current item.
func (m *appModel) computeMetaLines() []string {
	item := m.current()
	if item == nil {
		return []string{theme.Dim().Render("nenhum item selecionado")}
	}
	lines := []string{
		"fonte: " + item.Source,
		"quando: " + item.CreatedAt.Format("2006-01-02 15:04"),
	}
	if item.SeenAt == nil {
		lines = append(lines, "status: não visto")
	} else {
		lines = append(lines, "status: visto")
	}
	if item.Link != "" {
		lines = append(lines, "link: "+item.Link)
	}
	return lines
}

// renderListPanel wraps the table in a bordered panel with focus highlight.
func (m *appModel) renderListPanel() string {
	return theme.Panel(m.focus == focusList).Render(m.tbl.View())
}

// renderMetaPanel renders the metadata panel (source, timestamp, status, link).
func (m *appModel) renderMetaPanel() string {
	title := theme.Title().Render("metadados")
	lines := m.computeMetaLines()
	body := strings.Join(lines, "\n")
	body = tuiui.WrapText(body, m.rightInnerWidth)
	content := title + "\n" + tuiui.PadToHeight(body, m.metaLines)
	content = tuiui.PadLines(content, m.rightInnerWidth)
	return theme.Panel(false).Render(content)
}

// renderDescPanel renders the scrollable markdown description panel.
func (m *appModel) renderDescPanel() string {
	m.descVP.Focus(m.focus == focusDesc)

	item := m.current()
	m.descVP.SetTitle("descrição")

	if item == nil {
		m.descVP.SetMarkdown("", m.rightInnerWidth, theme)
		return m.descVP.View(theme, m.rightInnerWidth+4)
	}

	body := item.Body
	if body == "" {
		body = "sem descrição — o remetente só mandou o título"
	}
	m.descVP.SetMarkdown(body, m.rightInnerWidth, theme)
	return m.descVP.View(theme, m.rightInnerWidth+4)
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

	// Global keys work regardless of focus.
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

	// Focus switching: ctrl+h → list, ctrl+l → desc.
	if key.Matches(keyMsg, resolve("focus")) {
		focusKeys := resolve("focus").Keys()
		switch {
		case len(focusKeys) > 0 && keyMsg.String() == focusKeys[0]:
			m.focus = focusList
		case len(focusKeys) > 1 && keyMsg.String() == focusKeys[1]:
			if m.threePanel {
				m.focus = focusDesc
			}
		}
		return m, nil
	}

	// When desc is focused, j/k/scroll go to the viewport.
	if m.focus == focusDesc && m.threePanel {
		if m.descVP.Viewport().Update(keyMsg) {
			return m, nil
		}
		return m, nil
	}

	return m.forwardToTable(msg)
}

func (m appModel) forwardToTable(msg tea.Msg) (tea.Model, tea.Cmd) {
	prevCursor := m.tbl.Cursor()
	var cmd tea.Cmd
	m.tbl, cmd = m.tbl.Update(msg)
	if m.tbl.Cursor() != prevCursor {
		m.descVP.Viewport().Reset()
	}
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
	} else if m.threePanel {
		listBox := m.renderListPanel()
		metaBox := m.renderMetaPanel()
		descBox := m.renderDescPanel()
		rightCol := lipgloss.JoinVertical(lipgloss.Left, metaBox, descBox)
		body = lipgloss.JoinHorizontal(lipgloss.Top, listBox, strings.Repeat(" ", panelGap), rightCol)
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
