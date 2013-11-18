# Arvados API server Docker container.

FROM arvados/base
MAINTAINER Ward Vandewege <ward@clinicalfuture.com>

RUN apt-get update ;\
    apt-get install -q -y apt-utils git curl procps apache2-mpm-worker locales \
                          libcurl4-openssl-dev apache2-threaded-dev libapr1-dev libaprutil1-dev ;\
    git clone git://github.com/clinicalfuture/sso-devise-omniauth-provider.git /usr/src/sso-provider ;\
    bundle install --gemfile=/usr/src/sso-provider/Gemfile ;\
    gem install passenger -v 4.0.24 ;\
    passenger-install-apache2-module --auto

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
    chown www-data:www-data /usr/src/sso-provider/tmp_omniauth /usr/src/sso-provider/log -R ; \
    /bin/mkdir /var/run/apache2

ADD apache2_foreground.sh /etc/apache2/foreground.sh

# Start the supervisor.
CMD ["/etc/apache2/foreground.sh"]
