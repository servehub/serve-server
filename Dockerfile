FROM alpine

RUN apk add --update bash && rm -rf /var/cache/apk/*

WORKDIR /app

EXPOSE 8088

VOLUME ["/etc/serve", "/usr/local/bin/serve"]

ADD bin /app

CMD ["/app/serve-server", "--config=/etc/serve/serve-server.yml"]
