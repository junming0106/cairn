// Package ui 是 cairn 的 Bubble Tea 介面：完成紀錄 / 進行中 兩個頁籤。
package ui

import (
	"fmt"
	"strings"
	"time"

	"cairn/internal/store"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// 外部（AI）寫入紀錄檔後多久會反映到畫面上。
const reloadInterval = 2 * time.Second

type mode int

const (
	modeList mode = iota
	modeAddTask
	modeAddNote
)

// 頁籤。
const (
	tabDone = iota
	tabActive
)

var tabNames = []string{"完成紀錄", "進行中"}

// Model 是 TUI 狀態。
type Model struct {
	path   string
	log    *store.Log
	mtime  time.Time
	tab    int
	cursor [2]int // 每個頁籤各自記游標
	offset [2]int
	mode   mode
	input  textinput.Model
	detail viewport.Model
	width  int
	height int
	status string
	err    error
	ready  bool
}

// New 建立初始 Model。
func New(path string, l *store.Log) Model {
	ti := textinput.New()
	ti.CharLimit = 500
	return Model{path: path, log: l, mtime: store.ModTime(path), input: ti}
}

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(reloadInterval, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// Init 啟動輪詢，讓 AI 在背景寫入的進度自動出現。
func (m Model) Init() tea.Cmd { return tick() }

// visible 回傳目前頁籤要顯示的任務。
func (m Model) visible() []*store.Task {
	if m.tab == tabDone {
		return m.log.Done()
	}
	return m.log.Active()
}

func (m Model) current() *store.Task {
	v := m.visible()
	if len(v) == 0 || m.cursor[m.tab] >= len(v) {
		return nil
	}
	return v[m.cursor[m.tab]]
}

func (m *Model) save() {
	if err := store.Save(m.path, m.log); err != nil {
		m.err = err
		return
	}
	m.mtime = store.ModTime(m.path)
}

// reloadIfChanged 在檔案被外部改動時重新載入，並盡量保留游標位置。
func (m *Model) reloadIfChanged() {
	mt := store.ModTime(m.path)
	if mt.Equal(m.mtime) {
		return
	}
	l, err := store.Load(m.path)
	if err != nil {
		m.err = err
		return
	}
	var keep string
	if t := m.current(); t != nil {
		keep = t.ID
	}
	m.log = l
	m.mtime = mt
	m.err = nil
	m.status = "已同步外部更新 " + mt.Format("15:04:05")
	m.cursor[m.tab] = 0
	for i, t := range m.visible() {
		if t.ID == keep {
			m.cursor[m.tab] = i
			break
		}
	}
	m.clampCursor()
}

func (m *Model) clampCursor() {
	n := len(m.visible())
	c, o := m.cursor[m.tab], m.offset[m.tab]
	if c >= n {
		c = n - 1
	}
	if c < 0 {
		c = 0
	}
	rows := m.listRows()
	if c < o {
		o = c
	}
	if rows > 0 && c >= o+rows {
		o = c - rows + 1
	}
	if o < 0 {
		o = 0
	}
	m.cursor[m.tab], m.offset[m.tab] = c, o
}

// Update 處理事件。
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.detail = viewport.New(m.detailWidth(), m.detailHeight())
		m.ready = true
		m.clampCursor()
		return m, nil

	case tickMsg:
		m.reloadIfChanged()
		return m, tick()

	case tea.KeyMsg:
		if m.mode != modeList {
			return m.updateInput(msg)
		}
		return m.updateList(msg)
	}
	return m, nil
}

func (m Model) updateInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		m.mode = modeList
		m.input.Blur()
		m.input.SetValue("")
		return m, nil
	case "enter":
		val := strings.TrimSpace(m.input.Value())
		if val != "" {
			switch m.mode {
			case modeAddTask:
				t := m.log.AddTask(val, store.KindFeature, nil)
				m.save()
				m.tab = tabActive
				for i, v := range m.visible() {
					if v.ID == t.ID {
						m.cursor[m.tab] = i
					}
				}
				m.status = "已新增 " + t.ID
			case modeAddNote:
				if t := m.current(); t != nil {
					t.AddEntry(val, nil, nil)
					if t.Status == store.StatusTodo {
						t.SetStatus(store.StatusInProgress)
					}
					m.save()
					m.status = "已紀錄步驟到 " + t.ID
				}
			}
		}
		m.mode = modeList
		m.input.Blur()
		m.input.SetValue("")
		m.clampCursor()
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *Model) setTab(t int) {
	if t != m.tab {
		m.tab = t
		m.status = ""
		m.clampCursor()
	}
}

