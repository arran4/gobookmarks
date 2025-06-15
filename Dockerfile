FROM alpine
VOLUME /var/cache/gobookmarks
VOLUME /etc/gobookmarks
VOLUME /var/lib/gobookmarks
ENV EXTERNAL_URL=http://localhost:8080
ENV GITHUB_CLIENT_ID=""
ENV GITHUB_SECRET=""
ENV GITLAB_CLIENT_ID=""
ENV GITLAB_SECRET=""
ENV GBM_CSS_COLUMNS=""
ENV GBM_NAMESPACE=""
ENV GBM_TITLE=""
ENV FAVICON_CACHE_DIR=/var/cache/gobookmarks/favcache
ENV FAVICON_CACHE_SIZE=20971520
ENV GITHUB_SERVER=""
ENV GITLAB_SERVER=""
ENV LOCAL_GIT_PATH=/var/lib/gobookmarks/localgit
ENV GOBM_ENV_FILE=/etc/gobookmarks/gobookmarks.env
ENV GOBM_CONFIG_FILE=/etc/gobookmarks/config.json
EXPOSE 8080
EXPOSE 8443
COPY gobookmarks /bin/gobookmarks
RUN apk add --no-cache ca-certificates && update-ca-certificates
ENV PATH=/bin
ENTRYPOINT ["gobookmarks"]
