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

# Configure Rails databases.
ADD generated/config_databases.sh /tmp/config_databases.sh
ADD config_rake.sh /tmp/config_rake.sh
ENV RAILS_ENV production
RUN sh /tmp/config_databases.sh ; \
    sh /tmp/config_rake.sh ; \
    rm /tmp/config_databases.sh /tmp/config_rake.sh

# Configure Apache and Passenger.
ADD passenger.conf /etc/apache2/conf.d/passenger
RUN a2dissite default ; \
    a2ensite arvados ; \
    a2enmod rewrite ; \
    a2enmod ssl ; \
    /bin/mkdir /var/run/apache2 ; \
    passenger-install-apache2-module --auto

# Supervisor.
ADD supervisor.conf /etc/supervisor/conf.d/arvados.conf
ADD apache2_foreground.sh /etc/apache2/foreground.sh

# Start the supervisor.
CMD ["/usr/bin/supervisord", "-n"]