func (m Model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	// ── 頁籤 ──
	case "tab", "l", "right":
		m.setTab((m.tab + 1) % len(tabNames))
	case "shift+tab", "h", "left":
		m.setTab((m.tab + len(tabNames) - 1) % len(tabNames))
	case "1":
		m.setTab(tabDone)
	case "2":
		m.setTab(tabActive)

	// ── 清單 ──
	case "j", "down":
		m.cursor[m.tab]++
		m.clampCursor()
	case "k", "up":
		m.cursor[m.tab]--
		m.clampCursor()
	case "g", "home":
		m.cursor[m.tab] = 0
		m.clampCursor()
	case "G", "end":
		m.cursor[m.tab] = len(m.visible()) - 1
		m.clampCursor()

	// ── 右側詳情捲動 ──
	case "J":
		m.detail.LineDown(1)
	case "K":
		m.detail.LineUp(1)
	case "ctrl+d", "pgdown":
		m.detail.HalfViewDown()
	case "ctrl+u", "pgup":
		m.detail.HalfViewUp()

	// ── 編輯 ──
	case "a":
		m.mode = modeAddTask
		m.input.Placeholder = "新功能標題"
		m.input.Focus()
		m.status = ""
	case "n":
		if m.current() != nil {
			m.mode = modeAddNote
			m.input.Placeholder = "這個步驟做了什麼"
			m.input.Focus()
			m.status = ""
		}
	case "s":
		if t := m.current(); t != nil {
			was := m.tab
			t.SetStatus(store.NextStatus(t.Status))
			m.save()
			if t.Status == store.StatusDone {
				m.setTab(tabDone)
			} else if was == tabDone {
				m.setTab(tabActive)
			}
			for i, v := range m.visible() {
				if v.ID == t.ID {
					m.cursor[m.tab] = i
				}
			}
			m.clampCursor()
			m.status = t.ID + " → " + statusLabel[t.Status]
		}
	case "r":
		m.mtime = time.Time{}
		m.reloadIfChanged()
	}
	return m, nil
}

// ── 樣式 ──────────────────────────────────────────────

var (
	colDim    = lipgloss.AdaptiveColor{Light: "245", Dark: "243"}
	colText   = lipgloss.AdaptiveColor{Light: "236", Dark: "252"}
	colAccent = lipgloss.AdaptiveColor{Light: "62", Dark: "111"}
	colOn     = lipgloss.AdaptiveColor{Light: "255", Dark: "233"} // 有底色時的文字

	bgHeader = lipgloss.AdaptiveColor{Light: "254", Dark: "236"}
	bgSelect = lipgloss.AdaptiveColor{Light: "189", Dark: "24"}
	bgLabel  = lipgloss.AdaptiveColor{Light: "252", Dark: "238"}

	statusHue = map[string]lipgloss.AdaptiveColor{
		store.StatusTodo:       {Light: "244", Dark: "245"},
		store.StatusInProgress: {Light: "32", Dark: "39"},
		store.StatusBlocked:    {Light: "166", Dark: "209"},
		store.StatusDone:       {Light: "28", Dark: "77"},
	}
	statusMark = map[string]string{
		store.StatusTodo:       "○",
		store.StatusInProgress: "◐",
		store.StatusBlocked:    "▲",
		store.StatusDone:       "●",
	}
	statusLabel = map[string]string{
		store.StatusTodo:       "待辦",
		store.StatusInProgress: "進行中",
		store.StatusBlocked:    "卡住",
		store.StatusDone:       "完成",
	}

	kindHue = map[string]lipgloss.AdaptiveColor{
		store.KindFeature:  {Light: "26", Dark: "68"},
		store.KindFix:      {Light: "160", Dark: "167"},
		store.KindRefactor: {Light: "97", Dark: "104"},
		store.KindDocs:     {Light: "243", Dark: "245"},
	}
	kindLabel = map[string]string{
		store.KindFeature:  "新功能",
		store.KindFix:      "修正",
		store.KindRefactor: "重構",
		store.KindDocs:     "文件",
	}

	// mascot 是 cairn 的吉祥物：一疊小石頭。想換成自己的圖案的話，
	// 只要保持每一行的顯示寬度一致就好（這裡是 9 欄、4 行）。
	mascot = []string{
		" ╭─────╮ ",
		" │ • • │ ",
		"╭┴─────┴╮",
		"╰───────╯",
	}

	// wordmark 是 cairn 的字體招牌（3 行）。
	wordmark = []string{
		"┌─┐┌─┐┬┬─┐┌┐┌",
		"│  ├─┤│├┬┘│││",
		"└─┘┴ ┴┴┴└─┘└┘",
	}

	dimStyle  = lipgloss.NewStyle().Foreground(colDim)
	textStyle = lipgloss.NewStyle().Foreground(colText)
	paneStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colDim).Padding(0, 1)
)

