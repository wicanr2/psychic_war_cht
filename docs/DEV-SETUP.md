# 開發、驗收與打包

玩家介紹、下載與操作見 [README](../README.md)。本頁保存建置及驗收入口；現況與證據以 [CONTEXT](../CONTEXT.md) 為準，逐輪歷程見 [WORKLOG](../WORKLOG.md)。

## 工作環境

分析、建置、測試、GUI、音訊與影片一律在 Docker 內執行。共用主機的資源、掛載、UID、原版唯讀與清理規則見 [AGENTS §10](../AGENTS.md#10-dockergit-與發行)。

- Go 1.24.13、Ebiten 前端：[tools/go-ebiten.sh](../tools/go-ebiten.sh)、[Go Dockerfile](../tools/docker/go-ebiten.Dockerfile)。
- dosgolem：`go.mod` 使用 `worktrees/dosgolem` 的本機分支；本專案 1.0.0 所用提交見 CONTEXT。
- 原版資料：`workplace/original/psychic-war/`，必須含完整 DOS 英文版資料。版本與雜湊見 [001](re/001-pw-exe-first-look.md)；不加入 Git。
- 譯文：`text/*.json`。改譯文後執行 [tools/font/bake.sh](../tools/font/bake.sh)，字型來源放 `workplace/font-src/`。

一般產品測試範圍是 `./apps/... ./cmd/...`，在 Xvfb 內執行；`workplace/` 的診斷 Go 檔不是獨立套件，不納入全庫通配測試。Go 工作使用 `GOMAXPROCS=2` 與 `-p 1` 控制程序數。

## 正式封包

版號使用 `v.<主版>.<次版>.<修訂版>-YYYYMMDD`。先完成提交，再建立同名 tag，從乾淨工作樹打包。已發布 tag 與附件不移動或覆寫。

```sh
PSYCHICWAR_RELEASE_THEME=workplace/hd/<已驗證主題> tools/package.sh v.1.0.0-20261008
```

[package.sh](../tools/package.sh) 負責 Docker 編譯與組裝，三平台同時建立公開包與本機完整版：

| 路徑 | 內容 |
|---|---|
| `dist-all/<版本>/patch/` | Linux AppImage、Windows ZIP、macOS universal ZIP，只含可公開程式與已確認可散布素材 |
| `dist-all/<版本>/full-local/` | 含原版資料與全部 HD 的三平台本機完整版，檔名有 `-with-data` |
| `dist-all/<版本>/promo/` | 影片、源錄影、抽幀、影音檢查與權利資料，僅本機 |
| `dist-all/<版本>/smoke/` | 實際封包啟動、存檔、平台結構與發布核對收據 |
| `dist-all/<版本>/SHA256SUMS.json` | 正式產物大小、SHA-256、建置提交與權利分類 |

`dist-all/` 不進 Git。原版與全部 HD 不附於公開包；使用者另授權的 README 實機展示截圖例外，範圍見 [024 §7](spec/024-hd-theme.md#7-授權)。保留已發布版本的可追溯資料，不以清空間為由刪除或改寫遠端 Release。

中間檔放 `workplace/release-stage/<版本>/`；研究及擷取資料放 `workplace/`。Docker image、volume、build cache 不在產物清理範圍，不執行 `prune` 或 `rmi`。

## 驗實際產物

以下 Python 工具在各自 Docker 環境內執行，不在主機直接執行：

| 工具 | 驗證內容 |
|---|---|
| [release-stage.py](../tools/release-stage.py) | 封包明確清單、授權文件、原版／HD 邊界及內部 manifest |
| [release-verify.py](../tools/release-verify.py) | 六包內容與 SHA、Windows PE、macOS 雙架構／簽章／最低版本／相依，以及 Linux 實際 GUI、DAT、F10 與缺字型負對照 |
| [release-wine-smoke.py](../tools/release-wine-smoke.py) | Windows 兩個實際 ZIP 正常開機、F10 與 AppData 存檔；等待可見視窗，初始化上限 150 秒 |
| [release-manifest.py](../tools/release-manifest.py) | 六包驗收後產生 SHA 清單與交付收據；`--promo-required` 另外要求實際遊玩影片通過 |

Linux 實跑、Wine 與 macOS 結構檢查分開記錄。Windows／Mac 真機驗收仍屬 [#44](https://github.com/wicanr2/psychic_war_cht/issues/44)。遊戲存檔與即時存檔都落在使用者資料目錄，不能寫進 AppImage 或 `.app`。

## 實際遊玩推廣片

1. [capture-live.py](../tools/promo/capture-live.py) 從實際完整版 AppImage 正常開機，以原版按鍵遊玩、移動與戰鬥，並用 `Shift+F5` 切換原版／HD。保留輸入、stats、切換截圖及源錄影。
2. [record-live.py](../tools/promo/record-live.py) 在既有 FFmpeg 容器擷取 X11。GUI 容器使用專用 socket 及 `--ipc=shareable`；錄影器只加入該 GUI 容器的 IPC，不分享主機 IPC。
3. [make-live.py](../tools/promo/make-live.py) 合成片頭、實際錄影與片尾，配樂只使用 `workplace/dosboxx-audio/title-adlib.wav`。
4. [verify-live.py](../tools/promo/verify-live.py) 驗 codec、尺寸、幀數、音訊、黑幀、靜音、凍結、遊玩狀態與四次切換，再逐幕目視抽幀。

原版配樂擷取：[capture-original-adlib.sh](../tools/promo/capture-original-adlib.sh)。音訊技術檢查不等於人耳驗收；起跑負載須依 [020](spec/020-audio-device-measurement.md) 的契約核對。含原版音樂與 HD 的影片只留本機。

## 技術與研究入口

- [規格](spec/)：中文化、HD、原版操作與資料契約。
- [研究紀錄](re/)：原版位址、格式、音訊與對拍收據；README 不重複這些記錄。
- [038](re/038-hd-theme-feasibility.md)：HD 原位、遮擋、動作與原版狀態驗證。
- [未完成項](worklist.json)：工作條目與對應 Issue。
- [HD 總覽](../tools/hd/overview.py)：重生敵人與盟友的原版／HD 四張本機總覽，不代表全部圖號正常 GUI 已驗。
