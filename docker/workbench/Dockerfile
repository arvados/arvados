# Arvados Workbench Docker container.

FROM arvados/base
MAINTAINER Ward Vandewege <ward@clinicalfuture.com>

ENV DEBIAN_FRONTEND noninteractive

# Install packages
# Fix Locale
# Update Arvados source
RUN \
    apt-get update ;\
    apt-get install -q -y apt-utils git curl procps apache2-mpm-worker locales \
                          libcurl4-openssl-dev apache2-threaded-dev libapr1-dev libaprutil1-dev ;\
    curl -L https://get.rvm.io | bash -s stable --ruby=${RUBY_VERSION_NUM} ;\
    /bin/sed -ri 's/# en_US.UTF-8 UTF-8/en_US.UTF-8 UTF-8/' /etc/locale.gen ;\
    /usr/sbin/locale-gen ;\
    cd /usr/src/arvados; git pull ;\
    touch /usr/src/arvados/apps/workbench/log/production.log ;\
    chmod 666 /usr/src/arvados/apps/workbench/log/production.log ;\
    touch /usr/src/arvados/apps/workbench/db/production.sqlite3 ;\
    bundle install --gemfile=/usr/src/arvados/apps/workbench/Gemfile ;\
    cd /usr/src/arvados/apps/workbench; rake assets:precompile ;\
    passenger-install-apache2-module --auto ;\
    a2enmod rewrite

# Configure Apache
ADD apache2_vhost /etc/apache2/sites-available/workbench
RUN \
  a2dissite default ;\
  a2ensite workbench 

# Set up the production environment
ADD secret_token.rb /usr/src/arvados/apps/workbench/config/initializers/secret_token.rb
ADD production.rb /usr/src/arvados/apps/workbench/config/environments/production.rb
ADD passenger.conf /etc/apache2/conf.d/passenger

# Finally, start Apache
ENV APACHE_RUN_USER    www-data
ENV APACHE_RUN_GROUP   www-data
ENV APACHE_PID_FILE    /var/run/apache2.pid
ENV APACHE_RUN_DIR     /var/run/apache2
ENV APACHE_LOCK_DIR    /var/lock/apache2
ENV APACHE_LOG_DIR     /var/log/apache2
ENV LANG               C

CMD ["/usr/sbin/apache2", "-D", "FOREGROUND"]

