FROM alpine:3.5

RUN apk add --update --no-cache \
      ca-certificates \
      bash \
      git \
      openssh

WORKDIR /app

EXPOSE 8088

VOLUME ["/etc/serve", "/usr/local/bin", "/root/.ssh", "/tmp/serve"]

COPY bin/serve-server /app/serve-server

CMD ["/app/serve-server", "--config=/etc/serve/serve-server.yml"]
