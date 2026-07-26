#!/bin/bash
# Stop hook：如果這一輪動過原始碼卻沒有更新 cairn 進度紀錄，就攔下來要 AI 補寫。
# 基準點是 .cairn/.session-mark，每次通過檢查後重新標記。
set -u

PROJECT="$(cd "$(dirname "$0")/../.." && pwd)"
LOG="$PROJECT/.cairn/log.json"
MARK="$PROJECT/.cairn/.session-mark"

command -v cairn >/dev/null 2>&1 || exit 0        # cairn 沒裝就不管
[ -f "$LOG" ] || exit 0                          # 這個專案沒在用 cairn
[ -f "$MARK" ] || { touch "$MARK"; exit 0; }     # 第一次執行，只建立基準

# 這一輪有沒有動過檔案？（跳過 cairn 自己與常見的相依/建置目錄，不限語言）
CHANGED=$(find "$PROJECT" \
    \( -name .git -o -name .cairn -o -name .claude \
       -o -name node_modules -o -name .venv -o -name env -o -name venv \
       -o -name __pycache__ -o -name dist -o -name build -o -name vendor \) -prune -o \
    -type f -newer "$MARK" -print 2>/dev/null | head -5)

if [ -z "$CHANGED" ]; then                       # 只是聊天、沒改東西
    touch "$MARK"
    exit 0
fi

if [ "$LOG" -nt "$MARK" ]; then                  # 紀錄檔比基準新 → 已經寫過了
    touch "$MARK"
    exit 0
fi

FILES=$(echo "$CHANGED" | sed "s|^$PROJECT/||" | tr '\n' ' ')
jq -nc --arg f "$FILES" '{
  decision: "block",
  reason: ("這一輪改動了檔案（" + $f + "）但沒有更新 cairn 紀錄。請先跑 `cairn list` 找對應任務（沒有就 `cairn add \"<簡短功能標題>\" --kind feature|fix|refactor|docs`）。做到一半就用 `cairn log <ID> \"<這步做了什麼>\" --files <修改的檔案> --new <新建立的檔案>`；功能已經完成就用 `cairn done <ID> --summary \"<實作了哪些內容>\" --verify \"<你怎麼確認它可用>\" --limits \"<沒做完或已知的限制，沒有就省略>\" --files <修改的檔案> --new <新建立的檔案>`。檔案一定要列全：新建立的放 --new、改過的既有檔案放 --files。內容寫給人看，不要貼程式碼；verify 要寫你實際做過的驗證，沒驗證過就照實說。寫完就可以結束。")
}'
