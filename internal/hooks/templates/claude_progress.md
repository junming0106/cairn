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
