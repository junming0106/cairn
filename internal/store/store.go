// Package store 負責 .cairn/log.json 的讀寫與資料模型。
package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// 任務狀態。
const (
	StatusTodo       = "todo"
	StatusInProgress = "in_progress"
	StatusBlocked    = "blocked"
	StatusDone       = "done"
)

// Statuses 是狀態的顯示順序，也是 TUI 中 s 鍵循環的順序。
var Statuses = []string{StatusTodo, StatusInProgress, StatusBlocked, StatusDone}

var statusAliases = map[string]string{
	"todo": StatusTodo, "t": StatusTodo, "待辦": StatusTodo,
	"in_progress": StatusInProgress, "wip": StatusInProgress, "doing": StatusInProgress,
	"progress": StatusInProgress, "p": StatusInProgress, "進行中": StatusInProgress,
	"blocked": StatusBlocked, "block": StatusBlocked, "b": StatusBlocked, "卡住": StatusBlocked,
	"done": StatusDone, "d": StatusDone, "完成": StatusDone,
}

// NormalizeStatus 把使用者/AI 給的簡寫轉成標準狀態。
func NormalizeStatus(s string) (string, error) {
	if v, ok := statusAliases[strings.ToLower(strings.TrimSpace(s))]; ok {
		return v, nil
	}
	return "", fmt.Errorf("未知狀態 %q（可用：todo, in_progress, blocked, done）", s)
}

// 功能類型。
const (
	KindFeature  = "feature"
	KindFix      = "fix"
	KindRefactor = "refactor"
	KindDocs     = "docs"
)

var kindAliases = map[string]string{
	"feature": KindFeature, "feat": KindFeature, "f": KindFeature, "功能": KindFeature,
	"fix": KindFix, "bug": KindFix, "bugfix": KindFix, "修正": KindFix,
	"refactor": KindRefactor, "ref": KindRefactor, "chore": KindRefactor, "重構": KindRefactor,
	"docs": KindDocs, "doc": KindDocs, "文件": KindDocs,
}

// NormalizeKind 把類型簡寫轉成標準值。
func NormalizeKind(s string) (string, error) {
	if v, ok := kindAliases[strings.ToLower(strings.TrimSpace(s))]; ok {
		return v, nil
	}
	return "", fmt.Errorf("未知類型 %q（可用：feature, fix, refactor, docs）", s)
}

// Entry 是開發過程中的一個步驟。Files 是修改過的檔案，New 是這一步新建立的檔案。
type Entry struct {
	Time  time.Time `json:"time"`
	Note  string    `json:"note"`
	Files []string  `json:"files,omitempty"`
	New   []string  `json:"new,omitempty"`
}

