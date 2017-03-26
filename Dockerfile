FROM alpine

ADD bin /app

WORKDIR /app

EXPOSE 80

VOLUME ["/etc/serve", "/usr/local/bin/serve"]

CMD ["/app/serve-server", "--config=/etc/serve/serve-server.yml"]
