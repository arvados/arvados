# Arvados API server Docker container.

FROM arvados/passenger
MAINTAINER Ward Vandewege <ward@clinicalfuture.com>

RUN git clone git://github.com/clinicalfuture/sso-devise-omniauth-provider.git /usr/src/sso-provider ;\
    bundle install --gemfile=/usr/src/sso-provider/Gemfile

# Install generated config files
ADD generated/secret_token.rb /usr/src/sso-provider/config/initializers/secret_token.rb
ADD generated/seeds.rb /usr/src/sso-provider/db/seeds.rb
ADD generated/apache2_vhost /etc/apache2/sites-available/sso-provider
ADD generated/apache2_vhost /etc/apache2/sites-available/sso-provider

# Configure Apache and Passenger.
ADD passenger.conf /etc/apache2/conf.d/passenger
RUN a2dissite default ; \
    a2ensite sso-provider ; \
    a2enmod rewrite ; \
    a2enmod ssl ; \
    cd /usr/src/sso-provider; RAILS_ENV=production rake db:setup ; rake assets:precompile ; \
    chown www-data:www-data tmp_omniauth log config.ru -R ; \
    chown www-data:www-data db db/production.sqlite3 ; \
    /bin/mkdir /var/run/apache2

ADD apache2_foreground.sh /etc/apache2/foreground.sh

# Start the supervisor.
CMD ["/etc/apache2/foreground.sh"]
