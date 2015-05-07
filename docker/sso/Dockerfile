# Arvados API server Docker container.

FROM arvados/passenger
MAINTAINER Ward Vandewege <ward@curoverse.com>

RUN git clone git://github.com/curoverse/sso-devise-omniauth-provider.git /usr/src/sso-provider && \
    /usr/local/rvm/bin/rvm-exec default bundle install --gemfile=/usr/src/sso-provider/Gemfile

# Install generated config files
ADD generated/secret_token.rb /usr/src/sso-provider/config/initializers/secret_token.rb
ADD generated/seeds.rb /usr/src/sso-provider/db/seeds.rb
ADD generated/application.yml /usr/src/sso-provider/config/application.yml
ADD generated/apache2_vhost /etc/apache2/sites-available/sso-provider
ADD generated/apache2_vhost /etc/apache2/sites-available/sso-provider

# Configure Apache and Passenger.
RUN a2dissite default && \
    a2ensite sso-provider && \
    a2enmod rewrite && \
    a2enmod ssl && \
    cd /usr/src/sso-provider && \
    cp config/environments/production.rb.example config/environments/production.rb && \
    RAILS_ENV=production /usr/local/rvm/bin/rvm-exec default bundle exec rake db:setup && \
    /usr/local/rvm/bin/rvm-exec default bundle exec rake assets:precompile && \
    chown www-data:www-data log config.ru -R && \
    chown www-data:www-data db db/production.sqlite3 && \
    /bin/mkdir /var/run/apache2

ADD apache2_foreground.sh /etc/apache2/foreground.sh

# Start the supervisor.
CMD ["/etc/apache2/foreground.sh"]