// Task 是一項開發功能。完成後會出現在 TUI 的「完成紀錄」頁。
type Task struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`   // 簡單扼要的功能名稱
	Kind    string   `json:"kind"`    // feature / fix / refactor / docs
	Status  string   `json:"status"`  //
	Summary string   `json:"summary"` // 功能說明：實作了哪些內容
	Verify  string   `json:"verify"`  // 驗證方式：怎麼確認可用
	Limits  string   `json:"limits"`  // 已知限制 / 待辦
	Tags    []string `json:"tags,omitempty"`

	Created   time.Time  `json:"created"`
	Updated   time.Time  `json:"updated"`
	Completed *time.Time `json:"completed,omitempty"`

	Entries []Entry `json:"entries"`
}

// Log 是整份紀錄檔。
type Log struct {
	Version int     `json:"version"`
	Project string  `json:"project,omitempty"`
	Tasks   []*Task `json:"tasks"`
}

const dirName = ".cairn"
const fileName = "log.json"

// Discover 從 start 往上層找既有的 .cairn/log.json；找不到時回傳 start 下的預設路徑。
func Discover(start string) string {
	dir, err := filepath.Abs(start)
	if err != nil {
		dir = start
	}
	for {
		p := filepath.Join(dir, dirName, fileName)
		if _, err := os.Stat(p); err == nil {
			return p
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return filepath.Join(start, dirName, fileName)
}

// Load 讀取紀錄檔；檔案不存在時回傳空的 Log（而非錯誤）。
func Load(path string) (*Log, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Log{Version: 1, Tasks: []*Task{}}, nil
	}
	if err != nil {
		return nil, err
	}
	var l Log
	if err := json.Unmarshal(b, &l); err != nil {
		return nil, fmt.Errorf("%s 格式錯誤：%w", path, err)
	}
	if l.Tasks == nil {
		l.Tasks = []*Task{}
	}
	if l.Version == 0 {
		l.Version = 1
	}
	for _, t := range l.Tasks {
		if t.Kind == "" {
			t.Kind = KindFeature // 舊紀錄沒有類型欄位
		}
	}
	return &l, nil
}

// Save 以「寫暫存檔再 rename」的方式原子性寫入，避免 TUI 讀到半截 JSON。
func Save(path string, l *Log) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(path), ".log-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o644); err != nil { // CreateTemp 預設 0600，會擋掉別的工具讀取
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// ModTime 回傳紀錄檔的修改時間，供 TUI 偵測外部（AI）寫入。
func ModTime(path string) time.Time {
	fi, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return fi.ModTime()
}

// Find 依 ID 尋找任務，ID 大小寫不敏感，也接受純數字（3 → T-003）。
func (l *Log) Find(id string) *Task {
	want := strings.ToUpper(strings.TrimSpace(id))
	if n, err := strconv.Atoi(want); err == nil {
		want = fmt.Sprintf("T-%03d", n)
	}
	for _, t := range l.Tasks {
		if strings.ToUpper(t.ID) == want {
			return t
		}
	}
	return nil
}

// NextID 產生下一個未使用的任務 ID。
func (l *Log) NextID() string {
	max := 0
	for _, t := range l.Tasks {
		var n int
		if _, err := fmt.Sscanf(strings.ToUpper(t.ID), "T-%d", &n); err == nil && n > max {
			max = n
		}
	}
	return fmt.Sprintf("T-%03d", max+1)
}

// AddTask 新增任務並回傳它。
func (l *Log) AddTask(title, kind string, tags []string) *Task {
	now := time.Now()
	if kind == "" {
		kind = KindFeature
	}
	t := &Task{
		ID:      l.NextID(),
		Title:   strings.TrimSpace(title),
		Kind:    kind,
		Status:  StatusTodo,
		Tags:    tags,
		Created: now,
		Updated: now,
		Entries: []Entry{},
	}
	l.Tasks = append(l.Tasks, t)
	return t
}

// AddEntry 為任務追加一個開發步驟。files 是修改的檔案，created 是新增的檔案。
func (t *Task) AddEntry(note string, files, created []string) {
	now := time.Now()
	t.Entries = append(t.Entries, Entry{
		Time: now, Note: strings.TrimSpace(note), Files: files, New: created,
	})
	t.Updated = now
}

// Complete 把任務標記為完成，並填上完成紀錄需要的欄位。
func (t *Task) Complete(summary, verify, limits string) {
	now := time.Now()
	t.Summary = strings.TrimSpace(summary)
	t.Verify = strings.TrimSpace(verify)
	if limits != "" {
		t.Limits = strings.TrimSpace(limits)
	}
	t.Status = StatusDone
	t.Updated = now
	t.Completed = &now
}

// SetStatus 更新狀態。從完成改回未完成時會清掉完成時間。
func (t *Task) SetStatus(s string) {
	t.Status = s
	t.Updated = time.Now()
	if s == StatusDone {
		if t.Completed == nil {
			now := time.Now()
			t.Completed = &now
		}
	} else {
		t.Completed = nil
	}
}

// NextStatus 回傳循環中的下一個狀態。
func NextStatus(s string) string {
	for i, v := range Statuses {
		if v == s {
			return Statuses[(i+1)%len(Statuses)]
		}
	}
	return StatusTodo
}

// CompletedAt 回傳完成時間；舊紀錄沒有這個欄位時退回最後更新時間。
func (t *Task) CompletedAt() time.Time {
	if t.Completed != nil {
		return *t.Completed
	}
	return t.Updated
}

// Description 回傳功能說明；沒填 summary 的舊紀錄退回最後一筆步驟說明。
func (t *Task) Description() string {
	if t.Summary != "" {
		return t.Summary
	}
	if n := len(t.Entries); n > 0 {
		return t.Entries[n-1].Note
	}
	return ""
}

// FileChange 是一個檔案的變更；IsNew 表示這個檔案是這次新建立的。
type FileChange struct {
	Path  string
	IsNew bool
}

// AllFiles 回傳這項功能碰過的所有檔案（新增在前、去重、排序）。
func (t *Task) AllFiles() []FileChange {
	isNew := map[string]bool{}
	seen := map[string]bool{}
	paths := []string{}
	add := func(f string, created bool) {
		if created {
			isNew[f] = true
		}
		if !seen[f] {
			seen[f] = true
			paths = append(paths, f)
		}
	}
	for _, e := range t.Entries {
		for _, f := range e.New {
			add(f, true)
		}
		for _, f := range e.Files {
			add(f, false)
		}
	}
	sort.Strings(paths)
	out := make([]FileChange, 0, len(paths))
	for _, p := range paths { // 新增的排前面
		if isNew[p] {
			out = append(out, FileChange{Path: p, IsNew: true})
		}
	}
	for _, p := range paths {
		if !isNew[p] {
			out = append(out, FileChange{Path: p})
		}
	}
	return out
}

// Done 回傳已完成的任務，最近完成的在前。
func (l *Log) Done() []*Task {
	out := []*Task{}
	for _, t := range l.Tasks {
		if t.Status == StatusDone {
			out = append(out, t)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].CompletedAt().After(out[j].CompletedAt())
	})
	return out
}

// Active 回傳未完成的任務：進行中 → 卡住 → 待辦，同組內最近更新在前。
func (l *Log) Active() []*Task {
	rank := map[string]int{StatusInProgress: 0, StatusBlocked: 1, StatusTodo: 2}
	out := []*Task{}
	for _, t := range l.Tasks {
		if t.Status != StatusDone {
			out = append(out, t)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if rank[a.Status] != rank[b.Status] {
			return rank[a.Status] < rank[b.Status]
		}
		return a.Updated.After(b.Updated)
	})
	return out
}

// SortForDisplay 依「未完成優先、最近更新在前」排序，供 cairn list 使用。
func (l *Log) SortForDisplay() {
	l.Tasks = append(l.Active(), l.Done()...)
}

// Counts 回傳各狀態的任務數。
func (l *Log) Counts() map[string]int {
	c := map[string]int{}
	for _, t := range l.Tasks {
		c[t.Status]++
	}
	return c
}
