FROM scratch
ENTRYPOINT ["/a4webbmws"]
VOLUME /data
ENV DB_CONNECTION_PROVIDER=sqlite3
ENV DB_CONNECTION_STRING="file:/data/a4webbookmarks.db?_loc=auto"
# ENV DB_CONNECTION_PROVIDER=mysql
# ENV DB_CONNECTION_STRING="a4webmb:......@tcp(.....:3306)/a4webbm?parseTime=true"
ENV EXTERNAL_URL=http://localhost:8080
ENV OAUTH2_CLIENT_ID=""
ENV OAUTH2_SECRET=""
COPY ./dist/a4webbm_linux_amd64_v1/a4webbm /
