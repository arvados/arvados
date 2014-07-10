# Arvados API server Docker container.

FROM arvados/passenger
MAINTAINER Tim Pierce <twp@curoverse.com>

# Install postgres and apache.
# Clone a git repository of Arvados source -- not used to build, but
# will be used by the Commit model and anything else that needs to
# check a git repo for crunch scripts.
#
RUN apt-get update && \
    apt-get -q -y install procps postgresql postgresql-server-dev-9.1 apache2 \
                          supervisor && \
    git clone --bare git://github.com/curoverse/arvados.git /var/cache/git/arvados.git

RUN /bin/mkdir -p /usr/src/arvados/services
ADD generated/api.tar.gz /usr/src/arvados/services/

# Install generated config files
ADD generated/database.yml /usr/src/arvados/services/api/config/database.yml
ADD generated/omniauth.rb /usr/src/arvados/services/api/config/initializers/omniauth.rb
RUN /bin/cp /usr/src/arvados/services/api/config/environments/production.rb.example /usr/src/arvados/services/api/config/environments/production.rb
ADD generated/application.yml /usr/src/arvados/services/api/config/application.yml
ADD generated/apache2_vhost /etc/apache2/sites-available/arvados

# Configure Rails databases.
ENV RAILS_ENV production
ADD generated/config_databases.sh /tmp/config_databases.sh
ADD generated/superuser_token /tmp/superuser_token
RUN /usr/local/rvm/bin/rvm-exec default bundle install --gemfile=/usr/src/arvados/services/api/Gemfile && \
    sh /tmp/config_databases.sh && \
    rm /tmp/config_databases.sh && \
    /etc/init.d/postgresql start && \
    cd /usr/src/arvados/services/api && \
    /usr/local/rvm/bin/rvm-exec default bundle exec rake db:setup && \
    /usr/local/rvm/bin/rvm-exec default bundle exec rake assets:precompile && \
    /usr/local/rvm/bin/rvm-exec default ./script/create_superuser_token.rb $(cat /tmp/superuser_token) && \
    chown www-data:www-data config.ru && \
    chown www-data:www-data log -R && \
    mkdir -p tmp && \
    chown www-data:www-data tmp -R

# Configure Apache and Passenger.
RUN a2dissite default && \
    a2ensite arvados && \
    a2enmod rewrite && \
    a2enmod ssl && \
    /bin/mkdir /var/run/apache2

# Supervisor.
ADD supervisor.conf /etc/supervisor/conf.d/arvados.conf
ADD ssh.sh /usr/local/bin/ssh.sh
ADD apache2_foreground.sh /etc/apache2/foreground.sh

# Start the supervisor.
CMD ["/usr/bin/supervisord", "-n"]