// chip 是一個有底色的小標籤，左右各留一個空白。
func chip(text string, bg, fg lipgloss.TerminalColor, bold bool) string {
	return lipgloss.NewStyle().Background(bg).Foreground(fg).Bold(bold).Padding(0, 1).Render(text)
}

// bar 是一條填滿指定寬度的反白列。傳入的字串必須是純文字（不含 ANSI）。
func bar(text string, w int, bg, fg lipgloss.TerminalColor, bold bool) string {
	return lipgloss.NewStyle().Background(bg).Foreground(fg).Bold(bold).
		Width(w).MaxWidth(w).Render(text)
}

// ── 版面 ──────────────────────────────────────────────

// narrow 為真時改用上下堆疊版面（例如 tmux 分割出來的窄 pane）。
func (m Model) narrow() bool { return m.width < 76 }

func (m Model) listWidth() int {
	if m.narrow() {
		return m.width - 4
	}
	w := m.width * 36 / 100
	if w < 26 {
		w = 26
	}
	if w > 46 {
		w = 46
	}
	return w
}

func (m Model) detailWidth() int {
	if m.narrow() {
		return m.width - 4
	}
	w := m.width - m.listWidth() - 6 // 兩個窗格的框線與 padding
	if w < 20 {
		w = 20
	}
	return w
}

// bannerOn 決定要顯示大招牌還是單行標題（窄畫面或矮視窗時退回單行）。
func (m Model) bannerOn() bool { return !m.narrow() && m.height >= 28 }

// chrome 是窗格以外佔掉的行數：標題區 + 頁籤列 + 提示列 + 框線。
func (m Model) chrome() int {
	if m.bannerOn() {
		return 10
	}
	return 7
}

// paneHeight 是版面能給窗格內容的總高度（扣掉標題列、頁籤列、提示列、框線）。
func (m Model) paneHeight() int {
	h := m.height - m.chrome()
	if h < 3 {
		h = 3
	}
	return h
}

// listRows 是清單窗格的內容高度。每個項目佔兩行。
func (m Model) listRows() int {
	if !m.narrow() {
		return m.paneHeight()
	}
	r := m.paneHeight() / 4
	if r < 4 {
		r = 4
	}
	if r%2 == 1 {
		r++
	}
	if r > m.paneHeight()-4 {
		r = m.paneHeight() - 4
	}
	if r < 2 {
		r = 2
	}
	return r
}

// detailHeight 是詳情窗格的內容高度。
func (m Model) detailHeight() int {
	if !m.narrow() {
		return m.paneHeight()
	}
	h := m.paneHeight() - m.listRows() - 2 // 扣掉多出來的一組框線
	if h < 3 {
		h = 3
	}
	return h
}

