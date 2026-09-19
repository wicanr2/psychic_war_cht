# 開發環境與產物組織

## 產物一律放 `dist-all/`

所有可交付的打包輸出到單一 `dist-all/`（整個目錄 gitignore）。
散在 `dist/`、`dist-mac/`、專案根目錄各處會分不清哪個是最新，也吃磁碟。

| 檔名 | 是什麼 |
|---|---|
| `PsychicWar-<版本>-x86_64.AppImage` | Linux，可散布，不含原版素材 |
| `PsychicWar-<版本>-macos.zip` | macOS universal，可散布，不含原版素材 |
| `PsychicWar-<版本>-with-data-x86_64.AppImage` | Linux，**含原版素材，純本機自用** |
| `PsychicWar-<版本>-with-data-macos.zip` | macOS，**含原版素材，純本機自用** |
| `psychic-war-<版本>-promo.mp4` | 推廣片 |

規則：

- **每個平台只留最新一份。** `tools/package.sh` 產出前會刪同型的舊檔。
- **`-with-data` 絕不推 git、絕不上傳。** 它含 `PW.EXE` 與原版資料檔。
  可散布版打包時做 leak-scan，掃到原版檔就中止。
- **中間 staging 放 `workplace/pkg-stage/`，壓完就刪。** 別把解開的 bundle 留在磁碟。
- 要清空間就直接刪 `dist-all/` 裡的舊檔。
  ⚠ 這裡的「清舊版」**只限本專案 `dist-all/` 底下自己產的打包檔**。
  **不含 docker image／volume／build cache**，那些是跨專案共用資源，
  任何 `prune`／`rmi` 一律禁止（`rules/30-lcy-agent-boundaries`）。

指令：

```sh
tools/package.sh appimage                     # 只出 Linux
tools/package.sh all                          # AppImage ＋ macOS ＋ 收推廣片
PSYCHICWAR_WITH_DATA=1 tools/package.sh all   # 再多出兩份含原版素材的本機版
tools/package-check.sh dist-all/<產物>        # 解開產物驗收
```

規格是 `docs/spec/021-packaging.md`。

## Linux 用 AppImage

Linux 只出 AppImage，不出 tar.gz。AppImage 的結構就是「type2 runtime ＋ squashfs 映像」
串接，所以 `tools/appimagetool.sh` 只要 `mksquashfs` 與官方 runtime，不裝 appimagetool
本身（它自己也是 AppImage，在容器裡跑要 FUSE 或先解壓）。

`AppRun` 只轉呼叫。資料檔跟著執行檔走、存檔落在使用者資料目錄，
兩件事都由程式自己處理（`apps/psychicwar/paths.go`），因為 squashfs 是唯讀的。

## docker image

| image | 用途 | 建法 |
|---|---|---|
| `psychicwar-go-ebiten` | Go ＋ Ebiten 建置、Xvfb 實跑 | `tools/go-ebiten.sh` 第一次自動 build |
| `psychicwar-osxcross` | macOS 交叉編譯 | `tools/macos-pack.sh` 第一次自動 build |
| `psychicwar-appimage` | AppImage 打包 | `tools/appimagetool.sh` 第一次自動 build |
| `psychicwar-video` | 推廣片合成 | `tools/video.sh` 第一次自動 build |
| `psychicwar-py` | Python 工具 | `tools/py.sh` |

build 那一步要網路，之後一律 `--network none`。
**只清理自己 `--rm` 建立的 container；任何 `docker image/system/volume/builder prune` 或 `rmi` 一律禁止。**
