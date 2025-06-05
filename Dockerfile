FROM debian
VOLUME /data
ENV EXTERNAL_URL=http://localhost:8080
ENV OAUTH2_CLIENT_ID=""
ENV OAUTH2_SECRET=""
ENV GBM_PROVIDER=github
ENV GBM_GITLAB_BASE_URL="https://gitlab.com/api/v4"
ENV GBM_NAMESPACE=""
ENV GBM_COMMIT_EMAIL=""
EXPOSE 8080
EXPOSE 8443
COPY gobookmarks /bin/gobookmarks
RUN apt-get update && apt-get install -y \
  ca-certificates \
  && rm -rf /var/lib/apt/lists/* && update-ca-certificates
ENV PATH=/bin
ENTRYPOINT ["gobookmarks"]
