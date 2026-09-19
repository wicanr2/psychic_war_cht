# Wine ＋ Xvfb：在 Linux 上實跑 Windows 發行包（`tools/windows-verify.sh`、docs/re/035）。
#
#   docker build -t psychicwar-wine -f tools/docker/wine.Dockerfile tools/docker
#
# 由 `tools/windows-verify.sh` 在 image 不存在時自動建，建置那一步需要網路（apt-get）。
#
# **為什麼不是 mingw**：Ebiten v2.9.9 的 Windows 後端用 purego 在執行期載 DLL
# （`opengl32.dll`、`d3d11.dll`、`user32.dll`…），`CGO_ENABLED=0 GOOS=windows go build`
# 就編得出來（實測，docs/re/035 §3.1），所以 Windows 版**不需要 mingw 交叉編譯器**，
# 建置沿用 `psychicwar-go-ebiten` 的同一份 Go 工具鏈。這個 image 存在的理由只有一個：
# 驗收要在**目標平台的執行環境**跑（`rulebook/82` 硬規則第 2 點），Linux 上最接近的就是 wine。
#
# ⚠ **底一定要用 Ubuntu 24.04，不能用專案其他 image 的 Debian 12。**
# Debian bookworm 的 wine 是 8.0，裡面**沒有 `bcryptprimitives.dll`**，而 Go 1.22 以後的
# 執行期在 `osinit` 就要載它拿 `ProcessPrng`。少了它，任何 Go 編出來的 `.exe` 都會在
# 進 `main` 之前死掉：
#     fatal error: bcryptprimitives.dll not found
#     runtime: panic before malloc heap initialized
# 這是 **wine 的年紀問題，不是產物的問題**（真正的 Windows 8 以後都有這個 DLL），
# 但症狀看起來像「我們的 exe 是壞的」。bookworm-backports 也沒有新版 wine（查過）。
# Ubuntu 24.04 的 libwine 9.0 檔案清單裡有 `bcryptprimitives.dll`（packages.ubuntu.com 查證）。
#
# ⚠ 只裝 64 位元的 wine，**不開 i386**：要驗的 `PsychicWar.exe` 是 PE32+／x86-64，
# 用不到 32 位元那一套，而 `dpkg --add-architecture i386` 會讓 apt 多抓一整份索引與
# 一整套 32 位元相依。這台的容器網路量到 60–200 KB/s，`libwine` 本身就 100 MB。
#
# ⚠ `WINEPREFIX` 要放 `$HOME` 底下，不要放 `/tmp`：wine 會拒絕在 sticky bit 的目錄建 prefix。
FROM ubuntu:24.04
ENV DEBIAN_FRONTEND=noninteractive
# wine：跑產物。xvfb／x11-utils：無頭畫面。imagemagick：`import` 截圖。
# mesa 的 GL 由 wine 的 winex11 驅動用，軟體算（llvmpipe）。
RUN apt-get update -o Acquire::Retries=5 \
    && apt-get install -y --no-install-recommends -o Acquire::Retries=5 \
        wine wine64 xvfb x11-utils imagemagick libgl1 libglx-mesa0 ca-certificates \
    && rm -rf /var/lib/apt/lists/*
