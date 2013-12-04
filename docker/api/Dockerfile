# Arvados API server Docker container.

FROM arvados/passenger
MAINTAINER Tim Pierce <twp@clinicalfuture.com>

# Install postgres and apache
RUN apt-get -q -y install procps postgresql postgresql-server-dev-9.1 apache2 \
                          supervisor

RUN /bin/mkdir -p /usr/src/arvados/services
ADD generated/api.tar.gz /usr/src/arvados/services/

# Install generated config files
ADD generated/database.yml /usr/src/arvados/services/api/config/database.yml
ADD generated/secret_token.rb /usr/src/arvados/services/api/config/initializers/secret_token.rb
ADD generated/omniauth.rb /usr/src/arvados/services/api/config/initializers/omniauth.rb
ADD generated/production.rb /usr/src/arvados/services/api/config/environments/production.rb
ADD generated/apache2_vhost /etc/apache2/sites-available/arvados

# Configure Rails databases.
ENV RAILS_ENV production
ADD generated/config_databases.sh /tmp/config_databases.sh
ADD generated/superuser_token /tmp/superuser_token
RUN sh /tmp/config_databases.sh && \
    rm /tmp/config_databases.sh && \
    /etc/init.d/postgresql start && \
    cd /usr/src/arvados/services/api && \
    rake db:setup && \
    rake assets:precompile && \
    ./script/create_superuser_token.rb $(cat /tmp/superuser_token) && \
    chown www-data:www-data config.ru && \
    chown www-data:www-data log -R && \
    chown www-data:www-data tmp -R

# Configure Apache and Passenger.
ADD passenger.conf /etc/apache2/conf.d/passenger
RUN a2dissite default ; \
    a2ensite arvados ; \
    a2enmod rewrite ; \
    a2enmod ssl ; \
    /bin/mkdir /var/run/apache2

# Supervisor.
ADD supervisor.conf /etc/supervisor/conf.d/arvados.conf
ADD apache2_foreground.sh /etc/apache2/foreground.sh

# Start the supervisor.
CMD ["/usr/bin/supervisord", "-n"]
