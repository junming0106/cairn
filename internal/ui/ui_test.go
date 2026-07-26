package ui

import (
	"path/filepath"
	"strings"
	"testing"

	"cairn/internal/store"

	tea "github.com/charmbracelet/bubbletea"
)

// newTestModel 建一個有畫面尺寸的 Model，紀錄檔寫在暫存目錄。
func newTestModel(t *testing.T, l *store.Log) Model {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".cairn", "log.json")
	if err := store.Save(path, l); err != nil {
		t.Fatal(err)
	}
	m, _ := New(path, l).Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	return m.(Model)
}

func press(t *testing.T, m Model, keys ...string) Model {
	t.Helper()
	for _, k := range keys {
		var msg tea.KeyMsg
		switch k {
		case "enter":
			msg = tea.KeyMsg{Type: tea.KeyEnter}
		default:
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
		}
		next, _ := m.Update(msg)
		m = next.(Model)
	}
	return m
}

// 規格書頁只負責顯示：看得到 AI 寫進來的正文，s 切換待完成 / 已完成。
func TestSpecTabDisplay(t *testing.T) {
	l := &store.Log{Version: 1, Project: "測試", Tasks: []*store.Task{}, Specs: []*store.Spec{}}
	s := l.AddSpec("文章分類", "## 目標\n每篇文章可以掛一個分類。")
	m := newTestModel(t, l)

	m = press(t, m, "3")
	if m.tab != tabSpecs {
		t.Fatalf("按 3 應該切到規格書頁，實際 tab=%d", m.tab)
	}
	if s.Status != store.SpecTodo {
		t.Errorf("新規格書應該是待完成，實際 %s", s.Status)
	}
	v := m.View()
	for _, want := range []string{"文章分類", "每篇文章可以掛一個分類", "待完成"} {
		if !strings.Contains(v, want) {
			t.Errorf("規格書頁應該顯示 %q", want)
		}
	}

	// s 切換標籤
	m = press(t, m, "s")
	if s.Status != store.SpecDone {
		t.Fatalf("按 s 應該變已完成，實際 %s", s.Status)
	}
	if !strings.Contains(m.View(), "已完成") {
		t.Error("切換後標籤要顯示已完成")
	}
	m = press(t, m, "s")
	if s.Status != store.SpecTodo {
		t.Fatalf("再按 s 應該回到待完成，實際 %s", s.Status)
	}
}

// 規格書頁不該有新增與發言的輸入框，內容一律由 AI 從命令列寫入。
func TestSpecTabIsReadOnly(t *testing.T) {
	l := &store.Log{Version: 1, Tasks: []*store.Task{}, Specs: []*store.Spec{}}
	l.AddSpec("既有規格", "正文")
	m := newTestModel(t, l)

	m = press(t, m, "3", "a")
	if m.mode != modeList {
		t.Error("規格書頁按 a 不該開輸入框")
	}
	m = press(t, m, "n")
	if m.mode != modeList {
		t.Error("規格書頁按 n 不該開輸入框")
	}
	if len(m.log.Specs) != 1 {
		t.Errorf("按鍵不該新增規格書，實際 %d 份", len(m.log.Specs))
	}
}

// 任務頁的 n 仍然是加備註，沒有被規格書搶走。
func TestRemarkStillWorksOnTaskTab(t *testing.T) {
	l := &store.Log{Version: 1, Tasks: []*store.Task{}, Specs: []*store.Spec{}}
	task := l.AddTask("既有任務", store.KindFeature, nil)
	m := newTestModel(t, l)

	m = press(t, m, "2", "n")
	if m.mode != modeAddRemark {
		t.Fatal("進行中頁按 n 應該是加備註")
	}
	m = press(t, m, "備註內容", "enter")
	if len(task.Remarks) != 1 {
		t.Fatalf("備註沒寫進去：%+v", task.Remarks)
	}
}

// 規格書換狀態會改變排序，游標要跟著同一份規格書走。
func TestCursorFollowsSpecAcrossReorder(t *testing.T) {
	l := &store.Log{Version: 1, Tasks: []*store.Task{}, Specs: []*store.Spec{}}
	l.AddSpec("第一份", "正文一")
	l.AddSpec("第二份", "正文二")
	m := newTestModel(t, l)

	m = press(t, m, "3", "j") // 選到清單第二列
	before := m.currentSpec()
	if before == nil {
		t.Fatal("應該選到一份規格書")
	}
	m = press(t, m, "s")
	if got := m.currentSpec(); got == nil || got.ID != before.ID {
		t.Errorf("切狀態後游標應該還在 %s，實際 %v", before.ID, got)
	}
}

// 技能頁是最後一個頁籤，索引不能被規格書頁擠掉。
func TestSkillsTabStillLast(t *testing.T) {
	m := newTestModel(t, &store.Log{Version: 1, Tasks: []*store.Task{}, Specs: []*store.Spec{}})
	m = press(t, m, "4")
	if m.tab != tabSkills {
		t.Fatalf("按 4 應該是技能頁，實際 tab=%d", m.tab)
	}
	if len(tabNames) != 4 {
		t.Fatalf("頁籤數量應該是 4，實際 %d", len(tabNames))
	}
}
