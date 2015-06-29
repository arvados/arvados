# PostgreSQL Docker container for Arvados.

FROM arvados/debian:wheezy
MAINTAINER Ward Vandewege <ward@curoverse.com>

# TODO(twp): parameterize variables via autoconf or similar.
ENV POSTGRES_ROOT_PW   dummy_pw

ENV ARVADOS_DEV_DB     arvados_development
ENV ARVADOS_DEV_USER   arvados
ENV ARVADOS_DEV_PW     dummy_pw

ENV ARVADOS_TEST_DB    arvados_test
ENV ARVADOS_TEST_USER  arvados
ENV ARVADOS_TEST_PW    dummy_pw

ENV ARVADOS_PROD_DB    arvados_production
ENV ARVADOS_PROD_USER  arvados
ENV ARVADOS_PROD_PW    dummy_pw

# Install postgres and apache
RUN apt-get install -qy procps postgresql postgresql-server-dev-9.1

# Configure databases and users.
ADD postgresql.conf /etc/postgresql/9.1/main/
ADD pg_hba.conf     /etc/postgresql/9.1/main/

ADD postgresql_config.sh /tmp/postgresql_config.sh
RUN /tmp/postgresql_config.sh
RUN rm /tmp/postgresql_config.sh

# Accept database connections on port 5432 from outside the container.
EXPOSE 5432

CMD ["/bin/su", "postgres", "-c", "/usr/lib/postgresql/9.1/bin/postgres -D /var/lib/postgresql/9.1/main -c config_file=/etc/postgresql/9.1/main/postgresql.conf"]
