# Arvados API server Docker container.

FROM arvados/base
MAINTAINER Tim Pierce <twp@clinicalfuture.com>

# Install postgres and apache
RUN apt-get -q -y install procps postgresql postgresql-server-dev-9.1 apache2 \
    libcurl4-openssl-dev apache2-threaded-dev libapr1-dev libaprutil1-dev

# Configure databases and users.
ADD config_databases.sh /tmp/
RUN sh /tmp/config_databases.sh
RUN rm /tmp/config_databases.sh

# Install gems needed for API server
RUN bundle install --gemfile=/usr/src/arvados/services/api/Gemfile

# Set up the Rails database. This has to be done in a script so that
# postgres can run in the background while running rake db:setup.
ENV RAILS_ENV production
ADD database.yml /usr/src/arvados/services/api/config/database.yml
ADD config_rake.sh /tmp/config_rake.sh
RUN sh /tmp/config_rake.sh

ADD secret_token.rb /usr/src/arvados/services/api/config/initializers/secret_token.rb
ADD omniauth.rb /usr/src/arvados/services/api/config/initializers/omniauth.rb
ADD production.rb /usr/src/arvados/services/api/config/environments/production.rb

# Configure Apache.
ADD apache2_vhost /etc/apache2/sites-available/arvados
RUN a2dissite default
RUN a2ensite arvados

# Configure Passenger.
RUN passenger-install-apache2-module --auto
RUN a2enmod rewrite
ADD passenger.conf /etc/apache2/conf.d/passenger
