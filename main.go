// cairn 是給 AI 輔助開發用的終端專案紀錄檔：AI 用子命令寫進度，人用 TUI 看完成紀錄。
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"cairn/internal/hooks"
	"cairn/internal/store"
	"cairn/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

const usage = `cairn — 終端專案開發紀錄

用法：
  cairn                          開啟 TUI（完成 / 進行中 / 規格書 / 技能 四個頁籤）
  cairn dev [claude|codex]       用 tmux 左右分割：左邊 AI、右邊紀錄頁
  cairn init [專案名稱] [--hook]  在目前目錄建立 .cairn/log.json，加 --hook 順便裝好 Stop hook
  cairn hook install             安裝／更新 Claude Code 的 Stop hook 與 CLAUDE.md 進度紀錄段落

  cairn add <標題> [--kind feature|fix|refactor|docs] [--tags a,b]
                                開始一項功能，印出任務 ID
  cairn log <ID> <說明> [--files 改過的檔案] [--new 新增的檔案]
                                記錄一個開發步驟（狀態自動轉 in_progress）
  cairn done <ID> --summary <功能說明> --verify <驗證方式>
                [--limits <已知限制>] [--kind <類型>]
                [--files 改過的檔案] [--new 新增的檔案]
                                完成一項功能，寫入完整紀錄
  cairn status <ID> <狀態>        設定狀態：todo|in_progress|blocked|done

  cairn spec <子命令>             規格書：把跟人談好的需求寫下來
                                add / list / show / set / status / build
                                詳見 cairn spec help

  cairn list [--status <狀態>] [--json]
                                列出任務（純文字，適合 AI 讀取）
  cairn show <ID>                顯示單一功能的完整紀錄
  cairn path                     印出目前使用的紀錄檔路徑

紀錄檔位置：由目前目錄往上層尋找 .cairn/log.json，找不到則用 ./.cairn/log.json。
`

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		if err := runTUI(); err != nil {
			fail(err)
		}
		return
	}

	cmd, rest := args[0], args[1:]
	var err error
	switch cmd {
	case "init":
		err = cmdInit(rest)
	case "hook":
		err = cmdHook(rest)
	case "dev":
		err = cmdDev(rest)
	case "add":
		err = cmdAdd(rest)
	case "log":
		err = cmdLog(rest)
	case "done":
		err = cmdDone(rest)
	case "status":
		err = cmdStatus(rest)
	case "spec":
		err = cmdSpec(rest)
	case "list", "ls":
		err = cmdList(rest)
	case "show":
		err = cmdShow(rest)
	case "path":
		fmt.Println(logPath())
	case "help", "-h", "--help":
		fmt.Print(usage)
	default:
		err = fmt.Errorf("未知命令 %q\n\n%s", cmd, usage)
	}
	if err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "cairn: "+err.Error())
	os.Exit(1)
}

func logPath() string {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	return store.Discover(cwd)
}

// load 讀取紀錄檔並回傳路徑，供各子命令共用。
func load() (string, *store.Log, error) {
	p := logPath()
	l, err := store.Load(p)
	return p, l, err
}

func runTUI() error {
	p, l, err := load()
	if err != nil {
		return err
	}
	_, err = tea.NewProgram(ui.New(p, l), tea.WithAltScreen()).Run()
	return err
}

func cmdInit(args []string) error {
	args, withHook := hasFlag(args, "hook")
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	p := filepath.Join(cwd, ".cairn", "log.json")
	if _, err := os.Stat(p); err == nil {
		return fmt.Errorf("%s 已存在", p)
	}
	name := filepath.Base(cwd)
	if len(args) > 0 {
		name = strings.Join(args, " ")
	}
	if err := store.Save(p, &store.Log{Version: 1, Project: name, Tasks: []*store.Task{}}); err != nil {
		return err
	}
	fmt.Printf("已建立 %s（專案：%s）\n", p, name)
	if withHook {
		return installHook(cwd)
	}
	return nil
}

