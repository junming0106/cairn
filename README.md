# cairn

> 『石標』（Cairn）是登山者在山林中用來指引方向的記號。如同現代工程師在協同多個 AI 進行開發時，驀然回首，卻發現自己已迷失在數千行程式碼之中。期盼這個專案能成為你的石標，引領你找回正確的開發道路！

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

**1. 安裝 Node.js / npm**（已經有的話跳過，`npm -v` 可以確認）

```sh
brew install node        # macOS
sudo apt install nodejs npm   # Debian / Ubuntu
```

或到 [nodejs.org](https://nodejs.org) 下載 LTS 版。

**2. 安裝 tmux**（`cairn dev` 分割畫面要用，其餘指令不需要，可跳過）

```sh
brew install tmux        # macOS
sudo apt install tmux    # Debian / Ubuntu
sudo dnf install tmux    # Fedora
```

**3. 安裝 cairn**

```sh
npm install -g @junming_h/cairn
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

需要 tmux（見上面「安裝」第 2 步）。已經在 tmux 裡執行時會直接切分目前的視窗。

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

跑 `cairn init --hook`（新專案）或事後 `cairn hook install`（既有專案，可重複執行），會自動：

- 把「進度紀錄」「規格書」的使用說明寫進 `CLAUDE.md`（已經寫過就不會重複寫）
- 裝一個 Stop hook：動過程式碼卻沒寫紀錄時擋下 AI，要求先補寫
- hook script 放在 `.claude/hooks/cairn-stop-check.sh`，設定合併進 `.claude/settings.local.json`（機器專屬、不進 git）

不是用 Claude Code（例如 Codex）的話，把同一段說明自己寫進 `AGENTS.md` 即可，cairn 指令本身跟工具無關，只是沒有自動擋下未寫紀錄的 hook。

## 專案結構

```
cairn/
├── main.go                        CLI 子命令與進入點
├── spec_cmd.go                    cairn spec 子命令
├── internal/
│   ├── store/
│   │   ├── store.go               資料模型、原子寫入、狀態正規化
│   │   └── spec.go                規格書資料模型與狀態
│   ├── ui/
│   │   └── ui.go                  Bubble Tea 介面（完成 / 進行中 / 規格書 / 技能 四頁籤）
│   ├── skills/
│   │   └── skills.go              掃描專案、全域與插件的 AI 技能
│   └── hooks/
│       └── hooks.go               安裝 Stop hook 與 CLAUDE.md 進度紀錄段落
├── npm/                           npm 套件：postinstall 下載對應平台執行檔
└── .github/workflows/release.yml  推 v* 標籤時建置四個平台並發佈 Release
```