// View 繪製畫面。
func (m Model) View() string {
	if !m.ready {
		return "載入中…"
	}
	var body string
	if m.narrow() {
		body = lipgloss.JoinVertical(lipgloss.Left, m.renderList(), m.renderDetail())
	} else {
		body = lipgloss.JoinHorizontal(lipgloss.Top, m.renderList(), m.renderDetail())
	}
	head := m.renderHeader()
	if m.bannerOn() {
		head = m.renderBanner()
	}
	return strings.Join([]string{head, m.renderTabs(), body, m.renderFooter()}, "\n")
}

// counts 是「完成 N ・ 未完成 M」這串字。
func (m Model) counts() string {
	c := m.log.Counts()
	return fmt.Sprintf("完成 %d ・ 未完成 %d", c[store.StatusDone], len(m.log.Tasks)-c[store.StatusDone])
}

func (m Model) projectName() string {
	if m.log.Project == "" {
		return "未命名專案"
	}
	return m.log.Project
}

// renderBanner 畫出吉祥物 + 字體招牌 + 專案名稱。
func (m Model) renderBanner() string {
	pet := lipgloss.NewStyle().Foreground(colAccent)
	mark := lipgloss.NewStyle().Foreground(colText).Bold(true)

	lines := make([]string, len(mascot))
	for i := range mascot {
		lines[i] = " " + pet.Render(mascot[i]) + "  "
		switch {
		case i < len(wordmark):
			lines[i] += mark.Render(wordmark[i])
		default: // 最後一行放專案名稱與計數
			lines[i] += chip("["+m.projectName()+"]", bgLabel, colText, true) +
				dimStyle.Render("  "+m.counts())
		}
	}
	return strings.Join(lines, "\n")
}

// renderHeader 是矮視窗或窄畫面時的單行標題。
func (m Model) renderHeader() string {
	brand := chip("cairn", colAccent, colOn, true)
	rest := " [" + m.projectName() + "]   " + m.counts()
	w := m.width - lipgloss.Width(brand)
	if w < 1 {
		w = 1
	}
	return brand + bar(rest, w, bgHeader, colText, false)
}

func (m Model) renderTabs() string {
	counts := []int{len(m.log.Done()), len(m.log.Active())}
	segs := make([]string, len(tabNames))
	for i, name := range tabNames {
		label := fmt.Sprintf("%s %d", name, counts[i])
		if i == m.tab {
			segs[i] = chip(label, colAccent, colOn, true)
		} else {
			segs[i] = chip(label, bgLabel, colDim, false)
		}
	}
	return " " + strings.Join(segs, " ")
}

func (m Model) renderList() string {
	w := m.listWidth()
	cw := w - 2 // 扣掉窗格左右 padding
	v := m.visible()
	rows := []string{}
	if len(v) == 0 {
		if m.tab == tabDone {
			rows = append(rows, dimStyle.Render("（還沒有完成的功能）"))
		} else {
			rows = append(rows, dimStyle.Render("（沒有進行中的工作，按 a 新增）"))
		}
	}
	end := m.offset[m.tab] + m.listRows()
	if end > len(v) {
		end = len(v)
	}
	for i := m.offset[m.tab]; i < end; i++ {
		t := v[i]
		when := t.CompletedAt().Format("01-02")
		if m.tab != tabDone {
			when = humanizeShort(t.Updated)
		}
		title := truncate(t.Title, cw-4)
		meta := fmt.Sprintf("%s  %s  %s", t.ID, kindLabel[t.Kind], when)

		if i == m.cursor[m.tab] {
			// 選取項目整塊反白，兩行同一個底色
			rows = append(rows, bar(" "+statusMark[t.Status]+" "+title, cw, bgSelect, colText, true))
			rows = append(rows, bar("   "+meta, cw, bgSelect, colText, false))
		} else {
			mark := lipgloss.NewStyle().Foreground(statusHue[t.Status]).Render(statusMark[t.Status])
			rows = append(rows, " "+mark+" "+textStyle.Render(title))
			rows = append(rows, dimStyle.Render("   "+t.ID+"  ")+
				lipgloss.NewStyle().Foreground(kindHue[t.Kind]).Render(kindLabel[t.Kind])+
				dimStyle.Render("  "+when))
		}
	}
	for len(rows) < m.listRows() {
		rows = append(rows, "")
	}
	if len(rows) > m.listRows() {
		rows = rows[:m.listRows()]
	}
	return paneStyle.Width(w).Height(m.listRows()).Render(strings.Join(rows, "\n"))
}

