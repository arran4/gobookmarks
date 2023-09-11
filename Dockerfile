FROM debian
VOLUME /data
ENV DB_CONNECTION_PROVIDER=sqlite3
ENV DB_CONNECTION_STRING="file:/data/a4webbookmarks.db?_loc=auto"
# ENV DB_CONNECTION_PROVIDER=mysql
# ENV DB_CONNECTION_STRING="a4webmb:......@tcp(.....:3306)/a4webbm?parseTime=true"
ENV EXTERNAL_URL=http://localhost:8080
ENV OAUTH2_CLIENT_ID=""
ENV OAUTH2_SECRET=""
EXPOSE 8080
EXPOSE 8443
COPY a4webbmws /bin/a4webbmws
RUN apt-get update && apt-get install -y \
  libsqlite3-0 \
  && rm -rf /var/lib/apt/lists/*
ENV PATH=/bin
ENTRYPOINT ["a4webbmws"]
