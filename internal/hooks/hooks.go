// Package hooks 負責把 Claude Code 的 Stop hook 與 CLAUDE.md 進度紀錄段落
// 安裝到目標專案，讓 cairn init --hook / cairn hook install 一鍵完成原本要手動複製的設定。
package hooks

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed templates/cairn-stop-check.sh
var stopCheckScript []byte

//go:embed templates/claude_progress.md
var progressSnippet string

const stopHookStatusMessage = "檢查 cairn 進度紀錄"

// Install 在 projectRoot 底下安裝 Stop hook script、合併進 .claude/settings.local.json、
// 並把進度紀錄段落補進 CLAUDE.md。可重複執行：已經裝過的部分會略過，不會覆蓋使用者
// 自訂的 settings.local.json 其餘內容或 CLAUDE.md 其餘段落。
//
// hook 的 command 是機器上的絕對路徑，所以寫進 settings.local.json（依 Claude Code
// 慣例是個人機器專屬、不進 git）而不是共用的 settings.json，避免把本機路徑外洩進 repo
// 歷史、也避免其他人 clone 下來後路徑對不上。
func Install(projectRoot string) ([]string, error) {
	var msgs []string

	scriptPath, err := installScript(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("寫入 hook script 失敗：%w", err)
	}
	msgs = append(msgs, "已寫入 "+scriptPath)

	settingsPath, added, err := installSettings(projectRoot, scriptPath)
	if err != nil {
		return nil, fmt.Errorf("寫入 %s 失敗：%w", settingsPath, err)
	}
	if added {
		msgs = append(msgs, "已在 "+settingsPath+" 加入 Stop hook")
	} else {
		msgs = append(msgs, settingsPath+" 已有這個 hook，略過")
	}

	claudeMDPath, added, err := installClaudeMD(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("寫入 CLAUDE.md 失敗：%w", err)
	}
	if added {
		msgs = append(msgs, "已在 "+claudeMDPath+" 加入進度紀錄段落")
	} else {
		msgs = append(msgs, claudeMDPath+" 已有進度紀錄段落，略過")
	}

	msgs = append(msgs, "注意：settings.local.json 裡的 hook 路徑是絕對路徑，專案搬家後要重新執行 cairn hook install")
	if reminder := gitignoreReminder(projectRoot); reminder != "" {
		msgs = append(msgs, reminder)
	}
	return msgs, nil
}

// gitignoreReminder 檢查 .gitignore 有沒有排除 settings.local.json；沒有的話回傳提醒文字，
// 避免使用者不小心把含本機絕對路徑的個人設定檔 commit 進 git。
func gitignoreReminder(projectRoot string) string {
	b, err := os.ReadFile(filepath.Join(projectRoot, ".gitignore"))
	if err == nil && strings.Contains(string(b), "settings.local.json") {
		return ""
	}
	return "提醒：.gitignore 裡似乎沒有排除 .claude/settings.local.json，建議加一行，避免把本機路徑 commit 進 git"
}

func installScript(projectRoot string) (string, error) {
	dir := filepath.Join(projectRoot, ".claude", "hooks")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	p := filepath.Join(dir, "cairn-stop-check.sh")
	if err := os.WriteFile(p, stopCheckScript, 0o755); err != nil {
		return "", err
	}
	return p, nil
}

// installSettings 把 Stop hook 合併進 .claude/settings.local.json，保留其餘既有內容
// （permissions、enabledPlugins…），並且如果同一個 command 已經存在就不重複加。
func installSettings(projectRoot, scriptPath string) (string, bool, error) {
	dir := filepath.Join(projectRoot, ".claude")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", false, err
	}
	p := filepath.Join(dir, "settings.local.json")
	abs, err := filepath.Abs(scriptPath)
	if err != nil {
		return p, false, err
	}

	cfg := map[string]any{}
	if b, err := os.ReadFile(p); err == nil {
		if err := json.Unmarshal(b, &cfg); err != nil {
			return p, false, fmt.Errorf("%s 不是合法 JSON：%w", p, err)
		}
	} else if !os.IsNotExist(err) {
		return p, false, err
	}

	hooksField, _ := cfg["hooks"].(map[string]any)
	if hooksField == nil {
		hooksField = map[string]any{}
	}
	stopList, _ := hooksField["Stop"].([]any)

	for _, entry := range stopList {
		if hasCommand(entry, abs) {
			return p, false, nil
		}
	}

	stopList = append(stopList, map[string]any{
		"hooks": []any{
			map[string]any{
				"type":          "command",
				"command":       abs,
				"timeout":       15,
				"statusMessage": stopHookStatusMessage,
			},
		},
	})
	hooksField["Stop"] = stopList
	cfg["hooks"] = hooksField

	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return p, false, err
	}
	if err := os.WriteFile(p, append(b, '\n'), 0o644); err != nil {
		return p, false, err
	}
	return p, true, nil
}

// hasCommand 判斷某個 Stop hook 群組項目底下是否已經有相同 command 的條目。
func hasCommand(entry any, command string) bool {
	m, ok := entry.(map[string]any)
	if !ok {
		return false
	}
	list, ok := m["hooks"].([]any)
	if !ok {
		return false
	}
	for _, h := range list {
		hm, ok := h.(map[string]any)
		if !ok {
			continue
		}
		if c, _ := hm["command"].(string); c == command {
			return true
		}
	}
	return false
}

// installClaudeMD 把進度紀錄段落補進 CLAUDE.md；已經有這段就不重複附加。
func installClaudeMD(projectRoot string) (string, bool, error) {
	p := filepath.Join(projectRoot, "CLAUDE.md")
	b, err := os.ReadFile(p)
	if err != nil && !os.IsNotExist(err) {
		return p, false, err
	}
	existing := string(b)
	if strings.Contains(existing, "## 進度紀錄") {
		return p, false, nil
	}

	out := progressSnippet
	if strings.TrimSpace(existing) != "" {
		out = strings.TrimRight(existing, "\n") + "\n\n" + progressSnippet
	}
	if err := os.WriteFile(p, []byte(out), 0o644); err != nil {
		return p, false, err
	}
	return p, true, nil
}
