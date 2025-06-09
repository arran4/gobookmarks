FROM debian
VOLUME /data
VOLUME /etc/gobookmarks
ENV EXTERNAL_URL=http://localhost:8080
ENV OAUTH2_CLIENT_ID=""
ENV OAUTH2_SECRET=""
ENV GBM_CSS_COLUMNS=""
ENV GBM_NAMESPACE=""
ENV GBM_PROVIDER=github
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
