#!/usr/bin/env node
// postinstall：從 GitHub Release 下載對應平台的 cairn 執行檔。
const fs = require("fs");
const path = require("path");
const https = require("https");
const zlib = require("zlib");
const { execFileSync } = require("child_process");

const pkg = require("./package.json");
const REPO = pkg.repository.url.replace(/^git\+/, "").replace(/\.git$/, "");
const VERSION = "v" + pkg.version;

const PLATFORMS = {
  "darwin-arm64": "darwin-arm64",
  "darwin-x64": "darwin-amd64",
  "linux-arm64": "linux-arm64",
  "linux-x64": "linux-amd64",
};

function fail(msg) {
  console.error("cairn 安裝失敗：" + msg);
  console.error("你也可以自己建置：git clone " + REPO + " && cd cairn && go build -o cairn .");
  process.exit(1);
}

function download(url, dest, redirects = 0) {
  return new Promise((resolve, reject) => {
    if (redirects > 5) return reject(new Error("重新導向次數過多"));
    https
      .get(url, { headers: { "User-Agent": "cairn-installer" } }, (res) => {
        if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
          res.resume();
          return resolve(download(res.headers.location, dest, redirects + 1));
        }
        if (res.statusCode !== 200) {
          res.resume();
          return reject(new Error("HTTP " + res.statusCode + " — " + url));
        }
        const out = fs.createWriteStream(dest);
        res.pipe(zlib.createGunzip()).pipe(out);
        out.on("finish", () => out.close(resolve));
        out.on("error", reject);
      })
      .on("error", reject);
  });
}

async function main() {
  const key = `${process.platform}-${process.arch}`;
  const target = PLATFORMS[key];
  if (!target) fail(`不支援的平台 ${key}（支援：${Object.keys(PLATFORMS).join(", ")}）`);

  const binDir = path.join(__dirname, "bin");
  fs.mkdirSync(binDir, { recursive: true });
  const tarPath = path.join(binDir, "cairn.tar");
  const url = `${REPO}/releases/download/${VERSION}/cairn-${target}.tar.gz`;

  try {
    await download(url, tarPath);
    // tar.gz 已在下載時解壓成 tar，這裡只取出裡面的執行檔
    execFileSync("tar", ["-xf", tarPath, "-C", binDir], { stdio: "ignore" });
    fs.renameSync(path.join(binDir, `cairn-${target}`), path.join(binDir, "cairn"));
    fs.chmodSync(path.join(binDir, "cairn"), 0o755);
    fs.unlinkSync(tarPath);
  } catch (err) {
    fail(err.message);
  }
  console.log("cairn " + VERSION + " 安裝完成（" + key + "）");
}

main();
