FROM alpine

RUN apk add --update bash git openssh && rm -rf /var/cache/apk/*

WORKDIR /app

EXPOSE 8088

VOLUME ["/etc/serve", "/usr/local/bin", "/root/.ssh", "/tmp/serve"]

COPY bin/serve-server /app/serve-server

CMD ["/app/serve-server", "--config=/etc/serve/serve-server.yml"]
