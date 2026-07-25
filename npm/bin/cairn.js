#!/usr/bin/env node
// 轉接腳本：把參數原封不動交給下載好的原生執行檔，並沿用它的結束碼。
const path = require("path");
const { spawnSync } = require("child_process");
const fs = require("fs");

const bin = path.join(__dirname, "cairn");
if (!fs.existsSync(bin)) {
  console.error("找不到 cairn 執行檔，請重新安裝：npm rebuild cairn");
  process.exit(1);
}
const r = spawnSync(bin, process.argv.slice(2), { stdio: "inherit" });
process.exit(r.status === null ? 1 : r.status);
