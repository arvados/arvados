# Arvados API server Docker container.

FROM arvados/base
MAINTAINER Tim Pierce <twp@clinicalfuture.com>

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
RUN apt-get -q -y install procps postgresql postgresql-server-dev-9.1 apache2

# Configure databases and users.
ADD postgresql_config.sh /tmp/postgresql_config.sh
RUN /tmp/postgresql_config.sh
RUN rm /tmp/postgresql_config.sh

# Install gems needed for API server
RUN bundle install --gemfile=/usr/src/arvados/services/api/Gemfile

# Set up the Rails database. This has to be done in a script so that
# postgres can run in the background while running rake db:setup.
ENV RAILS_ENV production
ADD database.yml /usr/src/arvados/services/api/config/database.yml
ADD config_rake.sh /tmp/config_rake.sh
RUN /tmp/config_rake.sh

