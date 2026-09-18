# osxcross ＋ Go：在 Linux 上交叉編出 macOS 版的 `cmd/psychicwar`（docs/spec/021 §4）。
#
#   docker build -t psychicwar-osxcross -f tools/docker/osxcross.Dockerfile tools/docker
#
# 由 `tools/macos-pack.sh` 在 image 不存在時自動建，建置那一步需要網路（apt-get）。
#
# 為什麼不沿用 `psychicwar-go-ebiten` 當底：`crazymax/osxcross:15.5-debian` 裡的
# 執行檔要 glibc 2.38，而那個 image 是 `golang:1.24-bookworm`（Debian 12，glibc 2.36），
# `osxcross-conf` 一跑就報 `version GLIBC_2.38 not found`。所以改以 Ubuntu 24.04 起底，
# 再把**同一份** Go 工具鏈從 `psychicwar-go-ebiten` 整個搬過來——版本仍然是專案鎖定的
# go1.24.13，與 Linux 版走的是同一支編譯器。
#
# Ebiten 的 macOS 後端走 Cocoa／OpenGL／Metal，**一定要 CGO**，所以 `CC` 得指到
# osxcross 的 clang wrapper；`CGO_ENABLED=0` 編不過（`undefined: glfw.Window`、
# `v.initDisplayLink undefined`）。
#
# ⚠ SDK 的授權只允許在 Apple 硬體上使用：**這個 image 只留本機，不上傳、不散布**，
# 發行包裡也不含 SDK 的任何內容。
FROM psychicwar-go-ebiten AS gobase
FROM crazymax/osxcross:15.5-debian AS osxcross

FROM ubuntu:24.04
# icnsutils 提供 png2icns（組 .icns 圖示）；file 用來認 Mach-O 與包裝腳本。
RUN apt-get update && apt-get install -y --no-install-recommends \
        clang lld llvm libxml2 zlib1g liblzma5 libssl3 \
        file ca-certificates icnsutils \
    && rm -rf /var/lib/apt/lists/*
COPY --from=osxcross /osxcross /osxcross
# 少這一行 `ld64` 起不來（找不到 `libxar.so.1`），而 clang 只會轉述成
# 「unable to execute command: No such file or directory」，看起來像編譯器壞掉。
RUN echo /osxcross/lib > /etc/ld.so.conf.d/osxcross.conf && ldconfig
COPY --from=gobase /usr/local/go /usr/local/go
ENV PATH="/osxcross/bin:/usr/local/go/bin:${PATH}"
