# 沿用本機SDK工具鏈；換成本專案Go1.24.13。基底image ID列入收據。
# SDK及image只留本機，不公開散布。
FROM psychicwar-go-ebiten:latest AS go
FROM hr-osxcross:1.26.7-15.5-r1
USER root
RUN rm -rf /usr/local/go
COPY --from=go /usr/local/go /usr/local/go
USER 1000:1000
ENV PATH="/osxcross/bin:/usr/local/go/bin:${PATH}"
