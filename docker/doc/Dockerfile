# Arvados Documentation Docker container.

FROM arvados/base
maintainer Ward Vandewege <ward@curoverse.com>

# Install packages
RUN /bin/mkdir -p /usr/src/arvados && \
    apt-get update -qq && \
    apt-get install -qqy curl procps apache2-mpm-worker

ADD generated/doc.tar.gz /usr/src/arvados/

# Build static site
RUN /usr/local/rvm/bin/rvm-exec default bundle install --gemfile=/usr/src/arvados/doc/Gemfile && \
    /bin/sed -ri 's/^baseurl: .*$/baseurl: /' /usr/src/arvados/doc/_config.yml && \
    cd /usr/src/arvados/doc && \
    LANG="en_US.UTF-8" LC_ALL="en_US.UTF-8" /usr/local/rvm/bin/rvm-exec default bundle exec rake

# Configure Apache
ADD apache2_vhost /etc/apache2/sites-available/doc
RUN \
  a2dissite default && \
  a2ensite doc

ADD apache2_foreground.sh /etc/apache2/foreground.sh

# Start Apache
CMD ["/etc/apache2/foreground.sh"]