// cmdHook 是 `cairn hook` 的子命令分派。
func cmdHook(args []string) error {
	if len(args) == 0 || args[0] != "install" {
		return fmt.Errorf("用法：cairn hook install")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	return installHook(cwd)
}

// installHook 安裝 Stop hook 與 CLAUDE.md 進度紀錄段落，並印出結果。
func installHook(cwd string) error {
	msgs, err := hooks.Install(cwd)
	if err != nil {
		return err
	}
	for _, m := range msgs {
		fmt.Println(m)
	}
	return nil
}

// cmdDev 用 tmux 開出「左 AI、右紀錄頁」的分割畫面。
func cmdDev(args []string) error {
	ai := "claude"
	if len(args) > 0 && args[0] != "" {
		ai = args[0]
	}
	if _, err := exec.LookPath(ai); err != nil {
		return fmt.Errorf("找不到指令 %q，請確認它在 PATH 中", ai)
	}
	tmuxPath, err := exec.LookPath("tmux")
	if err != nil {
		return fmt.Errorf("需要 tmux：brew install tmux")
	}
	self, err := os.Executable()
	if err != nil {
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	var argv []string
	if os.Getenv("TMUX") != "" {
		// 已經在 tmux 裡：直接把目前視窗切一半給紀錄頁
		argv = []string{"tmux", "split-window", "-h", "-l", "42%", "-c", cwd, self}
	} else {
		session := "cairn-" + filepath.Base(cwd)
		argv = []string{"tmux", "new-session", "-A", "-s", session, "-c", cwd, ai,
			";", "set", "mouse", "on", // 可以直接用滑鼠點選 pane
			";", "split-window", "-h", "-l", "42%", "-c", cwd, self,
			";", "select-pane", "-L"}
	}
	return syscall.Exec(tmuxPath, argv, os.Environ())
}

// popFlag 從參數中取出 --name=v 或 --name v，回傳剩餘參數。
func popFlag(args []string, name string) ([]string, string) {
	for i, a := range args {
		switch {
		case a == "--"+name && i+1 < len(args):
			return append(args[:i:i], args[i+2:]...), args[i+1]
		case strings.HasPrefix(a, "--"+name+"="):
			return append(args[:i:i], args[i+1:]...), strings.TrimPrefix(a, "--"+name+"=")
		}
	}
	return args, ""
}

func hasFlag(args []string, name string) ([]string, bool) {
	for i, a := range args {
		if a == "--"+name {
			return append(args[:i:i], args[i+1:]...), true
		}
	}
	return args, false
}

func splitList(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func cmdAdd(args []string) error {
	args, tags := popFlag(args, "tags")
	args, kindArg := popFlag(args, "kind")
	kind := store.KindFeature
	if kindArg != "" {
		var err error
		if kind, err = store.NormalizeKind(kindArg); err != nil {
			return err
		}
	}
	title := strings.TrimSpace(strings.Join(args, " "))
	if title == "" {
		return fmt.Errorf("用法：cairn add <標題> [--kind feature|fix|refactor|docs] [--tags a,b]")
	}
	p, l, err := load()
	if err != nil {
		return err
	}
	t := l.AddTask(title, kind, splitList(tags))
	if err := store.Save(p, l); err != nil {
		return err
	}
	fmt.Println(t.ID)
	return nil
}

func cmdLog(args []string) error {
	args, files := popFlag(args, "files")
	args, created := popFlag(args, "new")
	if len(args) < 2 {
		return fmt.Errorf("用法：cairn log <ID> <說明> [--files 改過的檔案] [--new 新增的檔案]")
	}
	p, l, err := load()
	if err != nil {
		return err
	}
	t := l.Find(args[0])
	if t == nil {
		return fmt.Errorf("找不到任務 %s", args[0])
	}
	t.AddEntry(strings.Join(args[1:], " "), splitList(files), splitList(created))
	if t.Status == store.StatusTodo {
		t.SetStatus(store.StatusInProgress)
	}
	if err := store.Save(p, l); err != nil {
		return err
	}
	fmt.Printf("%s 已紀錄（目前 %s，共 %d 個步驟）\n", t.ID, t.Status, len(t.Entries))
	return nil
}

func cmdDone(args []string) error {
	args, summary := popFlag(args, "summary")
	args, verify := popFlag(args, "verify")
	args, limits := popFlag(args, "limits")
	args, kindArg := popFlag(args, "kind")
	args, files := popFlag(args, "files")
	args, created := popFlag(args, "new")

	if len(args) < 1 {
		return fmt.Errorf("用法：cairn done <ID> --summary <功能說明> --verify <驗證方式> [--limits <已知限制>] [--kind <類型>] [--files 改過的檔案] [--new 新增的檔案]")
	}
	if strings.TrimSpace(summary) == "" || strings.TrimSpace(verify) == "" {
		return fmt.Errorf("--summary（功能說明）與 --verify（驗證方式）都是必填，完成紀錄要讓人看得懂做了什麼、怎麼確認可用")
	}
	p, l, err := load()
	if err != nil {
		return err
	}
	t := l.Find(args[0])
	if t == nil {
		return fmt.Errorf("找不到任務 %s", args[0])
	}
	if kindArg != "" {
		k, err := store.NormalizeKind(kindArg)
		if err != nil {
			return err
		}
		t.Kind = k
	}
	if f, c := splitList(files), splitList(created); len(f) > 0 || len(c) > 0 {
		t.AddEntry("完成", f, c)
	}
	t.Complete(summary, verify, limits)
	// 這項任務是從規格書開出來的話，規格書也一起結掉。
	spec := l.SpecForTask(t.ID)
	if spec != nil {
		spec.SetSpecStatus(store.SpecDone)
	}
	if err := store.Save(p, l); err != nil {
		return err
	}
	fmt.Printf("%s 已完成：%s（%d 個檔案）\n", t.ID, t.Title, len(t.AllFiles()))
	if spec != nil {
		fmt.Printf("規格書 %s → 已完成\n", spec.ID)
	}
	return nil
}

func cmdStatus(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("用法：cairn status <ID> <todo|in_progress|blocked|done>")
	}
	s, err := store.NormalizeStatus(args[1])
	if err != nil {
		return err
	}
	if s == store.StatusDone {
		return fmt.Errorf("請改用 cairn done <ID> --summary <功能說明> --verify <驗證方式>，完成紀錄需要這些內容")
	}
	p, l, err := load()
	if err != nil {
		return err
	}
	t := l.Find(args[0])
	if t == nil {
		return fmt.Errorf("找不到任務 %s", args[0])
	}
	t.SetStatus(s)
	if err := store.Save(p, l); err != nil {
		return err
	}
	fmt.Printf("%s → %s\n", t.ID, s)
	return nil
}

func cmdList(args []string) error {
	args, asJSON := hasFlag(args, "json")
	args, filter := popFlag(args, "status")
	_ = args
	_, l, err := load()
	if err != nil {
		return err
	}
	if filter != "" {
		if filter, err = store.NormalizeStatus(filter); err != nil {
			return err
		}
	}
	l.SortForDisplay()

	out := make([]*store.Task, 0, len(l.Tasks))
	for _, t := range l.Tasks {
		if filter == "" || t.Status == filter {
			out = append(out, t)
		}
	}
	if asJSON {
		b, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(b))
		return nil
	}
	if len(out) == 0 {
		fmt.Println("（沒有任務）")
		return nil
	}
	for _, t := range out {
		fmt.Printf("%s  %-11s  %-8s  %s\n", t.ID, t.Status, t.Kind, t.Title)
	}
	return nil
}

func cmdShow(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("用法：cairn show <ID>")
	}
	_, l, err := load()
	if err != nil {
		return err
	}
	t := l.Find(args[0])
	if t == nil {
		return fmt.Errorf("找不到任務 %s", args[0])
	}
	fmt.Printf("%s  %s  [%s / %s]\n", t.ID, t.Title, t.Kind, t.Status)
	if s := l.SpecForTask(t.ID); s != nil {
		fmt.Printf("依據規格書：%s（cairn spec show %s）\n", s.ID, s.ID)
	}
	if len(t.Tags) > 0 {
		fmt.Printf("標籤：%s\n", strings.Join(t.Tags, ", "))
	}
	fmt.Printf("建立：%s", t.Created.Format("2006-01-02 15:04"))
	if t.Status == store.StatusDone {
		fmt.Printf("   完成：%s", t.CompletedAt().Format("2006-01-02 15:04"))
	}
	fmt.Println()

	printField := func(label, body string) {
		if strings.TrimSpace(body) != "" {
			fmt.Printf("\n%s：\n  %s\n", label, body)
		}
	}
	printField("功能說明", t.Description())
	printField("驗證方式", t.Verify)
	printField("已知限制", t.Limits)

	if files := t.AllFiles(); len(files) > 0 {
		fmt.Printf("\n改動檔案（%d）：\n", len(files))
		for _, f := range files {
			mark := "~"
			if f.IsNew {
				mark = "+"
			}
			fmt.Printf("  %s %s\n", mark, f.Path)
		}
	}
	if len(t.Entries) > 0 {
		fmt.Println("\n開發過程：")
		for _, e := range t.Entries {
			fmt.Printf("  %s  %s\n", e.Time.Format("2006-01-02 15:04"), e.Note)
		}
	}
	return nil
}