// section 產生「反白欄位標題 + 縮排內文」的區塊。
func (m Model) section(label, body string) string {
	if strings.TrimSpace(body) == "" {
		return ""
	}
	return chip(label, bgLabel, colText, true) + "\n" +
		indent(wrap(body, m.detailWidth()-4), "  ") + "\n\n"
}

func (m Model) renderDetail() string {
	t := m.current()
	var b strings.Builder
	switch {
	case t == nil:
		b.WriteString(dimStyle.Render("選一個項目查看詳情。"))

	case t.Status == store.StatusDone:
		b.WriteString(chip("功能名稱", bgLabel, colText, true) + "\n")
		for _, line := range strings.Split(wrap(t.Title, m.detailWidth()-4), "\n") {
			b.WriteString(bar("▌ "+line, m.detailWidth()-2, bgSelect, colText, true) + "\n")
		}
		b.WriteString("\n")
		b.WriteString(dimStyle.Render(t.ID) + " " +
			chip(kindLabel[t.Kind], kindHue[t.Kind], colOn, false) + " " +
			chip("● 完成", statusHue[store.StatusDone], colOn, false))
		if len(t.Tags) > 0 {
			b.WriteString(dimStyle.Render(" #" + strings.Join(t.Tags, " #")))
		}
		b.WriteString("\n" + dimStyle.Render("完成於 "+t.CompletedAt().Format("2006-01-02 15:04")+
			"  ・  歷時 "+duration(t.Created, t.CompletedAt())) + "\n\n")

		b.WriteString(m.section("功能說明", t.Description()))
		b.WriteString(m.section("驗證方式", t.Verify))
		b.WriteString(m.section("已知限制 / 待辦", t.Limits))
		b.WriteString(m.renderFiles(t))

		if len(t.Entries) > 0 {
			b.WriteString(chip("開發過程", bgLabel, colText, true) + "\n")
			for _, e := range t.Entries {
				b.WriteString(dimStyle.Render("  "+e.Time.Format("01-02 15:04")) + "  " +
					strings.TrimLeft(indent(wrap(e.Note, m.detailWidth()-18), strings.Repeat(" ", 16)), " ") + "\n")
			}
		}

	default:
		b.WriteString(chip("功能名稱", bgLabel, colText, true) + "\n")
		for _, line := range strings.Split(wrap(t.Title, m.detailWidth()-4), "\n") {
			b.WriteString(bar("▌ "+line, m.detailWidth()-2, bgSelect, colText, true) + "\n")
		}
		b.WriteString("\n")
		b.WriteString(dimStyle.Render(t.ID) + " " +
			chip(kindLabel[t.Kind], kindHue[t.Kind], colOn, false) + " " +
			chip(statusMark[t.Status]+" "+statusLabel[t.Status], statusHue[t.Status], colOn, false))
		b.WriteString("\n" + dimStyle.Render(fmt.Sprintf("建立 %s ・ 更新 %s",
			t.Created.Format("2006-01-02 15:04"), humanize(t.Updated))) + "\n\n")

		b.WriteString(m.renderFiles(t))
		if len(t.Entries) == 0 {
			b.WriteString(dimStyle.Render("尚無進度（按 n 新增步驟）"))
		} else {
			b.WriteString(chip("進度", bgLabel, colText, true) + "\n")
			for i := len(t.Entries) - 1; i >= 0; i-- { // 最新在上
				e := t.Entries[i]
				b.WriteString(lipgloss.NewStyle().Foreground(colAccent).Render("● ") +
					dimStyle.Render(e.Time.Format("01-02 15:04")) + "\n")
				b.WriteString(indent(wrap(e.Note, m.detailWidth()-4), "  ") + "\n\n")
			}
		}
	}

	m.detail.Width = m.detailWidth()
	m.detail.Height = m.detailHeight()
	m.detail.SetContent(b.String())
	return paneStyle.Width(m.detailWidth()).Height(m.detailHeight()).Render(m.detail.View())
}

