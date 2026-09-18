# 推廣片合成（skill `game-promo-video-ffmpeg`）：ffmpeg ＋ ImageMagick ＋ CJK 字型。
# 本專案自己的 image（`psychicwar-video`），不動共用資源。
# 預建的理由：每跑一次 apt 裝 200MB，時間都花在安裝上。
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
        ffmpeg imagemagick fonts-noto-cjk \
    && sed -i 's/rights="none" pattern="@\*"/rights="read" pattern="@*"/' /etc/ImageMagick-6/policy.xml \
    && rm -rf /var/lib/apt/lists/*
