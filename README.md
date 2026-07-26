# cairn

終端裡的 AI 開發紀錄。給「AI 輔助開發」設計：**AI 用子命令寫進度，人用 TUI 看進度**，兩邊共讀同一份 `.cairn/log.json`。

```
  cairn · LearnDjango · 完成 2 ・ 未完成 2
  ▌完成紀錄 2    進行中 2
╭──────────────────────────────╮╭────────────────────────────────────────╮
│ ▸ ● 修正註冊後未自動登入      ││ 修正註冊後未自動登入                    │
│     T-002  修正  07-25       ││ T-002  [修正]  ● 完成                  │
│   ● 美化登入頁面              ││ 完成於 2026-07-25 22:33 ・ 歷時 12 分鐘 │
│     T-001  新功能  07-25     ││                                        │
│                              ││ 功能說明                                │
│                              ││   註冊成功後直接呼叫 login() 建立       │
│                              ││   session，不再導回登入頁重輸一次。      │
│                              ││                                        │
│                              ││ 驗證方式                                │
│                              ││   用新帳號註冊，確認直接進到首頁。       │
│                              ││                                        │
│                              ││ 改動檔案（1）                           │
│                              ││   Learn_django/app/views.py            │
╰──────────────────────────────╯╰────────────────────────────────────────╯
  tab 換頁 · j/k 移動 · a 新功能 · n 記步驟 · s 切狀態 · q 離開
```

畫面寬度不足 76 欄時（例如 tmux 分割出來的窄 pane）會自動改成上下堆疊。

## 安裝

```sh
npm install -g @junming_h/cairn
```

或不安裝直接用：`npx @junming_h/cairn`

自己建置（需要 Go 1.23+）：

```sh
git clone https://github.com/junming0106/cairn.git && cd cairn
go build -o cairn . && mv cairn /opt/homebrew/bin/   # 或任何 PATH 目錄
```

## 使用

```sh
cd ~/your-project
cairn init                       # 建立 .cairn/log.json
cairn                            # 開 TUI
```

紀錄檔會從目前目錄往上層尋找 `.cairn/log.json`，所以在專案任何子目錄下都能用。

### TUI 按鍵

| 鍵                                | 動作                                                     |
| --------------------------------- | -------------------------------------------------------- |
| `tab` / `h` `l` / `1` `2` `3` `4` | 切換【完成】【進行中】【規格書】【技能】四個頁籤         |
| `j` / `k`                         | 上下選項目                                               |
| `J` / `K`                         | 捲動右側詳情                                             |
| `ctrl+d` / `ctrl+u`               | 詳情翻半頁                                               |
| `g` / `G`                         | 跳到最上 / 最下                                          |
| `n`                               | 為選取項目補一則備註（只有備註是人寫的，其餘由 AI 維護） |
| `s`                               | 切換規格書的待完成 / 已完成（只有規格書頁）              |
| `r`                               | 立即重讀檔案（技能頁則重新掃描）                         |
| `q`                               | 離開                                                     |

TUI 每 2 秒檢查檔案是否被改動，AI 在背景寫入的進度會自動出現，底部顯示「已同步外部更新」。

### 規格書頁

開發紀錄是「做完了什麼」，規格書是「還沒動手前，要做什麼」。

**Claude Code 進行討論，規格書頁負責紀錄討論結果。** 你在旁邊的 AI 視窗照常聊，把一項功能要做什麼講清楚之後，請它寫進規格書；CAIRN規格書頁就會出現這一份，可以隨時翻回來看當初談好的內容。每一份都有「待完成 / 已完成」的標籤。

```
│  ◇ 文章分類功能        │  規格書
│    S-001  待完成  剛剛  │ ▌ 文章分類功能
│  ◆ 匯出 CSV            │
│    S-002  已完成  2 天前│ S-001  ◇ 待完成
```

在這一頁按 `s` 可以自己切換待完成 / 已完成；`J`/`K` 捲動正文。其餘由 AI 從命令列維護：

```sh
cairn spec add "文章分類功能" --body "<正文>"     # 談定之後寫成規格書，印出 S-001
cairn spec list [--status todo|done] [--json]
cairn spec show S-001 [--json]
cairn spec set S-001 --body "<正文>"             # 需求改了就整份改寫，長文用 --body-file <檔案>
cairn spec status S-001 todo|done
cairn spec build S-001 [--kind feature|fix|refactor|docs]   # 依規格開任務，印出 T-001
```

`spec build` 開出來的任務跑完 `cairn done` 時，對應的規格書會自動標成已完成。規格 ID 可用 `S-001` 或 `1`；`cairn show T-001` 會標出它是依哪份規格書做的。

### 技能頁

列出這個專案目前可用的 AI 技能，來源有三種：

| 來源 | 位置                                                  |
| ---- | ----------------------------------------------------- |
| 專案 | `<專案>/.claude/skills/<名稱>/SKILL.md`               |
| 全域 | `~/.claude/skills/<名稱>/SKILL.md`                    |
| 插件 | 已啟用插件（設定檔的 `enabledPlugins`）帶的 `skills/` |

右側顯示技能的說明與 `SKILL.md` 全文，可用 `J`/`K` 捲動。按 `r` 重新掃描。

### 並排開發（tmux）

```sh
cairn dev            # 左邊 claude、右邊紀錄頁
cairn dev codex      # 換成 codex
```

需要 tmux（`brew install tmux`）。已經在 tmux 裡執行時會直接切分目前的視窗。

