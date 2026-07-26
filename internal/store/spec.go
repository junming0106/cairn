package store

// 規格書：AI 在終端跟人談完要做什麼之後，把結論寫成一份規格書。
// TUI 的「規格書」頁只負責顯示，讓人隨時看得到談好的內容與完成狀態。

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// 規格書狀態：只有待完成與已完成兩種。
const (
	SpecTodo = "todo"
	SpecDone = "done"
)

var specStatusAliases = map[string]string{
	"todo": SpecTodo, "t": SpecTodo, "pending": SpecTodo, "待完成": SpecTodo, "未完成": SpecTodo,
	"done": SpecDone, "d": SpecDone, "已完成": SpecDone, "完成": SpecDone,
}

// NormalizeSpecStatus 把簡寫轉成標準的規格書狀態。
func NormalizeSpecStatus(s string) (string, error) {
	if v, ok := specStatusAliases[strings.ToLower(strings.TrimSpace(s))]; ok {
		return v, nil
	}
	return "", fmt.Errorf("未知狀態 %q（可用：todo, done）", s)
}

// Spec 是一份規格書：Body 是談好的規格正文，TaskID 是對應的開發任務。
type Spec struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
	Body   string `json:"body"`
	TaskID string `json:"task_id,omitempty"`

	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`
}

// FindSpec 依 ID 尋找規格書，大小寫不敏感，也接受純數字（3 → S-003）。
func (l *Log) FindSpec(id string) *Spec {
	want := strings.ToUpper(strings.TrimSpace(id))
	if n, err := strconv.Atoi(want); err == nil {
		want = fmt.Sprintf("S-%03d", n)
	}
	for _, s := range l.Specs {
		if strings.ToUpper(s.ID) == want {
			return s
		}
	}
	return nil
}

// NextSpecID 產生下一個未使用的規格書 ID。
func (l *Log) NextSpecID() string {
	max := 0
	for _, s := range l.Specs {
		var n int
		if _, err := fmt.Sscanf(strings.ToUpper(s.ID), "S-%d", &n); err == nil && n > max {
			max = n
		}
	}
	return fmt.Sprintf("S-%03d", max+1)
}

// AddSpec 新增一份待完成的規格書。
func (l *Log) AddSpec(title, body string) *Spec {
	now := time.Now()
	s := &Spec{
		ID:      l.NextSpecID(),
		Title:   strings.TrimSpace(title),
		Status:  SpecTodo,
		Body:    strings.TrimSpace(body),
		Created: now,
		Updated: now,
	}
	l.Specs = append(l.Specs, s)
	return s
}

// SetBody 改寫規格正文。
func (s *Spec) SetBody(body string) {
	s.Body = strings.TrimSpace(body)
	s.Updated = time.Now()
}

// SetSpecStatus 更新規格書狀態。
func (s *Spec) SetSpecStatus(status string) {
	s.Status = status
	s.Updated = time.Now()
}

// NextSpecStatus 在待完成與已完成之間切換。
func NextSpecStatus(s string) string {
	if s == SpecDone {
		return SpecTodo
	}
	return SpecDone
}

// SpecList 回傳規格書：待完成的在前，同組內最近更新在前。
func (l *Log) SpecList() []*Spec {
	out := append([]*Spec{}, l.Specs...)
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if (a.Status == SpecDone) != (b.Status == SpecDone) {
			return b.Status == SpecDone
		}
		return a.Updated.After(b.Updated)
	})
	return out
}

// SpecForTask 回傳這個任務是從哪一份規格書開出來的。
func (l *Log) SpecForTask(taskID string) *Spec {
	for _, s := range l.Specs {
		if s.TaskID != "" && strings.EqualFold(s.TaskID, taskID) {
			return s
		}
	}
	return nil
}

// SpecCounts 回傳待完成與已完成的規格書數量。
func (l *Log) SpecCounts() (todo, done int) {
	for _, s := range l.Specs {
		if s.Status == SpecDone {
			done++
		} else {
			todo++
		}
	}
	return
}
