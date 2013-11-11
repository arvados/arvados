# Arvados API server Docker container.

FROM arvados/base
MAINTAINER Tim Pierce <twp@clinicalfuture.com>

# Install postgres and apache
RUN apt-get -q -y install procps postgresql postgresql-server-dev-9.1 apache2 \
    libcurl4-openssl-dev apache2-threaded-dev libapr1-dev libaprutil1-dev \
    supervisor

# Install gems needed for API server
RUN bundle install --gemfile=/usr/src/arvados/services/api/Gemfile

# Install generated config files
ADD generated/database.yml /usr/src/arvados/services/api/config/database.yml
ADD generated/secret_token.rb /usr/src/arvados/services/api/config/initializers/secret_token.rb
ADD generated/omniauth.rb /usr/src/arvados/services/api/config/initializers/omniauth.rb
ADD generated/production.rb /usr/src/arvados/services/api/config/environments/production.rb
ADD generated/apache2_vhost /etc/apache2/sites-available/arvados

# Configure databases and users.
ADD generated/config_databases.sh /tmp/config_databases.sh
RUN sh /tmp/config_databases.sh ; \
    rm /tmp/config_databases.sh

# Set up the Rails database. This has to be done in a script so that
# postgres can run in the background while running rake db:setup.
ENV RAILS_ENV production
ADD config_rake.sh /tmp/config_rake.sh
RUN sh /tmp/config_rake.sh ; \
    rm /tmp/config_rake.sh

# Configure Apache and Passenger.
ADD passenger.conf /etc/apache2/conf.d/passenger
RUN a2dissite default ; \
    a2ensite arvados ; \
    a2enmod rewrite ; \
    passenger-install-apache2-module --auto

# Supervisor.
ADD supervisor.conf /etc/supervisor/conf.d/arvados.conf
ADD apache2_foreground.sh /etc/apache2/foreground.sh

# Start the supervisor.
CMD ["/usr/bin/supervisord", "-n"]
