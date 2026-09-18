# AppImage 打包（docs/spec/021 §4）。
#
# AppImage 的結構就是「runtime（一支小 ELF）＋ squashfs 映像」串接起來，所以這裡不裝
# appimagetool——它自己也是 AppImage，在容器裡跑要 FUSE 或先解壓，多一層沒有好處。
# 只要 mksquashfs 與官方的 type2 runtime 就夠。
#
# runtime 在 build 階段下載（build 有網路），打包時 `--network none`。
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
        squashfs-tools ca-certificates curl file \
    && curl -fsSLo /opt/runtime-x86_64 \
        https://github.com/AppImage/type2-runtime/releases/download/continuous/runtime-x86_64 \
    && chmod +x /opt/runtime-x86_64 \
    && apt-get purge -y curl && apt-get autoremove -y \
    && rm -rf /var/lib/apt/lists/*