// renderFiles 列出改動檔案，新增的標 +（綠色）、修改的標 ~。
func (m Model) renderFiles(t *store.Task) string {
	files := t.AllFiles()
	if len(files) == 0 {
		return ""
	}
	added := 0
	for _, f := range files {
		if f.IsNew {
			added++
		}
	}
	label := fmt.Sprintf("改動檔案 %d", len(files))
	if added > 0 {
		label += fmt.Sprintf("（新增 %d）", added)
	}
	var b strings.Builder
	b.WriteString(chip(label, bgLabel, colText, true) + "\n")
	for _, f := range files {
		mark := dimStyle.Render("~")
		style := textStyle
		if f.IsNew {
			mark = lipgloss.NewStyle().Foreground(statusHue[store.StatusDone]).Bold(true).Render("+")
			style = lipgloss.NewStyle().Foreground(statusHue[store.StatusDone])
		}
		b.WriteString("  " + mark + " " + style.Render(truncate(f.Path, m.detailWidth()-6)) + "\n")
	}
	b.WriteString("\n")
	return b.String()
}

func (m Model) renderFooter() string {
	if m.mode != modeList {
		label := "新功能"
		if m.mode == modeAddNote {
			if t := m.current(); t != nil {
				label = t.ID + " 步驟"
			}
		}
		return " " + chip(label, colAccent, colOn, true) + " " + m.input.View() +
			dimStyle.Render("  (enter 送出 · esc 取消)")
	}
	if m.err != nil {
		return " " + chip("錯誤", statusHue[store.StatusBlocked], colOn, true) + " " +
			truncateRaw(textStyle.Render(m.err.Error()), m.width-8)
	}
	help := dimStyle.Render("tab 換頁 · j/k 移動 · J/K 捲動詳情 · a 新功能 · n 記步驟 · s 切狀態 · r 重讀 · q 離開")
	if m.status != "" {
		help = chip(m.status, statusHue[store.StatusDone], colOn, false) + " " + help
	}
	return " " + truncateRaw(help, m.width-1)
}

// ── 文字工具 ──────────────────────────────────────────

func truncate(s string, max int) string {
	if max < 4 {
		max = 4
	}
	if lipgloss.Width(s) <= max {
		return s
	}
	r := []rune(s)
	for len(r) > 0 && lipgloss.Width(string(r))+1 > max {
		r = r[:len(r)-1]
	}
	return string(r) + "…"
}

// truncateRaw 針對已上色的字串做寬度裁切（不附加省略號以免破壞 ANSI）。
func truncateRaw(s string, max int) string {
	if max <= 0 || lipgloss.Width(s) <= max {
		return s
	}
	return lipgloss.NewStyle().MaxWidth(max).Render(s)
}

func wrap(s string, width int) string {
	if width < 8 {
		width = 8
	}
	return lipgloss.NewStyle().Width(width).Render(s)
}

func indent(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		if i == 0 {
			lines[i] = prefix + l
		} else {
			lines[i] = strings.Repeat(" ", lipgloss.Width(prefix)) + l
		}
	}
	return strings.Join(lines, "\n")
}

func humanize(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "剛剛"
	case d < time.Hour:
		return fmt.Sprintf("%d 分鐘前", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d 小時前", int(d.Hours()))
	default:
		return fmt.Sprintf("%d 天前", int(d.Hours()/24))
	}
}

func humanizeShort(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "剛剛"
	case d < time.Hour:
		return fmt.Sprintf("%d 分前", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d 時前", int(d.Hours()))
	default:
		return t.Format("01-02")
	}
}

// duration 描述兩個時間之間的長度。
func duration(from, to time.Time) string {
	d := to.Sub(from)
	switch {
	case d < time.Minute:
		return "不到 1 分鐘"
	case d < time.Hour:
		return fmt.Sprintf("%d 分鐘", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%.1f 小時", d.Hours())
	default:
		return fmt.Sprintf("%d 天", int(d.Hours()/24))
	}
}
