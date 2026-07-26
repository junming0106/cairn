package main

// cairn spec — 規格書子命令。人跟 AI 在終端談要做什麼，AI 談完把結論寫成規格書；
// TUI 的規格書頁只負責顯示，讓人隨時看得到談好的內容與完成狀態。

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"cairn/internal/store"
)

const specUsage = `cairn spec — 規格書：把跟人談好的需求寫下來，做完再標成完成

  cairn spec add <標題> --body <正文> | --body-file <檔案>
                                跟人談定一項功能後寫成規格書，印出規格 ID
  cairn spec list [--status todo|done] [--json]
                                列出規格書
  cairn spec show <ID> [--json]  顯示規格正文
  cairn spec set <ID> --body <正文> | --body-file <檔案>
                                需求改了就整份改寫正文
  cairn spec status <ID> <todo|done>
                                切換待完成 / 已完成
  cairn spec build <ID> [--kind feature|fix|refactor|docs]
                                依規格書開出開發任務，印出任務 ID

規格 ID 可用 S-001 或 1。任務 cairn done 時，對應的規格書會自動標成已完成。
`

func cmdSpec(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("用法：\n\n%s", specUsage)
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "add":
		return cmdSpecAdd(rest)
	case "list", "ls":
		return cmdSpecList(rest)
	case "show":
		return cmdSpecShow(rest)
	case "set":
		return cmdSpecSet(rest)
	case "status":
		return cmdSpecStatus(rest)
	case "build":
		return cmdSpecBuild(rest)
	case "help", "-h", "--help":
		fmt.Print(specUsage)
		return nil
	default:
		return fmt.Errorf("未知的 spec 命令 %q\n\n%s", sub, specUsage)
	}
}

// findSpec 讀取紀錄檔並取出指定規格書。
func findSpec(id string) (string, *store.Log, *store.Spec, error) {
	p, l, err := load()
	if err != nil {
		return "", nil, nil, err
	}
	s := l.FindSpec(id)
	if s == nil {
		return "", nil, nil, fmt.Errorf("找不到規格書 %s", id)
	}
	return p, l, s, nil
}

// readBody 取出 --body 或 --body-file 的正文。
func readBody(body, bodyFile string) (string, error) {
	if bodyFile != "" {
		b, err := os.ReadFile(bodyFile)
		if err != nil {
			return "", err
		}
		body = string(b)
	}
	if strings.TrimSpace(body) == "" {
		return "", fmt.Errorf("規格正文不能是空的：要寫清楚做什麼、怎麼算完成（長文用 --body-file <檔案>）")
	}
	return body, nil
}

func cmdSpecAdd(args []string) error {
	args, body := popFlag(args, "body")
	args, bodyFile := popFlag(args, "body-file")
	title := strings.TrimSpace(strings.Join(args, " "))
	if title == "" {
		return fmt.Errorf("用法：cairn spec add <標題> --body <正文> | --body-file <檔案>")
	}
	body, err := readBody(body, bodyFile)
	if err != nil {
		return err
	}
	p, l, err := load()
	if err != nil {
		return err
	}
	s := l.AddSpec(title, body)
	if err := store.Save(p, l); err != nil {
		return err
	}
	fmt.Println(s.ID)
	return nil
}