### 命令列（給 AI 用）

```sh
cairn add "美化登入頁面" --kind feature        # 印出 T-001，kind: feature|fix|refactor|docs
cairn log T-001 "把 login.html 包成白色卡片" --files templates/login.html
cairn done T-001 \
  --summary "登入頁改用 form-container 樣式並包成白色卡片…" \
  --verify  "開 /login/ 目視確認排版，並用錯誤密碼送出確認錯誤訊息正常" \
  --limits  "註冊頁還沒套用同樣樣式" \
  --files   templates/login.html,static/style.css
cairn status T-001 blocked                     # todo|in_progress|blocked
cairn list                                     # 純文字，適合 AI 讀
cairn show T-001
cairn path
```

- `cairn done` 的 `--summary`（功能說明）與 `--verify`（驗證方式）是**必填** — 完成紀錄要讓人看得懂做了什麼、怎麼確認可用。
- `cairn status <ID> done` 會被擋下並提示改用 `cairn done`。
- ID 可用 `T-001` 或 `1`；狀態接受簡寫（`wip` → `in_progress`）；類型接受簡寫（`feat` → `feature`）。
- 寫檔是「暫存檔 + rename」，TUI 不會讀到半截 JSON。

## 讓 AI 自動維護紀錄

把這段加進專案的 `CLAUDE.md`（或 `AGENTS.md`）：

```markdown
## 進度紀錄

本專案用 `cairn` 追蹤開發進度，人會在終端用 TUI 查看。

- 開始一項功能前：`cairn add "<簡短功能標題>" --kind feature|fix|refactor|docs`，記下回傳的 ID。
- 開發過程中每個步驟：`cairn log <ID> "<這一步做了什麼>" --files <改動的檔案>`
- 功能完成時：`cairn done <ID> --summary "<實作了哪些內容>" --verify "<你怎麼確認它可用>" --limits "<已知限制，沒有就省略>" --files <改動的檔案>`
- 被外部因素卡住時：`cairn status <ID> blocked`
- 接手既有工作前先讀 `cairn list` 與 `cairn show <ID>`，確認上次進行到哪。

內容寫給人看，不要貼程式碼。`--verify` 要寫你實際做過的驗證，沒驗證過就照實說。

## 規格書

跟人談定一項功能要做什麼之後，先寫成規格書再動手：

- `cairn spec add "<標題>" --body "<正文>"` — 正文要寫清楚做什麼、範圍到哪、怎麼算完成（長文用 `--body-file`）
- 動手前先 `cairn spec list` / `cairn spec show <ID>`，確認要做的是哪一份、有沒有已經做過
- `cairn spec build <ID>` 依規格開出開發任務，之後照上面的紀錄流程走
- 談話中需求改了就 `cairn spec set <ID> --body "<新正文>"` 整份改寫
```

再搭配一個 Stop hook，就能在「動過程式碼卻沒寫紀錄」時擋下 AI 要它補寫。這段加上 hook 不用手動複製，跑 `cairn init --hook`（新專案）或事後 `cairn hook install`（既有專案，可重複執行）就會自動產生 `.claude/hooks/cairn-stop-check.sh`、合併進 `.claude/settings.local.json`（機器專屬、不進 git），並把上面這段補進 `CLAUDE.md`。

## 資料格式

`.cairn/log.json`，純文字、可進 git、可 `git diff` 看歷史：

```json
{
  "version": 1,
  "project": "LearnDjango",
  "specs": [
    {
      "id": "S-001",
      "title": "文章分類功能",
      "status": "todo",
      "body": "## 目標\n每篇文章可以掛一個分類。\n\n## 完成條件\n- 後台能建立分類…",
      "task_id": "T-002",
      "created": "2026-07-26T09:10:00+08:00",
      "updated": "2026-07-26T09:40:00+08:00"
    }
  ],
  "tasks": [
    {
      "id": "T-001",
      "title": "美化登入頁面",
      "kind": "feature",
      "status": "done",
      "summary": "登入頁改用既有的 form-container 樣式並包成白色卡片…",
      "verify": "開 /login/ 目視確認排版，並用錯誤密碼送出一次…",
      "limits": "註冊頁還沒套用同樣樣式。",
      "created": "2026-07-25T22:21:00+08:00",
      "updated": "2026-07-25T22:33:00+08:00",
      "completed": "2026-07-25T22:33:00+08:00",
      "entries": [
        {
          "time": "2026-07-25T22:25:00+08:00",
          "note": "把 login.html 包成白色卡片",
          "files": ["Learn_django/app/templates/login.html"]
        }
      ]
    }
  ]
}
```

舊版紀錄（沒有 `kind` / `summary` / `verify` / `specs`）仍可讀取：類型預設為 `feature`，功能說明退回最後一筆步驟說明，沒有 `specs` 就是還沒用過規格書。

## 專案結構

```
main.go                       CLI 子命令與進入點
spec_cmd.go                   cairn spec 子命令
internal/store/store.go       資料模型、原子寫入、狀態正規化
internal/store/spec.go        規格書資料模型與狀態
internal/ui/ui.go             Bubble Tea 介面（完成 / 進行中 / 規格書 / 技能 四頁籤）
internal/skills/skills.go     掃描專案、全域與插件的 AI 技能
npm/                          npm 套件：postinstall 下載對應平台執行檔
.github/workflows/release.yml 推 v* 標籤時建置四個平台並發佈 Release
```
