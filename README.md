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
npm install -g @__GH_USER__/cairn
```

或不安裝直接用：`npx @__GH_USER__/cairn`

自己建置（需要 Go 1.23+）：

```sh
git clone https://github.com/__GH_USER__/cairn.git && cd cairn
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

| 鍵 | 動作 |
|---|---|
| `tab` / `h` `l` / `1` `2` | 切換【完成紀錄】【進行中】兩個頁籤 |
| `j` / `k` | 上下選項目 |
| `J` / `K` | 捲動右側詳情 |
| `ctrl+d` / `ctrl+u` | 詳情翻半頁 |
| `g` / `G` | 跳到最上 / 最下 |
| `a` | 新增功能 |
| `n` | 為選取項目記一個步驟 |
| `s` | 循環狀態 待辦 → 進行中 → 卡住 → 完成 |
| `r` | 立即重讀檔案 |
| `q` | 離開 |

TUI 每 2 秒檢查檔案是否被改動，AI 在背景寫入的進度會自動出現，底部顯示「已同步外部更新」。

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
```

再搭配一個 Stop hook，就能在「動過程式碼卻沒寫紀錄」時擋下 AI 要它補寫 — 範例見 `LearnDjango/.claude/`。

## 資料格式

`.cairn/log.json`，純文字、可進 git、可 `git diff` 看歷史：

```json
{
  "version": 1,
  "project": "LearnDjango",
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

舊版紀錄（沒有 `kind` / `summary` / `verify`）仍可讀取：類型預設為 `feature`，功能說明退回最後一筆步驟說明。

## 專案結構

```
main.go                       CLI 子命令與進入點
internal/store/store.go       資料模型、原子寫入、狀態正規化
internal/ui/ui.go             Bubble Tea 介面（完成紀錄 / 進行中 雙頁籤）
npm/                          npm 套件：postinstall 下載對應平台執行檔
.github/workflows/release.yml 推 v* 標籤時建置四個平台並發佈 Release
```
