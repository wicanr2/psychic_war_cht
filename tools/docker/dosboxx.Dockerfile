# 本專案原版AdLib錄音工具，恢復缺失的civ1-dosboxx-input:20260830能力。
# 來源沿civ1/docker/dosboxx-bridge-civ1.Dockerfile；此處不修改civ1工具。
# context：Git匯出的DOSBox-X commit 5fcf624b787e1017273b313de6f9a70f12422102。
# 入口與來源保全見CONTEXT、WORKLOG及tools/dosboxx-audio.sh。
FROM debian:trixie-slim AS build

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential automake autoconf libtool pkg-config git ca-certificates \
    libncurses-dev libglu1-mesa-dev libgl-dev \
    libx11-dev libxrandr-dev libxext-dev libxcursor-dev libxinerama-dev libxi-dev \
    libpng-dev zlib1g-dev libfreetype-dev libslirp-dev \
    libfluidsynth-dev libavcodec-dev libavformat-dev libavutil-dev libswscale-dev \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /src
COPY . /src
RUN chmod +x autogen.sh build-debug configure 2>/dev/null || true
RUN ./build-debug > /tmp/build.log 2>&1 || { tail -40 /tmp/build.log; exit 1; }
RUN test -x src/dosbox-x \
    && grep -q 'define C_DEBUG 1' config.h \
    && grep -q 'define C_HEAVY_DEBUG 1' config.h

FROM debian:trixie-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
    libncurses6 libx11-6 libxrandr2 libxext6 libxcursor1 libxinerama1 libxi6 \
    libpng16-16t64 zlib1g libfreetype6 libslirp0 \
    libfluidsynth3 libavcodec61 libavformat61 libavutil59 libswscale8 \
    xvfb x11-apps x11-utils xdotool ffmpeg imagemagick procps \
    python3 unzip util-linux netcat-openbsd \
    && rm -rf /var/lib/apt/lists/*
COPY --from=build /src/src/dosbox-x /usr/local/bin/dosbox-x
WORKDIR /work
