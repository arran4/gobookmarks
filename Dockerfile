FROM debian
VOLUME /data
VOLUME /etc/gobookmarks
ENV EXTERNAL_URL=http://localhost:8080
ENV GITHUB_CLIENT_ID=""
ENV GITHUB_SECRET=""
ENV GITLAB_CLIENT_ID=""
ENV GITLAB_SECRET=""
ENV GBM_CSS_COLUMNS=""
ENV GBM_NAMESPACE=""
ENV FAVICON_CACHE_DIR=/data/favicons
ENV FAVICON_CACHE_SIZE=20971520
ENV GOBM_ENV_FILE=/etc/gobookmarks/gobookmarks.env
ENV GOBM_CONFIG_FILE=/etc/gobookmarks/config.json
EXPOSE 8080
EXPOSE 8443
COPY gobookmarks /bin/gobookmarks
RUN apt-get update && apt-get install -y \
  ca-certificates \
  && rm -rf /var/lib/apt/lists/* && update-ca-certificates
ENV PATH=/bin
ENTRYPOINT ["gobookmarks"]
