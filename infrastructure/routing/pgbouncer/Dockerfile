ARG PGBOUNCER_BASE_IMAGE=alpine:3.21
FROM ${PGBOUNCER_BASE_IMAGE}

RUN apk add --no-cache pgbouncer gettext

RUN addgroup -S pgbouncer \
    && adduser -S -D -H -h /var/empty -s /sbin/nologin -G pgbouncer pgbouncer

COPY pgbouncer.ini.tpl /etc/pgbouncer/pgbouncer.ini.tpl
COPY entrypoint.sh /usr/local/bin/pgbouncer-entrypoint.sh

RUN chmod +x /usr/local/bin/pgbouncer-entrypoint.sh

EXPOSE 6432

ENTRYPOINT ["sh", "/usr/local/bin/pgbouncer-entrypoint.sh"]
