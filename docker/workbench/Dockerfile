# Arvados Workbench Docker container.

FROM arvados/passenger
MAINTAINER Ward Vandewege <ward@curoverse.com>

# Update Arvados source
RUN /bin/mkdir -p /usr/src/arvados/apps
ADD generated/workbench.tar.gz /usr/src/arvados/apps/
RUN /bin/cp /usr/src/arvados/apps/workbench/config/environments/production.rb.example /usr/src/arvados/apps/workbench/config/environments/production.rb
ADD generated/application.yml /usr/src/arvados/apps/workbench/config/application.yml

RUN /usr/local/rvm/bin/rvm-exec default bundle install --gemfile=/usr/src/arvados/apps/workbench/Gemfile && \
    touch /usr/src/arvados/apps/workbench/log/production.log && \
    chmod 666 /usr/src/arvados/apps/workbench/log/production.log && \
    touch /usr/src/arvados/apps/workbench/db/production.sqlite3 && \
    cd /usr/src/arvados/apps/workbench && \
    /usr/local/rvm/bin/rvm-exec default bundle exec rake assets:precompile && \
    chown -R www-data:www-data /usr/src/arvados/apps/workbench

# Configure Apache
ADD generated/apache2_vhost /etc/apache2/sites-available/workbench
RUN \
  a2dissite default && \
  a2ensite workbench && \
  a2enmod rewrite

ADD apache2_foreground.sh /etc/apache2/foreground.sh

# Start Apache
CMD ["/etc/apache2/foreground.sh"]

