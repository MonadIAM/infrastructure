ARG POSTGRES_BASE_IMAGE=postgres:17.2-alpine3.21
FROM ${POSTGRES_BASE_IMAGE}

RUN apk add --no-cache \
        python3 \
        py3-pip \
        py3-yaml \
        py3-psycopg2 \
        py3-urllib3 \
        py3-cryptography \
        gettext \
        curl \
        bash \
    && pip install --no-cache-dir --break-system-packages "patroni[consul]==4.0.4"

COPY patroni.yml.tpl /patroni.yml.tpl
COPY post-bootstrap.sh /docker-entrypoint-initdb.d/post-bootstrap.sh
COPY post-bootstrap.sql /docker-entrypoint-initdb.d/post-bootstrap.sql
COPY entrypoint.sh /usr/local/bin/patroni-entrypoint.sh

RUN chmod +x /docker-entrypoint-initdb.d/post-bootstrap.sh /usr/local/bin/patroni-entrypoint.sh \
    && chmod 0444 /docker-entrypoint-initdb.d/post-bootstrap.sql

EXPOSE 5432 8008

ENTRYPOINT ["/usr/local/bin/patroni-entrypoint.sh"]