func cmdSpecList(args []string) error {
	args, asJSON := hasFlag(args, "json")
	args, filter := popFlag(args, "status")
	_ = args
	_, l, err := load()
	if err != nil {
		return err
	}
	if filter != "" {
		if filter, err = store.NormalizeSpecStatus(filter); err != nil {
			return err
		}
	}
	out := []*store.Spec{}
	for _, s := range l.SpecList() {
		if filter == "" || s.Status == filter {
			out = append(out, s)
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
		fmt.Println("（沒有規格書）")
		return nil
	}
	for _, s := range out {
		label := "待完成"
		if s.Status == store.SpecDone {
			label = "已完成"
		}
		line := fmt.Sprintf("%s  %s  %s", s.ID, label, s.Title)
		if s.TaskID != "" {
			line += "  → " + s.TaskID
		}
		fmt.Println(line)
	}
	return nil
}

func cmdSpecShow(args []string) error {
	args, asJSON := hasFlag(args, "json")
	if len(args) < 1 {
		return fmt.Errorf("用法：cairn spec show <ID> [--json]")
	}
	_, _, s, err := findSpec(args[0])
	if err != nil {
		return err
	}
	if asJSON {
		b, err := json.MarshalIndent(s, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(b))
		return nil
	}
	label := "待完成"
	if s.Status == store.SpecDone {
		label = "已完成"
	}
	fmt.Printf("%s  %s  [%s]\n", s.ID, s.Title, label)
	if s.TaskID != "" {
		fmt.Printf("開發任務：%s\n", s.TaskID)
	}
	fmt.Printf("建立：%s   更新：%s\n",
		s.Created.Format("2006-01-02 15:04"), s.Updated.Format("2006-01-02 15:04"))
	fmt.Printf("\n規格正文：\n%s\n", indentLines(s.Body, "  "))
	return nil
}

func cmdSpecSet(args []string) error {
	args, body := popFlag(args, "body")
	args, bodyFile := popFlag(args, "body-file")
	if len(args) < 1 {
		return fmt.Errorf("用法：cairn spec set <ID> --body <正文> | --body-file <檔案>")
	}
	body, err := readBody(body, bodyFile)
	if err != nil {
		return err
	}
	p, l, s, err := findSpec(args[0])
	if err != nil {
		return err
	}
	s.SetBody(body)
	if err := store.Save(p, l); err != nil {
		return err
	}
	fmt.Printf("%s 規格正文已更新\n", s.ID)
	return nil
}

func cmdSpecStatus(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("用法：cairn spec status <ID> <todo|done>")
	}
	st, err := store.NormalizeSpecStatus(args[1])
	if err != nil {
		return err
	}
	p, l, s, err := findSpec(args[0])
	if err != nil {
		return err
	}
	s.SetSpecStatus(st)
	if err := store.Save(p, l); err != nil {
		return err
	}
	fmt.Printf("%s → %s\n", s.ID, st)
	return nil
}

func cmdSpecBuild(args []string) error {
	args, kindArg := popFlag(args, "kind")
	if len(args) < 1 {
		return fmt.Errorf("用法：cairn spec build <ID> [--kind feature|fix|refactor|docs]")
	}
	kind := store.KindFeature
	if kindArg != "" {
		var err error
		if kind, err = store.NormalizeKind(kindArg); err != nil {
			return err
		}
	}
	p, l, s, err := findSpec(args[0])
	if err != nil {
		return err
	}
	if s.TaskID != "" {
		return fmt.Errorf("%s 已經開過任務 %s，請直接用 cairn log %s 記錄進度", s.ID, s.TaskID, s.TaskID)
	}
	if s.Status == store.SpecDone {
		return fmt.Errorf("%s 已經標成完成了", s.ID)
	}

	t := l.AddTask(s.Title, kind, nil)
	t.AddEntry("依規格書 "+s.ID+" 開始開發", nil, nil)
	t.SetStatus(store.StatusInProgress)
	s.TaskID = t.ID
	s.Updated = t.Updated
	if err := store.Save(p, l); err != nil {
		return err
	}
	fmt.Println(t.ID)
	fmt.Fprintf(os.Stderr, "%s → %s。完成後用 cairn done %s --summary … --verify …，規格書會一起標成已完成\n",
		s.ID, t.ID, t.ID)
	return nil
}

// indentLines 把整段文字每一行都加上前綴。
func indentLines(s, prefix string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i, l := range lines {
		lines[i] = prefix + l
	}
	return strings.Join(lines, "\n")
}
